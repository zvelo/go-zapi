package zapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"path"
	"strings"

	"golang.org/x/oauth2"

	"github.com/pkg/errors"

	"zvelo.io/msg"
)

const (
	queryURLV1Path     = "/v1/queries/url"
	queryContentV1Path = "/v1/queries/content"
)

func restURL(base, dir string) (string, error) {
	if !strings.Contains(base, "://") {
		base = "https://" + base
	}

	p, err := url.Parse(base)
	if err != nil {
		return "", err
	}

	p.Path = path.Join(p.Path, dir)

	return p.String(), nil
}

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
	QueryURLV1(context.Context, *msg.QueryURLRequests, ...CallOption) (*msg.QueryReplies, error)
	QueryURLResultV1(context.Context, *msg.QueryPollRequest, ...CallOption) (*msg.QueryResult, error)
	QueryContentV1(context.Context, *msg.QueryContentRequests, ...CallOption) (*msg.QueryReplies, error)
	QueryContentResultV1(context.Context, *msg.QueryPollRequest, ...CallOption) (*msg.QueryResult, error)
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

func (c *restClient) do(ctx context.Context, req *http.Request) (*http.Response, error) {
	return c.client.Do(req.WithContext(ctx))
}

func (c *restClient) queryV1(ctx context.Context, path string, in interface{}, opts ...CallOption) (*msg.QueryReplies, error) {
	url, err := restURL(c.options.host, path)
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(in)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", url, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.do(ctx, req)
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

func (c *restClient) queryResultV1(ctx context.Context, reqID string, opts ...CallOption) (*msg.QueryResult, error) {
	url, err := restURL(c.options.host, path.Join(queryURLV1Path, reqID))
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.do(ctx, req)
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

func (c *restClient) QueryURLV1(ctx context.Context, in *msg.QueryURLRequests, opts ...CallOption) (*msg.QueryReplies, error) {
	return c.queryV1(ctx, queryURLV1Path, in, opts...)
}

func (c *restClient) QueryContentV1(ctx context.Context, in *msg.QueryContentRequests, opts ...CallOption) (*msg.QueryReplies, error) {
	return c.queryV1(ctx, queryContentV1Path, in, opts...)
}

func (c *restClient) QueryURLResultV1(ctx context.Context, in *msg.QueryPollRequest, opts ...CallOption) (*msg.QueryResult, error) {
	return c.queryResultV1(ctx, in.RequestId, opts...)
}

func (c *restClient) QueryContentResultV1(ctx context.Context, in *msg.QueryPollRequest, opts ...CallOption) (*msg.QueryResult, error) {
	return c.queryResultV1(ctx, in.RequestId, opts...)
}
