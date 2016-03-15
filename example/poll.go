package main

import (
	"flag"
	"fmt"
	"os"

	"zvelo.io/msg/go-msg"
)

var pollURLs []string

func init() {
	fs := flag.NewFlagSet("poll", flag.ExitOnError)
	fs.Usage = cmdUsage(fs, "url [url...]")

	cmd["poll"] = subcommand{
		FlagSet: fs,
		Setup:   setupPoll,
		Action:  pollURL,
		Usage:   "query url using polling",
	}
}

func setupPoll() error {
	pollURLs = cmd["poll"].FlagSet.Args()

	if len(pollURLs) == 0 {
		return fmt.Errorf("at least one url is required")
	}

	return nil
}

func pollURL() error {
	reply, err := zClient.Query(&msg.QueryURLRequests{
		Url: pollURLs,
		Dataset: []msg.DataSetType{
			msg.DataSetType_CATEGORIZATION,
			msg.DataSetType_ADFRAUD,
		},
	})
	if err != nil {
		return err
	}

	// TODO(jrubin) assert(len(reply.RequestIDs) > 0)

	errCh := make(chan error)

	// TODO(jrubin) only poll for the first reqid?
	resultCh := zClient.Poll(reply.RequestId[0], errCh)

	for {
		select {
		case err := <-errCh:
			fmt.Println(err) // TODO(jrubin)
		case result, ok := <-resultCh:
			if !ok {
				fmt.Fprintf(os.Stderr, "timeout\n")
				return nil
			}

			fmt.Println(result) // TODO(jrubin)
			return nil
		}
	}
}
