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

type TokenSourcer interface {
	TokenSource(context.Context) oauth2.TokenSource
}

func ClientCredentials(clientID, clientSecret string, scopes ...string) TokenSourcer {
	if len(scopes) == 0 {
		scopes = defaultScopes()
	}

	return &clientcredentials.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		TokenURL:     Endpoint.TokenURL,
		Scopes:       scopes,
	}
}

var Endpoint = oauth2.Endpoint{
	AuthURL:  "https://auth.zvelo.com/oauth2/auth",
	TokenURL: "https://auth.zvelo.com/oauth2/token",
}

type tokenSourcer struct {
	*oauth2.Config
}

func (c *tokenSourcer) TokenSource(ctx context.Context) oauth2.TokenSource {
	return c.Config.TokenSource(ctx, nil)
}

func Credentials(clientID, clientSecret, redirectURL string, scopes ...string) TokenSourcer {
	if len(scopes) == 0 {
		scopes = defaultScopes()
	}

	return &tokenSourcer{&oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Endpoint:     Endpoint,
		RedirectURL:  redirectURL,
		Scopes:       scopes,
	}}
}
