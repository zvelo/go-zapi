package main

import (
	"context"
	"fmt"
	"os"

	"github.com/urfave/cli"

	"zvelo.io/msg/mock"
)

var mockAddr string

func init() {
	cmd := cli.Command{
		Name:   "mock",
		Usage:  "start a mock zveloAPI server",
		Action: startMock,
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:        "addr",
				EnvVar:      "ZVELO_MOCK_ADDR",
				Usage:       "address:port for the mock server to listen on ",
				Value:       ":https",
				Destination: &mockAddr,
			},
		},
	}
	cmd.BashComplete = bashCommandComplete(cmd)
	app.Commands = append(app.Commands, cmd)
}

func startMock(_ *cli.Context) error {
	fmt.Fprintf(os.Stderr, "mock zveloAPI server listening at: %s\n", mockAddr)
	return mock.ListenAndServeTLS(context.Background(), mockAddr)
}
