package main

import (
	"context"
	"fmt"
	htmlTemplate "html/template"
	"net/http"
	"os"
	textTemplate "text/template"

	"github.com/pkg/browser"
	"github.com/pkg/errors"
	"github.com/urfave/cli"

	"golang.org/x/oauth2"
)

var tokenHTMLTplStr = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <title>zapi Token</title>
</head>
<body>
<ul>
  {{if .AccessToken}}<li>Access Token: <code>{{.AccessToken}}</code></li>{{end}}
  {{if .RefreshToken}}<li>Refresh Token: <code>{{.RefreshToken}}</code></li>{{end}}
  {{if .Expiry}}<li>Expires in: <code>{{.Expiry}}</code></li>{{end}}
  {{if .IDToken}}<li>ID Token: <code>{{.IDToken}}</code></li>{{end}}
</ul>
</body>
</html>
`

var tokenTextTplStr = `
{{- if .AccessToken}}Access Token: {{.AccessToken}}
{{end}}
{{- if .RefreshToken}}Refresh Token: {{.RefreshToken}}
{{end}}
{{- if .Expiry}}Expires in: {{.Expiry}}
{{end}}
{{- if .IDToken}}ID Token: {{.IDToken}}
{{end -}}
`

var (
	tokenHTMLTpl = htmlTemplate.Must(htmlTemplate.New("token").Parse(tokenHTMLTplStr))
	tokenTextTpl = textTemplate.Must(textTemplate.New("token").Parse(tokenTextTplStr))
)

type tokenTplCtx struct {
	*oauth2.Token
	IDToken interface{}
}

func init() {
	app.Commands = append(app.Commands, cli.Command{
		Name:   "token",
		Usage:  "retrieve a token for use elsewhere",
		Action: token,
	})
}

func token(_ *cli.Context) error {
	token, err := tokenSource.Token()
	if err != nil {
		return err
	}

	return tokenTextTpl.Execute(os.Stdout, tokenTplCtx{token, token.Extra("id_token")})
}

type userAccreditor struct {
	*oauth2.Config
	Addr   string
	NoOpen bool

	state    string
	token    *oauth2.Token
	tokenErr error
}

var _ oauth2.TokenSource = (*userAccreditor)(nil)

func (a *userAccreditor) Token() (*oauth2.Token, error) {
	a.state = randString(32)

	u := a.AuthCodeURL(a.state)

	if !a.NoOpen {
		fmt.Fprintf(os.Stderr, "opening in browser: %s\n", u)
		if err := browser.OpenURL(u); err != nil {
			return nil, err
		}
	} else {
		fmt.Fprintf(os.Stderr, "open this url in your browser: %s\n", u)
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)

	mux := http.NewServeMux()
	mux.Handle("/", a.handler(ctx, cancel))

	server := http.Server{
		Addr:    a.Addr,
		Handler: mux,
	}

	go func() {
		<-ctx.Done()
		_ = server.Shutdown(ctx)
	}()

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return nil, err
	}

	return a.token, a.tokenErr
}

func (a *userAccreditor) handler(ctx context.Context, done func()) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer done()

		if a.state == "" || a.state != r.URL.Query().Get("state") {
			http.Error(w, "invalid state", http.StatusUnauthorized)
			return
		}

		errCode := r.URL.Query().Get("error")

		if errCode != "" {
			a.tokenErr = errors.Errorf("%s: %s", errCode, r.URL.Query().Get("error_description"))
		}

		switch errCode {
		case "access_denied", "unauthorized_client":
			http.Error(w, a.tokenErr.Error(), http.StatusUnauthorized)
			return
		case "invalid_request":
			http.Error(w, a.tokenErr.Error(), http.StatusBadRequest)
			return
		case "unsupported_response_type", "invalid_scope":
			http.Error(w, a.tokenErr.Error(), http.StatusInternalServerError)
			return
		case "server_error", "temporarily_unavailable":
			http.Error(w, a.tokenErr.Error(), http.StatusServiceUnavailable)
			return
		}

		var err error
		a.token, err = a.Exchange(ctx, r.URL.Query().Get("code"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		_ = tokenHTMLTpl.Execute(w, tokenTplCtx{a.token, a.token.Extra("id_token")})
	})
}
