package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/zvelo/go-zapi/zapitype"
)

var pollURLs []string

func init() {
	fs := flag.NewFlagSet("poll", flag.ExitOnError)

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
	reply, err := zClient.Query(&zapitype.QueryURLRequests{
		URLs: pollURLs,
		DataSets: []zapitype.DataSetType{
			zapitype.DataSetTypeCategorization,
			zapitype.DataSetTypeAdFraud,
		},
	})
	if err != nil {
		return err
	}

	// TODO(jrubin) assert(len(reply.RequestIDs) == 1)

	errCh := make(chan error)
	resultCh := zClient.Poll(reply.RequestIDs[0], errCh)

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
