package zapi

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"

	"github.com/gogo/protobuf/jsonpb"
	"github.com/pkg/errors"

	"golang.org/x/oauth2"

	"google.golang.org/grpc/metadata"

	"zvelo.io/msg"
)

const (
	queryV1Path = "/v1/query"
	graphQLPath = "/graphql"
)

var (
	jsonMarshaler   jsonpb.Marshaler
	jsonUnmarshaler jsonpb.Unmarshaler
)

type restV1Client struct {
	options *options
	client  *http.Client
}

// CallOption configures a Call before it starts or extracts information from a
// Call after it completes. It is only used with the RESTv1Client.
// grpc.CallOption is still available for the GRPCv1Client.
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

// A RESTv1Client implements a very similar interface to GRPCv1Client but uses a
// standard HTTP/REST transport instead of gRPC. Generally the gRPC client is
// preferred for its efficiency.
type RESTv1Client interface {
	Query(ctx context.Context, in *msg.QueryRequests, opt ...CallOption) (*msg.QueryReplies, error)
	Result(ctx context.Context, reqID string, opt ...CallOption) (*msg.QueryResult, error)
	GraphQL(ctx context.Context, query string, result interface{}, opt ...CallOption) error
}

// NewRESTv1 returns a properly configured RESTv1Client
func NewRESTv1(ts oauth2.TokenSource, opts ...Option) RESTv1Client {
	o := defaults(ts)
	for _, opt := range opts {
		opt(o)
	}

	return &restV1Client{
		options: o,
		client:  &http.Client{Transport: &transport{options: o}},
	}
}

func (c *restV1Client) Do(ctx context.Context, req *http.Request) (*http.Response, error) {
	return c.client.Do(req.WithContext(ctx))
}

func (c *restV1Client) GraphQL(ctx context.Context, query string, result interface{}, opts ...CallOption) error {
	url := c.options.restURL(graphQLPath)

	query = `{"query":` + strconv.QuoteToASCII(query) + `}`

	req, err := http.NewRequest("POST", url, strings.NewReader(query))
	if err != nil {
		return err
	}

	if md, ok := metadata.FromOutgoingContext(ctx); ok {
		for k, vs := range md {
			for _, v := range vs {
				req.Header.Add(k, v)
			}
		}
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.Do(ctx, req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	for _, opt := range opts {
		opt.after(resp)
	}

	if resp.StatusCode != http.StatusOK {
		return errors.Errorf("http error: %s", resp.Status)
	}

	if ps, ok := result.(*string); ok {
		s, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		*ps = string(s)
		return nil
	}

	return json.NewDecoder(resp.Body).Decode(result)
}

func (c *restV1Client) Query(ctx context.Context, in *msg.QueryRequests, opts ...CallOption) (*msg.QueryReplies, error) {
	url := c.options.restURL(queryV1Path)

	var buf bytes.Buffer
	if err := jsonMarshaler.Marshal(&buf, in); err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", url, &buf)
	if err != nil {
		return nil, err
	}

	if md, ok := metadata.FromOutgoingContext(ctx); ok {
		for k, vs := range md {
			for _, v := range vs {
				req.Header.Add(k, v)
			}
		}
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
		return nil, errors.Errorf("http error: %s", resp.Status)
	}

	var replies msg.QueryReplies
	if err := jsonUnmarshaler.Unmarshal(resp.Body, &replies); err != nil {
		return nil, err
	}

	return &replies, nil
}

func (c *restV1Client) Result(ctx context.Context, reqID string, opts ...CallOption) (*msg.QueryResult, error) {
	url := c.options.restURL(queryV1Path, reqID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	if md, ok := metadata.FromOutgoingContext(ctx); ok {
		for k, vs := range md {
			for _, v := range vs {
				req.Header.Add(k, v)
			}
		}
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
	if err := jsonUnmarshaler.Unmarshal(resp.Body, &result); err != nil {
		return nil, err
	}

	return &result, nil
}
