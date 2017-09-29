package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"sort"
	"strings"
	"text/tabwriter"
	"text/template"

	"github.com/coreos/go-oidc"
	"github.com/urfave/cli"
)

var tokenTplStr = `
{{- if .AccessToken}}Access Token:  {{.AccessToken}}
{{end}}
{{- if .RefreshToken}}Refresh Token: {{.RefreshToken}}
{{end}}
{{- if .Expiry}}Expires At:    {{.Expiry}}
{{end -}}
`

type oidcToken struct {
	*oidc.IDToken
	Claims map[string]interface{}
}

var idTokenTplStr = `Issuer:        {{.Issuer}}
Audience:      {{join .Audience}}
Subject:       {{.Subject}}
Issued At:     {{.IssuedAt}}
Claims:
  {{claims .Claims}}
`

var (
	verifier   *oidc.IDTokenVerifier
	tokenTpl   = template.Must(template.New("token").Parse(tokenTplStr))
	idTokenTpl = template.Must(template.New("id_token").
			Funcs(template.FuncMap{
			"join": func(i []string) string {
				return strings.Join(i, ", ")
			},
			"claims": func(i map[string]interface{}) string {
				var buf bytes.Buffer
				w := tabwriter.NewWriter(&buf, 0, 0, 0, ' ', 0)
				var keys []string
				for k := range i {
					keys = append(keys, k)
				}

				sort.Strings(keys)

				for _, k := range keys {
					v := i[k]
					if v == nil || v == "" {
						continue
					}
					switch k {
					case "iat", "exp", "aud", "sub", "iss", "auth_time":
						continue
					}
					fmt.Fprintf(w, "  %s: \t%v\n", k, v)
				}
				_ = w.Flush()
				return strings.TrimSpace(buf.String())
			},
		}).
		Parse(idTokenTplStr))
)

func init() {
	cmd := cli.Command{
		Name:   "token",
		Usage:  "retrieve a token for use elsewhere",
		Before: tokenSetup,
		Action: token,
	}
	cmd.BashComplete = bashCommandComplete(cmd)
	app.Commands = append(app.Commands, cmd)
}

func setupVerifier() error {
	if clientID == "" {
		return nil
	}

	for _, s := range scopes {
		if s != "openid" {
			continue
		}

		provider, err := oidc.NewProvider(context.Background(), "https://auth.zvelo.com")
		if err != nil {
			return err
		}

		verifier = provider.Verifier(&oidc.Config{ClientID: clientID})
		break
	}

	return nil
}

func tokenSetup(_ *cli.Context) error {
	return setupVerifier()
}

func token(_ *cli.Context) error {
	if tokenSource == nil {
		return nil
	}

	token, err := tokenSource.Token()
	if err != nil {
		return err
	}

	if err = tokenTpl.Execute(os.Stdout, token); err != nil {
		return err
	}

	rawIDToken, ok := token.Extra("id_token").(string)

	if verifier == nil || !ok {
		return nil
	}

	var ot oidcToken

	if ot.IDToken, err = verifier.Verify(context.Background(), rawIDToken); err != nil {
		return err
	}

	if err = ot.IDToken.Claims(&ot.Claims); err != nil {
		return err
	}

	return idTokenTpl.Execute(os.Stdout, ot)
}
