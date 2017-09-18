package userauth

import (
	"context"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"strings"
	"sync"

	zapi "zvelo.io/go-zapi"
	"zvelo.io/go-zapi/internal/zvelo"

	"github.com/pkg/browser"
	"github.com/pkg/errors"

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

var tokenHTMLTpl = template.Must(template.New("token").Parse(tokenHTMLTplStr))

type userAccreditor struct {
	sync.Mutex
	oauth2.Config
	addr  string
	open  bool
	ctx   context.Context
	debug bool
}

var _ oauth2.TokenSource = (*userAccreditor)(nil)

// DefaultCallbackAddr is the default address:host on which a local http.Server
// will be started. It can be overridden using WithCallbackAddr.
const DefaultCallbackAddr = ":4445"

// DefaultRedirectURL is the default RedirectURL that the oauth2 flow will use.
// It can be overridden using WithRedirectURL.
const DefaultRedirectURL = "http://localhost:4445/callback"

func defaultScopes() []string {
	return strings.Fields(zapi.DefaultScopes)
}

// An Option is used to configure the oauth2 user credential flow.
type Option func(*userAccreditor)

// WithRedirectURL returns an Option that specifies the RedirectURL that will be
// used by the oauth2 flow. The client must be configured on the oauth2 server
// to permit this value or else the flow will fail.
func WithRedirectURL(val string) Option {
	if val == "" {
		val = DefaultRedirectURL
	}

	return func(a *userAccreditor) {
		a.Config.RedirectURL = val
	}
}

// WithScope returns an Option that specifies the scopes that will be requested
// for the token. If not specified, zapi.DefaultScopes will be used.
func WithScope(val ...string) Option {
	if len(val) == 0 {
		val = defaultScopes()
	}

	return func(a *userAccreditor) {
		a.Config.Scopes = val
	}
}

// WithCallbackAddr returns an Option that specifies the address:host on which a
// local http.Server will be configured to listen for the redirect. It should be
// reachable by going to the RedirectURL.
func WithCallbackAddr(val string) Option {
	if val == "" {
		val = DefaultCallbackAddr
	}

	return func(a *userAccreditor) {
		a.addr = val
	}
}

// WithoutOpen returns an Option that will prevent a browser window from being
// opened automatically and will instead direct the user to open the url emitted
// on stderr.
func WithoutOpen() Option {
	return func(a *userAccreditor) {
		a.open = false
	}
}

// WithDebug returns an option that causes incoming http.Requests to the
// callback server to be logged to stderr.
func WithDebug() Option {
	return func(a *userAccreditor) {
		a.debug = true
	}
}

func defaults(ctx context.Context, clientID, clientSecret string) *userAccreditor {
	return &userAccreditor{
		Config: oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			Endpoint:     zapi.Endpoint,
			RedirectURL:  DefaultRedirectURL,
			Scopes:       defaultScopes(),
		},
		addr: DefaultCallbackAddr,
		ctx:  ctx,
		open: true,
	}
}

// TokenSource returns an oauth2.TokenSource that will provide tokens using the
// three legged oauth2 flow.
func TokenSource(ctx context.Context, clientID, clientSecret string, opts ...Option) oauth2.TokenSource {
	a := defaults(ctx, clientID, clientSecret)
	for _, opt := range opts {
		opt(a)
	}
	return a
}

func (a *userAccreditor) Token() (*oauth2.Token, error) {
	a.Lock()
	defer a.Unlock()

	state := zvelo.RandString(32)

	u := a.AuthCodeURL(state)

	if a.open {
		fmt.Fprintf(os.Stderr, "opening in browser: %s\n", u)
		if err := browser.OpenURL(u); err != nil {
			return nil, err
		}
	} else {
		fmt.Fprintf(os.Stderr, "open this url in your browser: %s\n", u)
	}

	ctx, cancel := context.WithCancel(a.ctx)
	defer cancel()

	ch := make(chan result)

	mux := http.NewServeMux()
	mux.Handle("/", a.handler(ctx, state, ch))

	server := http.Server{
		Addr:    a.addr,
		Handler: mux,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Fprintf(os.Stderr, "callback listener error: %s\n", err)
		}
	}()

	defer func() { _ = server.Shutdown(ctx) }()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case res := <-ch:
		cancel()
		return res.token, res.err
	}
}

type result struct {
	token *oauth2.Token
	err   error
}

func (a *userAccreditor) handler(ctx context.Context, state string, ch chan<- result) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if a.debug {
			zvelo.DebugRequest(r)
		}

		if state == "" || state != r.URL.Query().Get("state") {
			// don't return the result on the channel, this can happen when the
			// browser requests a favicon
			http.Error(w, "invalid state", http.StatusUnauthorized)
			return
		}

		var res result
		defer func() { ch <- res }()

		errCode := r.URL.Query().Get("error")

		if errCode != "" {
			res.err = errors.Errorf("%s: %s", errCode, r.URL.Query().Get("error_description"))
		}

		switch errCode {
		case "access_denied", "unauthorized_client":
			http.Error(w, res.err.Error(), http.StatusUnauthorized)
			return
		case "invalid_request":
			http.Error(w, res.err.Error(), http.StatusBadRequest)
			return
		case "unsupported_response_type", "invalid_scope":
			http.Error(w, res.err.Error(), http.StatusInternalServerError)
			return
		case "server_error", "temporarily_unavailable":
			http.Error(w, res.err.Error(), http.StatusServiceUnavailable)
			return
		}

		res.token, res.err = a.Exchange(ctx, r.URL.Query().Get("code"))
		if res.err != nil {
			http.Error(w, res.err.Error(), http.StatusBadRequest)
			return
		}

		_ = tokenHTMLTpl.Execute(w, struct {
			*oauth2.Token
			IDToken interface{}
		}{res.token, res.token.Extra("id_token")})
	})
}
