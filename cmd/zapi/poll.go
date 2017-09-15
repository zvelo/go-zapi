package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"github.com/pkg/errors"
	"github.com/urfave/cli"

	zapi "zvelo.io/go-zapi"
	"zvelo.io/msg"
)

var pollOnce bool
var pollRequestIDs []string

func init() {
	app.Commands = append(app.Commands, cli.Command{
		Name:      "poll",
		Usage:     "poll for results with a request-id",
		ArgsUsage: "request_id [request_id...]",
		Before:    setupPoll,
		Action:    poll,
		Flags: []cli.Flag{
			cli.BoolFlag{
				Name:        "once",
				EnvVar:      "ZVELO_POLL_ONCE",
				Usage:       "make just a single poll request",
				Destination: &pollOnce,
			},
		},
	})
}

func setupPoll(c *cli.Context) error {
	pollRequestIDs = c.Args()

	if len(pollRequestIDs) == 0 {
		return errors.New("at least one request_id is required")
	}

	return nil
}

func poll(_ *cli.Context) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	reqIDs := map[string]string{}
	for _, reqID := range pollRequestIDs {
		reqIDs[reqID] = ""
	}

	if pollOnce {
		return pollReqIDsOnce(ctx, reqIDs)
	}

	return pollReqIDs(ctx, reqIDs)
}

func pollReqIDs(ctx context.Context, reqIDs map[string]string) error {
	polling := map[string]string{}
	for reqID, url := range reqIDs {
		polling[reqID] = url
	}

	for len(polling) > 0 {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(pollInterval):
		}

		if err := pollReqIDsOnce(ctx, polling); err != nil {
			return err
		}
	}

	return nil
}

func pollReqIDsOnce(ctx context.Context, reqIDs map[string]string) error {
	for reqID, url := range reqIDs {
		complete, err := pollReqID(ctx, reqID, url)

		if err != nil {
			return err
		}

		if complete {
			delete(reqIDs, reqID)
		}
	}

	return nil
}

func pollReqID(ctx context.Context, reqID, url string) (bool, error) {
	if url == "" {
		fmt.Fprintf(os.Stderr, "polling for: %s\n", reqID)
	} else {
		fmt.Fprintf(os.Stderr, "polling for: %s (%s)\n", url, reqID)
	}

	var result *msg.QueryResult
	var traceID string
	var err error

	if rest {
		result, traceID, err = pollREST(ctx, reqID)
	} else {
		result, traceID, err = pollGRPC(ctx, reqID)
	}

	if err != nil {
		return false, err
	}

	fmt.Println()

	if traceID != "" {
		fmt.Fprintf(os.Stderr, "Trace ID:           %s\n", traceID)
	}

	if err := queryResultTpl.ExecuteTemplate(os.Stdout, "QueryResult", result); err != nil {
		return false, err
	}

	if result == nil || result.QueryStatus == nil {
		return false, nil
	}

	return result.QueryStatus.Complete, nil
}

func pollREST(ctx context.Context, reqID string) (*msg.QueryResult, string, error) {
	req := msg.QueryPollRequest{RequestId: reqID}
	var resp *http.Response
	result, err := restClient.QueryContentResultV1(ctx, &req, zapi.Response(&resp))
	traceID := resp.Header.Get("trace-id")
	return result, traceID, err
}

func pollGRPC(ctx context.Context, reqID string) (*msg.QueryResult, string, error) {
	req := msg.QueryPollRequest{RequestId: reqID}
	if forceTrace {
		ctx = metadata.NewOutgoingContext(ctx, metadata.Pairs(
			"jaeger-debug-id", randString(32),
		))
	}
	var header metadata.MD
	result, err := grpcClient.QueryContentResultV1(ctx, &req, grpc.Header(&header))
	var traceID string
	if tids, ok := header["trace-id"]; ok && len(tids) > 0 {
		traceID = tids[0]
	}
	return result, traceID, err
}
