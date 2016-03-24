package main

import (
	"encoding/base64"
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
	queryURLs []string
	queryPoll bool
	handler   callbackHandler
)

type callbackHandler struct {
	config struct {
		ListenAddress, CallbackURL string
		Timeout                    time.Duration
		PartialResults             bool
	}
	doneCh chan *http.Request // TODO(jrubin)
}

func init() {
	fs := flag.NewFlagSet("query", flag.ExitOnError)
	fs.Usage = cmdUsage(fs, "url [url...]")

	fs.BoolVar(&queryPoll, "poll", getDefaultBool("ZVELO_POLL"), "poll for results [$ZVELO_POLL]")

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
		"publicly accessible base URL that routes to the address used by the address flag [$ZVELO_CALLBACK_URL]",
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

	cmd["query"] = subcommand{
		FlagSet: fs,
		Setup:   setupQuery,
		Action:  queryURL,
		Usage:   "query for url",
	}
}

func setupQuery() error {
	if queryPoll && len(handler.config.CallbackURL) > 0 {
		return fmt.Errorf("poll and callback can't both be true")
	}

	if len(handler.config.CallbackURL) > 0 {
		handler.doneCh = make(chan *http.Request, 1) // TODO(jrubin)
		go http.ListenAndServe(handler.config.ListenAddress, handler)
	}

	queryURLs = cmd["query"].FlagSet.Args()

	if len(queryURLs) == 0 {
		return fmt.Errorf("at least one url is required")
	}

	return nil
}

func queryURL() error {
	req := &msg.QueryURLRequests{
		Url: queryURLs,
		Dataset: []msg.DataSetType{
			msg.DataSetType_CATEGORIZATION,
			msg.DataSetType_ADFRAUD,
		},
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

	// TODO(jrubin) check reply.Status?
	// TODO(jrubin) assert(len(reply.RequestIDs) > 0)

	if queryPoll {
		return pollForResults(reply)
	}

	if len(handler.config.CallbackURL) > 0 {
		return waitForCallback(reply)
	}

	strID := make([]string, len(reply.RequestId))

	for i, id := range reply.RequestId {
		strID[i] = base64.StdEncoding.EncodeToString(id)
	}

	fmt.Printf("Request ID(s): %s\n", strings.Join(strID, ", "))
	return nil
}

func pollForResults(reply *msg.QueryReply) error {
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
