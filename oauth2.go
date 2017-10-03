package zapi

import "golang.org/x/oauth2"

// DefaultScopes are the default scopes used when no scopes are explicietly
// defined
const DefaultScopes = "zvelo.dataset"

// Endpoint is an oauth2.Endpoint that is useful for working with the oauth2
// package
var Endpoint = oauth2.Endpoint{
	AuthURL:  "https://auth.zvelo.com/oauth2/auth",
	TokenURL: "https://auth.zvelo.com/oauth2/token",
}
