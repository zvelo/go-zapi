package zapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"

	"golang.org/x/oauth2"

	"github.com/pkg/errors"

	"zvelo.io/msg"
)

const queryV1Path = "/v1/query"

type restClient struct {
	options *options
	client  *http.Client
}

// CallOption configures a Call before it starts or extracts information from
// a Call after it completes. It is only used with the RESTClient.
// grpc.CallOption is still available for the GRPCClient.
type CallOption interface {
	after(*http.Response)
}

type afterCall func(*http.Response)

func (o afterCall) after(resp *http.Response) { o(resp) }

// Response will return the entire http.Response received from a zveloAPI call.
// This is useful to request or response headers, see http error messages, read
// the raw body and more.
func Response(h **http.Response) CallOption {
	return afterCall(func(resp *http.Response) {
		*h = resp
	})
}

// A RESTClient implements a very similar interface to GRPCClient but uses a
// standard HTTP/REST transport instead of gRPC. Generally the gRPC client is
// preferred for its efficiency.
type RESTClient interface {
	QueryV1(ctx context.Context, in *msg.QueryRequests, opt ...CallOption) (*msg.QueryReplies, error)
	QueryResultV1(ctx context.Context, reqID string, opt ...CallOption) (*msg.QueryResult, error)
}

// NewREST returns a properly configured RESTClient
func NewREST(ts oauth2.TokenSource, opts ...Option) RESTClient {
	o := defaults(ts)
	for _, opt := range opts {
		opt(o)
	}

	return &restClient{
		options: o,
		client:  &http.Client{Transport: &transport{options: o}},
	}
}

func (c *restClient) Do(ctx context.Context, req *http.Request) (*http.Response, error) {
	return c.client.Do(req.WithContext(ctx))
}

func (c *restClient) QueryV1(ctx context.Context, in *msg.QueryRequests, opts ...CallOption) (*msg.QueryReplies, error) {
	url := c.options.restURL(queryV1Path)

	data, err := json.Marshal(in)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", url, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.Do(ctx, req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	for _, opt := range opts {
		opt.after(resp)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, errors.Errorf("http error: %s (%d)", resp.Status, resp.StatusCode)
	}

	var replies msg.QueryReplies
	if err := json.NewDecoder(resp.Body).Decode(&replies); err != nil {
		return nil, err
	}

	return &replies, nil
}

func (c *restClient) QueryResultV1(ctx context.Context, reqID string, opts ...CallOption) (*msg.QueryResult, error) {
	url := c.options.restURL(queryV1Path, reqID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.Do(ctx, req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	for _, opt := range opts {
		opt.after(resp)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, errors.Errorf("http error: %s (%d)", resp.Status, resp.StatusCode)
	}

	var result msg.QueryResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
