package zapi

import (
	"context"
	"strings"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
)

// DefaultScopes are the default scopes used when no scopes are explicietly
// defined
const DefaultScopes = "zvelo.dataset"

func defaultScopes() []string {
	return strings.Fields(DefaultScopes)
}

// Endpoint is an oauth2.Endpoint that is useful for working with the oauth2
// package
var Endpoint = oauth2.Endpoint{
	AuthURL:  "https://auth.zvelo.com/oauth2/auth",
	TokenURL: "https://auth.zvelo.com/oauth2/token",
}

// ClientCredentials returns a TokenSource that will retrieve client credentials
// for use with zveloAPI
func ClientCredentials(ctx context.Context, clientID, clientSecret string, scopes ...string) oauth2.TokenSource {
	if len(scopes) == 0 {
		scopes = defaultScopes()
	}

	c := clientcredentials.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		TokenURL:     Endpoint.TokenURL,
		Scopes:       scopes,
	}

	return c.TokenSource(ctx)
}
