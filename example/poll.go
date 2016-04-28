package main

import (
	"flag"
	"fmt"
	"os"

	"zvelo.io/msg"
)

var requestID string
var pollOnce bool

func init() {
	fs := flag.NewFlagSet("poll", flag.ExitOnError)
	fs.Usage = cmdUsage(fs, "request_id")

	fs.BoolVar(&pollOnce, "once", getDefaultBool("POLL_ONCE"), "make just a single poll request [$ZVELO_POLL_ONCE]")

	cmd["poll"] = subcommand{
		FlagSet: fs,
		Setup:   setupPoll,
		Action:  pollURL,
		Usage:   "poll for results with a request_id",
	}
}

func setupPoll() error {
	requestID = cmd["poll"].FlagSet.Arg(0)

	if len(requestID) == 0 {
		return fmt.Errorf("request_id is required")
	}

	return nil
}

func pollURL() error {
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
