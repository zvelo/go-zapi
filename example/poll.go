package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/zvelo/go-zapi"
	"github.com/zvelo/go-zapi/zapitype"
)

var pollURLs []string

func init() {
	fs := flag.NewFlagSet("poll", flag.ExitOnError)

	fs.StringVar(&zClient.Username, "username", getDefaultString("ZVELO_USERNAME", ""), "Username to obtain a token as [$ZVELO_USERNAME]")
	fs.StringVar(&zClient.Password, "password", getDefaultString("ZVELO_PASSWORD", ""), "Password to obtain a token with [$ZVELO_PASSWORD]")
	fs.StringVar(&zClient.Token, "token", getDefaultString("ZVELO_TOKEN", ""), "Token for making the query [$ZVELO_TOKEN]")
	fs.DurationVar(&zClient.PollTimeout, "timeout", zapi.DefaultPollTimeout, "timeout after this much time has elapsed")
	fs.DurationVar(&zClient.PollInterval, "interval", zapi.DefaultPollInterval, "amount of time between polling requests")

	cmd["poll"] = subcommand{
		FlagSet: fs,
		Setup:   setupPoll,
		Action:  pollURL,
		Usage:   "query url using polling",
	}
}

func setupPoll() error {
	if len(zClient.Token) == 0 &&
		(len(zClient.Username) == 0 || len(zClient.Password) == 0) {
		return fmt.Errorf("-token or -username and -password are required")
	}

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
