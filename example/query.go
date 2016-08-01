package main

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/urfave/cli"
	"zvelo.io/msg"
)

const (
	callbackDefaultTimeout = 15 * time.Minute
)

type config struct {
	URLs                       []string
	ListenAddress, CallbackURL string
	Timeout                    time.Duration
	PartialResults             bool
	Poll                       bool
}

type callbackHandler struct {
	cfg    config
	doneCh chan *msg.QueryResult
	errCh  chan error
}

var cbHandler callbackHandler

func init() {
	app.Commands = append(app.Commands, cli.Command{
		Name:   "query",
		Usage:  "query for a URL",
		Action: queryURL,
		Before: setupQuery,
		Flags: []cli.Flag{
			cli.StringSliceFlag{
				Name:  "url, u",
				Usage: "list of URLs to query for",
			},
			cli.StringFlag{
				Name:   "listen, l",
				EnvVar: "ZVELO_QUERY_LISTEN_ADDRESS",
				Usage:  "address and port to listen for callbacks",
				Value:  "[::1]:8080",
			},
			cli.StringFlag{
				Name:   "callback, cb",
				EnvVar: "ZVELO_QUERY_CALLBACK_URL",
				Usage:  "publicly accessible base URL that routes to the address used by -listen flag",
			},
			cli.DurationFlag{
				Name:   "timeout, qt",
				EnvVar: "ZVELO_QUERY_TIMEOUT",
				Usage:  "maximum amount of time to wait for the callback to be called",
				Value:  callbackDefaultTimeout,
			},
			cli.BoolFlag{
				Name:   "partial-results, pr",
				EnvVar: "ZVELO_QUERY_PARTIAL_RESULTS",
				Usage:  "request that datasets be delivered as soon as they become available instead of waiting for all datasets to become available before responding",
			},
			cli.BoolFlag{
				Name:   "poll, qp",
				EnvVar: "ZVELO_QUERY_POLL",
				Usage:  "poll for resutls",
			},
		},
	})
}

func setupQuery(c *cli.Context) error {
	if err := setupClient(c); err != nil {
		return err
	}

	if err := setupDS(c); err != nil {
		return err
	}

	cbHandler.cfg = config{
		URLs:           c.StringSlice("url"),
		ListenAddress:  c.String("listen"),
		CallbackURL:    c.String("callback"),
		Timeout:        c.Duration("timeout"),
		PartialResults: c.Bool("partial-results"),
		Poll:           c.Bool("poll"),
	}

	if cbHandler.cfg.Poll && len(cbHandler.cfg.CallbackURL) > 0 {
		return fmt.Errorf("poll and callback can't both be true")
	}

	if len(cbHandler.cfg.CallbackURL) > 0 {
		cbHandler.errCh = make(chan error, 1)
		cbHandler.doneCh = make(chan *msg.QueryResult, 1)
		go func() { _ = http.ListenAndServe(cbHandler.cfg.ListenAddress, cbHandler) }()
	}

	cbHandler.cfg.URLs = c.StringSlice("url")

	if len(cbHandler.cfg.URLs) == 0 {
		return fmt.Errorf("at least one url is required")
	}

	return nil
}

func queryURL(c *cli.Context) error {
	req := &msg.QueryURLRequests{
		Url:     cbHandler.cfg.URLs,
		Dataset: datasets,
	}

	if len(cbHandler.cfg.CallbackURL) > 0 {
		req.Callback = cbHandler.cfg.CallbackURL
		req.PartialResults = cbHandler.cfg.PartialResults
		// TODO(jrubin) add support for Accept field in msg.QueryURLRequests
	} else {
		req.Poll = true
	}

	reply, err := zClient.Query(req)
	if err != nil {
		return err
	}

	if cbHandler.cfg.Poll {
		return pollForResults(reply)
	}

	if len(cbHandler.cfg.CallbackURL) > 0 {
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
		case <-time.After(cbHandler.cfg.Timeout):
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
