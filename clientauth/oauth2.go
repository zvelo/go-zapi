package clientauth

import (
	"context"
	"strings"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"

	zapi "zvelo.io/go-zapi"
)

func defaultScopes() []string {
	return strings.Fields(zapi.DefaultScopes)
}

type options struct {
	scopes   []string
	tokenURL string
}

func defaults() *options {
	return &options{
		tokenURL: zapi.Endpoint.TokenURL,
	}
}

// An Option is used to configure different parts of this package
type Option func(*options)

// WithTokenURL returns an option that overrides the default TokenURL,
// zapi.Endpoint.TokenURL
func WithTokenURL(val string) Option {
	if val == "" {
		val = zapi.Endpoint.TokenURL
	}

	return func(o *options) {
		o.tokenURL = val
	}
}

// WithScope returns an option that overrides the default scopes,
// zapi.DefaultScopes
func WithScope(val ...string) Option {
	if len(val) == 0 {
		val = defaultScopes()
	}

	return func(o *options) {
		o.scopes = val
	}
}

// ClientCredentials returns a TokenSource that will retrieve client credentials
// for use with zveloAPI
func ClientCredentials(ctx context.Context, clientID, clientSecret string, opts ...Option) oauth2.TokenSource {
	o := defaults()
	for _, opt := range opts {
		opt(o)
	}

	c := clientcredentials.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		TokenURL:     o.tokenURL,
		Scopes:       o.scopes,
	}

	return c.TokenSource(ctx)
}
