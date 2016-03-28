package main

import (
	"encoding/base64"
	"flag"
	"fmt"
	"os"

	"zvelo.io/msg/go-msg"
)

var requestID []byte
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
	str := cmd["poll"].FlagSet.Arg(0)

	if len(str) == 0 {
		return fmt.Errorf("request_id is required")
	}

	var err error
	requestID, err = base64.StdEncoding.DecodeString(str)

	return err
}

func pollURL() error {
	dsts := []msg.DataSetType{
		// TODO(jrubin) get datasets from cmdline
		msg.DataSetType_CATEGORIZATION,
		msg.DataSetType_ADFRAUD,
	}

	if pollOnce {
		result, err := zClient.PollOnce(requestID, dsts)
		if err != nil {
			return err
		}

		return handlePollResult(result)
	}

	errCh := make(chan error)
	resultCh := zClient.Poll(requestID, dsts, errCh)

	for {
		select {
		case err := <-errCh:
			fmt.Println(err) // TODO(jrubin)
		case result, ok := <-resultCh:
			if !ok {
				fmt.Fprintf(os.Stderr, "timeout\n")
				return nil
			}

			return handlePollResult(result)
		}
	}
}

func handlePollResult(result *msg.QueryResult) error {
	fmt.Println(result) // TODO(jrubin)
	return nil
}
