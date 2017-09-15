package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"text/tabwriter"
	"text/template"
	"time"

	"github.com/urfave/cli"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	zapi "zvelo.io/go-zapi"
	"zvelo.io/msg"
)

var (
	queryReq          msg.QueryURLRequests
	queryListen       string
	queryPollInterval time.Duration
	queryTimeout      time.Duration

	queryCh = make(chan *msg.QueryResult)
)

func init() {
	app.Commands = append(app.Commands, cli.Command{
		Name:      "query",
		Usage:     "query for a URL",
		ArgsUsage: "url [url...]",
		Before:    setupQuery,
		Action:    query,
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:        "listen",
				EnvVar:      "ZVELO_QUERY_LISTEN_ADDRESS",
				Usage:       "address and port to listen for callbacks",
				Value:       ":8080",
				Destination: &queryListen,
			},
			cli.StringFlag{
				Name:        "callback",
				EnvVar:      "ZVELO_QUERY_CALLBACK_URL",
				Usage:       "publicly accessible base URL that routes to the address used by -listen flag",
				Destination: &queryReq.Callback,
			},
			cli.DurationFlag{
				Name:        "poll-interval",
				EnvVar:      "ZVELO_QUERY_POLL_INTERVAL",
				Usage:       "repeatedly poll after this much time has elapsed until the query is marked as complete",
				Value:       1 * time.Second,
				Destination: &queryPollInterval,
			},
			cli.DurationFlag{
				Name:        "timeout",
				EnvVar:      "ZVELO_QUERY_TIMEOUT",
				Usage:       "maximum amount of time to wait for query results to complete",
				Value:       15 * time.Minute,
				Destination: &queryTimeout,
			},
			cli.BoolFlag{
				Name:        "partial-results",
				EnvVar:      "ZVELO_QUERY_PARTIAL_RESULTS",
				Usage:       "request that datasets be delivered as soon as they become available instead of waiting for all datasets to become available before responding",
				Destination: &queryReq.PartialResults,
			},
			cli.BoolFlag{
				Name:        "poll",
				EnvVar:      "ZVELO_QUERY_POLL",
				Usage:       "poll for resutls",
				Destination: &queryReq.Poll,
			},
		},
	})
}

func setupQuery(c *cli.Context) error {
	if queryReq.Poll && queryReq.Callback != "" {
		return errors.New("poll and callback can't both be enabled")
	}

	for _, u := range c.Args() {
		if !strings.Contains(u, "://") {
			u = "http://" + u
		}
		queryReq.Url = append(queryReq.Url, u)
	}

	queryReq.Dataset = datasets

	if len(queryReq.Url) == 0 {
		return errors.New("at least one url is required")
	}

	if queryReq.Callback != "" {
		go func() {
			_ = http.ListenAndServe(
				queryListen,
				zapi.CallbackHandler(callbackHandler()),
			)
		}()
	}

	return nil
}

func callbackHandler() zapi.Handler {
	return zapi.HandlerFunc(func(in *msg.QueryResult) {
		queryCh <- in
	})
}

func query(_ *cli.Context) error {
	ctx, cancel := context.WithTimeout(context.Background(), queryTimeout)
	defer cancel()

	if rest {
		return queryREST(ctx)
	}

	return queryGRPC(ctx)
}

func queryREST(ctx context.Context) error {
	var resp *http.Response
	replies, err := restClient.QueryURLV1(ctx, &queryReq, zapi.Response(&resp))
	if err != nil {
		return err
	}

	return queryWait(ctx, resp.Header.Get("trace-id"), replies.Replies)
}

func queryGRPC(ctx context.Context) error {
	if forceTrace {
		ctx = metadata.NewOutgoingContext(ctx, metadata.Pairs(
			"jaeger-debug-id", randString(32),
		))
	}

	var header metadata.MD
	replies, err := grpcClient.QueryURLV1(ctx, &queryReq, grpc.Header(&header))
	if err != nil {
		return err
	}

	var traceID string
	if tids, ok := header["trace-id"]; ok && len(tids) > 0 {
		traceID = tids[0]
	}

	return queryWait(ctx, traceID, replies.Replies)
}

