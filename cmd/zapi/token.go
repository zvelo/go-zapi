package main

import (
	"os"
	"text/template"

	"github.com/urfave/cli"

	"golang.org/x/oauth2"
)

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

var tokenTextTpl = template.Must(template.New("token").Parse(tokenTextTplStr))

func init() {
	cmd := cli.Command{
		Name:   "token",
		Usage:  "retrieve a token for use elsewhere",
		Action: token,
	}
	cmd.BashComplete = BashCommandComplete(cmd)
	app.Commands = append(app.Commands, cmd)
}

func token(_ *cli.Context) error {
	token, err := tokenSource.Token()
	if err != nil {
		return err
	}

	return tokenTextTpl.Execute(os.Stdout, struct {
		*oauth2.Token
		IDToken interface{}
	}{token, token.Extra("id_token")})
}
