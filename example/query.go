package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"zvelo.io/msg"
)

const (
	callbackDefaultListenAddress = "[::1]:8080"
	callbackDefaultTimeout       = 15 * time.Minute
)

type callbackHandler struct {
	config struct {
		URLs                       []string
		ListenAddress, CallbackURL string
		Timeout                    time.Duration
		PartialResults             bool
		Poll                       bool
	}
	doneCh chan *msg.QueryResult
	errCh  chan error
}

var cbHandler callbackHandler

func init() {
	fs := flag.NewFlagSet("query", flag.ExitOnError)
	fs.Usage = cmdUsage(fs, "url [url...]")

	fs.BoolVar(&cbHandler.config.Poll, "poll", getDefaultBool("ZVELO_QUERY_POLL"), "poll for results [$ZVELO_QUERY_POLL]")

	fs.StringVar(
		&cbHandler.config.ListenAddress,
		"listen-address",
		getDefaultString("ZVELO_QUERY_LISTEN_ADDRESS", callbackDefaultListenAddress),
		"address and port to listen for callbacks on [$ZVELO_QUERY_LISTEN_ADDRESS]",
	)

	fs.StringVar(
		&cbHandler.config.CallbackURL,
		"callback-url",
		getDefaultString("ZVELO_QUERY_CALLBACK_URL", ""),
		"publicly accessible base URL that routes to the address used by the -listen-address flag [$ZVELO_QUERY_CALLBACK_URL]",
	)

	fs.DurationVar(
		&cbHandler.config.Timeout,
		"timeout",
		callbackDefaultTimeout, // TODO(jrubin) use getDefaultDuration and $ZVELO_QUERY_TIMEOUT
		"maximum amount of time to wait for the callback to be called",
	)

	fs.BoolVar(
		&cbHandler.config.PartialResults,
		"partial-results",
		false, // TODO(jrubin) use getDefaultBool and $ZVELO_QUERY_PARTIAL_RESULTS
		"request that datasets be delivered as soon as they become available instead of waiting for all datasets to become available before responding",
	)

	cmd["query"] = subcommand{
		FlagSet: fs,
		Setup:   setupQuery,
		Action:  queryURL,
		Usage:   "query for url",
	}
}

func setupQuery() error {
	if cbHandler.config.Poll && len(cbHandler.config.CallbackURL) > 0 {
		return fmt.Errorf("poll and callback can't both be true")
	}

	if len(cbHandler.config.CallbackURL) > 0 {
		cbHandler.errCh = make(chan error, 1)
		cbHandler.doneCh = make(chan *msg.QueryResult, 1)
		go func() { _ = http.ListenAndServe(cbHandler.config.ListenAddress, cbHandler) }()
	}

	cbHandler.config.URLs = cmd["query"].FlagSet.Args()

	if len(cbHandler.config.URLs) == 0 {
		return fmt.Errorf("at least one url is required")
	}

	return nil
}

func queryURL() error {
	req := &msg.QueryURLRequests{
		Url:     cbHandler.config.URLs,
		Dataset: datasets,
	}

	if len(cbHandler.config.CallbackURL) > 0 {
		req.Callback = cbHandler.config.CallbackURL
		req.PartialResults = cbHandler.config.PartialResults
		// TODO(jrubin) add support for Accept field in msg.QueryURLRequests
	} else {
		req.Poll = true
	}

	reply, err := zClient.Query(req)
	if err != nil {
		return err
	}

	if cbHandler.config.Poll {
		return pollForResults(reply)
	}

	if len(cbHandler.config.CallbackURL) > 0 {
		waitForCallback(reply)
		return nil
	}

	fmt.Printf("Request ID(s): %s\n", strings.Join(reply.RequestId, ", "))
	return nil
}

func pollForResults(reply *msg.QueryReply) error {
	errCh := make(chan error)
	resultCh := make(chan *msg.QueryResult)

	for _, reqID := range reply.RequestId {
		zClient.Poll(reqID, resultCh, errCh)
	}

	for i := 0; i < len(reply.RequestId); {
		select {
		case err := <-errCh:
			fmt.Fprintf(os.Stderr, "%s\n", err)
		case result, ok := <-resultCh:
			i++

			if !ok {
				fmt.Fprintf(os.Stderr, "timeout\n")
				return nil
			}

			return result.Pretty(os.Stdout)
		}
	}

	return nil
}

func waitForCallback(reply *msg.QueryReply) {
	for i := 0; i < len(reply.RequestId); {
		select {
		case err := <-cbHandler.errCh:
			fmt.Fprintf(os.Stderr, "%s\n", err)
		case result := <-cbHandler.doneCh:
			i++
			if err := result.Pretty(os.Stdout); err != nil {
				fmt.Fprintf(os.Stderr, "%s\n", err)
			}
		case <-time.After(cbHandler.config.Timeout):
			i++
			fmt.Fprintf(os.Stderr, "timeout")
		}
	}
}

func (h callbackHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "OK")

	result, err := zClient.ProcessCallback(r)
	if err != nil {
		h.errCh <- err
		return
	}

	h.doneCh <- result
}
