package main

import (
	"flag"
	"fmt"
	"html"
	"net/http"
	"os"
	"strings"
	"time"

	"zvelo.io/msg/go-msg"
)

const (
	callbackDefaultListenAddress = "[::1]:8080"
	callbackDefaultTimeout       = 15 * time.Minute
)

var (
	handler  callbackHandler
	dataset  string
	datasets = []msg.DataSetType{}
)

type callbackHandler struct {
	config struct {
		URLs                       []string
		ListenAddress, CallbackURL string
		Timeout                    time.Duration
		PartialResults             bool
		Poll                       bool
	}
	doneCh chan *http.Request // TODO(jrubin)
}

func init() {
	fs := flag.NewFlagSet("query", flag.ExitOnError)
	fs.Usage = cmdUsage(fs, "url [url...]")

	fs.BoolVar(&handler.config.Poll, "poll", getDefaultBool("ZVELO_POLL"), "poll for results [$ZVELO_POLL]")

	fs.StringVar(
		&handler.config.ListenAddress,
		"listen-address",
		getDefaultString("ZVELO_LISTEN_ADDRESS", callbackDefaultListenAddress),
		"address and port to listen for callbacks on [$ZVELO_LISTEN_ADDRESS]",
	)

	fs.StringVar(
		&handler.config.CallbackURL,
		"callback-url",
		getDefaultString("ZVELO_CALLBACK_URL", ""),
		"publicly accessible base URL that routes to the address used by the -listen-address flag [$ZVELO_CALLBACK_URL]",
	)

	fs.DurationVar(
		&handler.config.Timeout,
		"timeout",
		callbackDefaultTimeout,
		"maximum amount of time to wait for the callback to be called",
	)

	fs.BoolVar(
		&handler.config.PartialResults,
		"partial-results",
		false,
		"request that datasets be delivered as soon as they become available instead of waiting for all datasets to become available before responding",
	)

	allDatasets := make([]string, len(msg.DataSetType_name)-1)
	i := 0
	for dst, name := range msg.DataSetType_name {
		if dst == int32(msg.DataSetType_ECHO) {
			continue
		}

		allDatasets[i] = name
		i++
	}

	fs.StringVar(
		&dataset,
		"dataset",
		getDefaultString("ZVELO_DATASET", "CATEGORIZATION"),
		"comma separated list of datasets to retrieve (available options: "+strings.Join(allDatasets, ", ")+") [$ZVELO_DATASET]",
	)

	cmd["query"] = subcommand{
		FlagSet: fs,
		Setup:   setupQuery,
		Action:  queryURL,
		Usage:   "query for url",
	}
}

func setupQuery() error {
	if handler.config.Poll && len(handler.config.CallbackURL) > 0 {
		return fmt.Errorf("poll and callback can't both be true")
	}

	if len(handler.config.CallbackURL) > 0 {
		handler.doneCh = make(chan *http.Request, 1) // TODO(jrubin)
		go func() { _ = http.ListenAndServe(handler.config.ListenAddress, handler) }()
	}

	handler.config.URLs = cmd["query"].FlagSet.Args()

	if len(handler.config.URLs) == 0 {
		return fmt.Errorf("at least one url is required")
	}

	for _, dsName := range strings.Split(dataset, ",") {
		dsName = strings.TrimSpace(dsName)
		dst, err := msg.NewDataSetType(dsName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "invalid dataset type: %s\n", dsName)
			continue
		}
		datasets = append(datasets, dst)
	}

	if len(datasets) == 0 {
		return fmt.Errorf("at least one valid dataset is required")
	}

	return nil
}

func queryURL() error {
	req := &msg.QueryURLRequests{
		Url:     handler.config.URLs,
		Dataset: datasets,
	}

	if len(handler.config.CallbackURL) > 0 {
		req.Callback = handler.config.CallbackURL
		req.PartialResults = handler.config.PartialResults
	} else {
		req.Poll = true
	}

	reply, err := zClient.Query(req)
	if err != nil {
		return err
	}

	if handler.config.Poll {
		return pollForResults(reply)
	}

	if len(handler.config.CallbackURL) > 0 {
		return waitForCallback(reply)
	}

	fmt.Printf("Request ID(s): %s\n", strings.Join(reply.RequestId, ", "))
	return nil
}

func pollForResults(reply *msg.QueryReply) error {
	errCh := make(chan error)

	// TODO(jrubin) only poll for the first reqid?
	resultCh := zClient.Poll(reply.RequestId[0], errCh)

	for {
		select {
		case err := <-errCh:
			fmt.Fprintf(os.Stderr, "%s\n", err) // TODO(jrubin)
		case result, ok := <-resultCh:
			if !ok {
				fmt.Fprintf(os.Stderr, "timeout\n")
				return nil
			}

			return handleResult(result)
		}
	}
}

func waitForCallback(reply *msg.QueryReply) error {
	// TODO(jrubin)
	select {
	case req := <-handler.doneCh:
		// TODO(jrubin)
		zClient.DebugRequest(req)
		return nil
	case <-time.After(handler.config.Timeout):
		// TODO(jrubin)
		fmt.Fprintf(os.Stderr, "timeout")
		return nil
	}
}

func (h callbackHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// TODO(jrubin)
	fmt.Fprintf(w, "Hello, %q", html.EscapeString(r.URL.Path))
	// TODO(jrubin) when are we really done?
	h.doneCh <- r
}
