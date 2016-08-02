package main

import (
	"fmt"
	"os"

	"github.com/urfave/cli"
	"zvelo.io/msg"
)

var requestID string
var pollOnce bool

func init() {
	app.Commands = append(app.Commands, cli.Command{
		Name:   "poll",
		Usage:  "poll for results with a request-id",
		Action: pollURL,
		Before: setupPoll,
		Flags: []cli.Flag{
			cli.BoolFlag{
				Name:   "once, o",
				EnvVar: "ZVELO_POLL_ONCE",
				Usage:  "make just a single poll request",
			},
			cli.StringFlag{
				Name:  "request-id, rid",
				Usage: "request id to poll for",
			},
		},
	})
}

func setupPoll(c *cli.Context) error {
	if err := setupClient(c); err != nil {
		return err
	}

	requestID = c.String("request-id")
	pollOnce = c.Bool("once")

	if len(requestID) == 0 {
		return fmt.Errorf("request_id is required")
	}

	return nil
}

func pollURL(c *cli.Context) error {
	if pollOnce {
		result, err := zClient.PollOnce(requestID)
		if err != nil {
			return err
		}

		return result.Pretty(os.Stdout)
	}

	errCh := make(chan error)
	resultCh := make(chan *msg.QueryResult)

	zClient.Poll(requestID, resultCh, errCh)

	for {
		select {
		case err := <-errCh:
			fmt.Fprintf(os.Stderr, "%s\n", err)
		case result, ok := <-resultCh:
			if !ok {
				fmt.Fprintf(os.Stderr, "timeout\n")
				return nil
			}

			return result.Pretty(os.Stdout)
		}
	}
}
