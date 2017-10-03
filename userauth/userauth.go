package userauth

import (
	"context"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
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
	addr               string
	open               bool
	authCodeURLHandler AuthCodeURLHandler
	ctx                context.Context
	debug              io.Writer
	ignoreErrors       bool
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

// WithIgnoreErrors returns an Option that prevents redirect uris from the
// server containing errors from stopping the listener server. This should only
// be used for testing.
func WithIgnoreErrors() Option {
	return func(a *userAccreditor) {
		a.ignoreErrors = true
	}
}

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

// WithEndpoint returns an Option that specifies the oauth2 endpoint that will
// be used to get tokens. By default it uses zapi.Endpoint.
func WithEndpoint(val oauth2.Endpoint) Option {
	if val == (oauth2.Endpoint{}) {
		val = zapi.Endpoint
	}

	return func(a *userAccreditor) {
		a.Config.Endpoint = val
	}
}

// An AuthCodeURLHandler is asynchronously passed the AuthCodeURL when Token()
// is called
type AuthCodeURLHandler interface {
	AuthCodeURL(string)
}

// The AuthCodeURLHandlerFunc type is an adapter to allow the use of ordinary
// functions as AuthCodeURLHandlers. If f is a function with the appropriate
// signature, AuthCodeURLHandlerFunc(f) is a Handler that calls f.
type AuthCodeURLHandlerFunc func(string)

// AuthCodeURL calls f(u)
func (f AuthCodeURLHandlerFunc) AuthCodeURL(u string) {
	f(u)
}

var _ AuthCodeURLHandler = (*AuthCodeURLHandlerFunc)(nil)

// WithAuthCodeURLHandler returns an Option that will cause the passed in
// handler to be called in a new goroutine with the value of the URL that should
// be opened. This allows programmatic user authentication (e.g. without
// interacting with the browser). Overrides WithoutOpen().
func WithAuthCodeURLHandler(val AuthCodeURLHandler) Option {
	return func(a *userAccreditor) {
		a.authCodeURLHandler = val
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
// callback server to be logged to the writer.
func WithDebug(val io.Writer) Option {
	if val == nil {
		val = ioutil.Discard
	}

	return func(a *userAccreditor) {
		a.debug = val
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
		addr:  DefaultCallbackAddr,
		ctx:   ctx,
		open:  true,
		debug: ioutil.Discard,
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

	if a.authCodeURLHandler != nil {
		go a.authCodeURLHandler.AuthCodeURL(u)
	} else if a.open {
		fmt.Fprintf(os.Stderr, "opening in browser: %s\n", u)
		if err := browser.OpenURL(u); err != nil {
			return nil, err
		}
	} else {
		fmt.Fprintf(os.Stderr, "open this url in your browser: %s\n", u)
	}

	ch := make(chan result)

	mux := http.NewServeMux()
	mux.Handle("/", a.handler(state, ch))

	server := http.Server{
		Addr:    a.addr,
		Handler: mux,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Fprintf(os.Stderr, "callback listener error: %s\n", err)
		}
	}()

	defer func() { _ = server.Shutdown(a.ctx) }()

	select {
	case <-a.ctx.Done():
		return nil, a.ctx.Err()
	case res := <-ch:
		return res.token, res.err
	}
}

type result struct {
	token *oauth2.Token
	err   error
}

func (a *userAccreditor) handler(state string, ch chan<- result) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		zvelo.DebugRequest(a.debug, r)

		if state == "" || state != r.URL.Query().Get("state") {
			// don't return the result on the channel, this can happen when the
			// browser requests a favicon
			http.Error(w, "invalid state", http.StatusUnauthorized)
			return
		}

		var queryErr error
		var res result

		defer func() {
			if res.err == nil && queryErr != nil {
				if a.ignoreErrors {
					return
				}

				res.err = queryErr
			}

			ch <- res
		}()

		errCode := r.URL.Query().Get("error")

		if errCode != "" {
			queryErr = errors.Errorf("%s: %s", errCode, r.URL.Query().Get("error_description"))
		}

		switch errCode {
		case "access_denied", "unauthorized_client":
			http.Error(w, queryErr.Error(), http.StatusUnauthorized)
			return
		case "invalid_request":
			http.Error(w, queryErr.Error(), http.StatusBadRequest)
			return
		case "unsupported_response_type", "invalid_scope":
			http.Error(w, queryErr.Error(), http.StatusInternalServerError)
			return
		case "server_error", "temporarily_unavailable":
			http.Error(w, queryErr.Error(), http.StatusServiceUnavailable)
			return
		}

		res.token, res.err = a.Exchange(a.ctx, r.URL.Query().Get("code"))
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
