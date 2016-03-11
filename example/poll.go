package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/zvelo/go-zapi/zapitype"
)

var pollConfig = struct {
	URL     string
	Timeout time.Duration
}{}

func init() {
	fs := flag.NewFlagSet("poll", flag.ExitOnError)

	fs.StringVar(&zClient.Username, "username", getDefaultString("ZVELO_USERNAME", ""), "Username to obtain a token as [$ZVELO_USERNAME]")
	fs.StringVar(&zClient.Password, "password", getDefaultString("ZVELO_PASSWORD", ""), "Password to obtain a token with [$ZVELO_PASSWORD]")
	fs.StringVar(&zClient.Token, "token", getDefaultString("ZVELO_TOKEN", ""), "Token for making the query [$ZVELO_TOKEN]")
	fs.StringVar(&pollConfig.URL, "url", "", "URL to query")
	fs.DurationVar(&pollConfig.Timeout, "timeout", 15*time.Minute, "timeout after this much time has elapsed")

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

	if len(pollConfig.URL) == 0 {
		return fmt.Errorf("-url is required")
	}

	return nil
}

func pollURL() error {
	reply, err := zClient.Query(&zapitype.QueryURLRequests{
		URLs: []string{pollConfig.URL},
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
	resultCh := zClient.Poll(reply.RequestIDs[0], 15*time.Second, errCh)
	timeoutCh := time.After(pollConfig.Timeout)

	for {
		select {
		case err := <-errCh:
			fmt.Println(err) // TODO(jrubin)
		case result := <-resultCh:
			fmt.Println(result) // TODO(jrubin)
			return nil
		case <-timeoutCh:
			fmt.Fprintf(os.Stderr, "timeout\n")
			return nil
		}
	}
}
