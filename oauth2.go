package zapi

import (
	"context"
	"strings"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
)

const DefaultScopes = "zvelo.dataset"

func defaultScopes() []string {
	return strings.Fields(DefaultScopes)
}

var Endpoint = oauth2.Endpoint{
	AuthURL:  "https://auth.zvelo.com/oauth2/auth",
	TokenURL: "https://auth.zvelo.com/oauth2/token",
}

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
