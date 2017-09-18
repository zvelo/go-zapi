package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/pkg/errors"
	"github.com/urfave/cli"
)

const completionFlagName = "compgen"

func init() {
	cli.BashCompletionFlag = cli.BoolFlag{
		Name:   completionFlagName,
		Hidden: true,
	}

	app.Commands = append(app.Commands, cli.Command{
		Name:        "complete",
		Usage:       "generate autocomplete script",
		Description: "eval \"$(" + name + " complete)\"",
		Before:      co.setup,
		Action:      co.run,
	})
}

func bashComplete(c *cli.Context) {
	complete(c, c.App.Commands, c.App.Flags)
}

func bashCommandComplete(cmd cli.Command) cli.BashCompleteFunc {
	return func(c *cli.Context) {
		complete(c, cmd.Subcommands, cmd.Flags)
	}
}

func complete(c *cli.Context, cmds []cli.Command, flags []cli.Flag) {
	for _, command := range cmds {
		if command.Hidden {
			continue
		}

		for _, name := range command.Names() {
			fmt.Println(name)
		}
	}

	for _, flag := range flags {
		for _, name := range strings.Split(flag.GetName(), ",") {
			if name == completionFlagName {
				continue
			}

			switch name = strings.TrimSpace(name); len(name) {
			case 0:
			case 1:
				fmt.Println("-" + name)
			default:
				fmt.Println("--" + name)
			}
		}
	}
}

type shell int

const (
	bash shell = iota
	zsh
)

type cc struct {
	AppName  string
	FlagName string
	Shell    shell
	Bash     shell
	Zsh      shell
}

var co = cc{
	AppName:  name,
	FlagName: completionFlagName,
	Bash:     bash,
	Zsh:      zsh,
}

func (co *cc) setup(c *cli.Context) error {
	switch shell := filepath.Base(os.Getenv("SHELL")); shell {
	case "bash":
		co.Shell = bash
	case "zsh":
		co.Shell = zsh
	default:
		return errors.Errorf("unsupported shell: %s", shell)
	}
	return nil
}

func (co *cc) run(c *cli.Context) error {
	return completeTpl.Execute(os.Stdout, co)
}

var completeTpl = template.Must(template.New("shellFunc").Parse(shellFunc))

var shellFunc = `{{ if eq .Shell .Bash }}#!/bin/bash{{ end }}{{ if eq .Shell .Zsh }}autoload -U compinit && compinit
autoload -U bashcompinit && bashcompinit{{ end }}

_{{ .AppName }}_autocomplete() {
     local cur opts base
     COMPREPLY=()
     cur="${COMP_WORDS[COMP_CWORD]}"
     opts=$( ${COMP_WORDS[@]:0:$COMP_CWORD} --{{ .FlagName }} )
     COMPREPLY=( $(compgen -W "${opts}" -- ${cur}) )
     return 0
 }

 complete -F _{{ .AppName }}_autocomplete {{ .AppName }}
`