var tplStr = `
{{define "DataSet" -}}
{{- if .Categorization -}}
Categories:         {{range .Categorization.Values}}{{category .}} {{end}}
{{end}}

{{- if .Malicious -}}
Malicious:          {{.Malicious.Verdict}}{{if .Malicious.Category}}:{{category .Malicious.Category}}{{end}}
{{end}}

{{- if .Echo}}Echo:               {{.Echo.Url}}
{{end}}
{{- end}}

{{define "Status" -}}
{{- if .Code}}Error Code:         {{.Code}}
{{end}}
{{- if .Message}}Error Message:      {{.Message}}
{{end}}
{{- end}}

{{define "QueryStatus" -}}
Complete:           {{.Complete}}
{{if .FetchCode}}Fetch Status:       {{httpStatus .FetchCode}}
{{end}}
{{- if .Location}}Redirect Location:  {{.Location}}
{{end}}
{{- if .Error}}{{template "Status" .Error}}{{end}}
{{- end}}

{{define "QueryResult" -}}
{{- if .Url}}URL:                {{.Url}}
{{end}}
{{- if .RequestId}}Request ID:         {{.RequestId}}
{{end}}
{{- if .RequestDataset -}}
Requested Datasets: {{range .RequestDataset}}{{dataset .}} {{end}}
{{end}}
{{- if .ResponseDataset}}{{template "DataSet" .ResponseDataset}}{{end}}
{{- if .QueryStatus}}{{template "QueryStatus" .QueryStatus}}{{end}}
{{- end}}`

var tpl = template.Must(template.New("QueryResult").Funcs(template.FuncMap{
	"dataset": func(i uint32) string {
		return msg.DataSetType(i).String()
	},
	"category": func(i uint32) string {
		cat := msg.Category(i)
		return fmt.Sprintf("%s(%d)", cat, i)
	},
	"httpStatus": func(i int32) string {
		return fmt.Sprintf("%s(%d)", http.StatusText(int(i)), i)
	},
}).Parse(tplStr))

func queryWait(ctx context.Context, traceID string, replies []*msg.QueryReply) error {
	w := tabwriter.NewWriter(os.Stderr, 0, 0, 1, ' ', 0)

	if traceID != "" {
		fmt.Fprintf(w, "Trace ID:\t%s\n", traceID)
	}

	reqIDs := map[string]string{}
	for i, reply := range replies {
		reqIDs[reply.RequestId] = queryReq.Url[i]
		fmt.Fprintf(w, "%s:\t%s\n", queryReq.Url[i], reply.RequestId)
	}

	if err := w.Flush(); err != nil {
		return err
	}

	if queryReq.Callback != "" {
		return queryWaitCallback(ctx)
	}

	if queryReq.Poll {
		return queryPoll(ctx, reqIDs)
	}

	return nil
}

func queryWaitCallback(ctx context.Context) error {
	var numComplete int

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case result := <-queryCh:
			fmt.Fprintf(os.Stderr, "\nreceived callback\n")

			fmt.Println()
			if err := tpl.ExecuteTemplate(os.Stdout, "QueryResult", result); err != nil {
				return err
			}

			if result.QueryStatus != nil && result.QueryStatus.Complete {
				numComplete++

				if numComplete == len(queryReq.Url) {
					return nil
				}
			}
		}
	}
}

func queryPollREST(ctx context.Context, reqID string) (*msg.QueryResult, string, error) {
	req := msg.QueryPollRequest{RequestId: reqID}
	var resp *http.Response
	result, err := restClient.QueryContentResultV1(ctx, &req, zapi.Response(&resp))
	traceID := resp.Header.Get("trace-id")
	return result, traceID, err
}

func queryPollGRPC(ctx context.Context, reqID string) (*msg.QueryResult, string, error) {
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

func queryPoll(ctx context.Context, reqIDs map[string]string) error {
	var result *msg.QueryResult
	var err error

	polling := map[string]string{}
	for reqID, url := range reqIDs {
		polling[reqID] = url
	}

	for len(polling) > 0 {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(queryPollInterval):
		}

		for reqID, url := range polling {
			fmt.Fprintf(os.Stderr, "\npolling for: %s (%s)\n", url, reqID)

			var traceID string

			if rest {
				result, traceID, err = queryPollREST(ctx, reqID)
			} else {
				result, traceID, err = queryPollGRPC(ctx, reqID)
			}

			if err != nil {
				return err
			}

			fmt.Println()
			if traceID != "" {
				fmt.Fprintf(os.Stderr, "Trace ID:           %s\n", traceID)
			}
			if err := tpl.ExecuteTemplate(os.Stdout, "QueryResult", result); err != nil {
				return err
			}

			if result != nil && result.QueryStatus != nil && result.QueryStatus.Complete {
				delete(polling, reqID)
			}
		}
	}

	return nil
}
