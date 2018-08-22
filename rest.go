package zapi

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"

	"github.com/gogo/protobuf/jsonpb"
	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"

	"golang.org/x/net/http2"
	"golang.org/x/oauth2"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	msg "zvelo.io/msg/msgpb"
)

const (
	queryV1Path   = "/v1/query"
	suggestV1Path = "/v1/suggest"
	streamV1Path  = "/v1/stream"
	graphQLPath   = "/graphql"
)

var (
	jsonMarshaler   jsonpb.Marshaler
	jsonUnmarshaler = jsonpb.Unmarshaler{
		AllowUnknownFields: true,
	}
)

type restV1Client struct {
	options *options
	client  *http.Client
}

// CallOption configures a Call before it starts or extracts information from a
// Call after it completes. It is only used with the RESTv1Client.
// grpc.CallOption is still available for the GRPCv1Client.
type CallOption interface {
	before(*http.Request)
	after(*http.Response)
}

type afterCall func(*http.Response)

func (o afterCall) before(*http.Request)      {}
func (o afterCall) after(resp *http.Response) { o(resp) }

type beforeCall func(*http.Request)

func (o beforeCall) before(r *http.Request) { o(r) }
func (o beforeCall) after(*http.Response)   {}

// Response will return the entire http.Response received from a zveloAPI call.
// This is useful to request or response headers, see http error messages, read
// the raw body and more.
func Response(h **http.Response) CallOption {
	return afterCall(func(resp *http.Response) {
		*h = resp
	})
}

// WithHeader adds the header named key with value val to outgoing REST http
// requests
func WithHeader(key, val string) CallOption {
	return beforeCall(func(r *http.Request) {
		r.Header.Add(key, val)
	})
}

// RESTv1StreamClient provides an interface to receive streamed query results
// from zveloAPI servers.
type RESTv1StreamClient interface {
	Recv() (*msg.QueryResult, error)
}

// A RESTv1Client implements a very similar interface to GRPCv1Client but uses a
// standard HTTP/REST transport instead of gRPC. Generally the gRPC client is
// preferred for its efficiency.
type RESTv1Client interface {
	Query(ctx context.Context, in *msg.QueryRequests, opts ...CallOption) (*msg.QueryReplies, error)
	Result(ctx context.Context, reqID string, opts ...CallOption) (*msg.QueryResult, error)
	GraphQL(ctx context.Context, query string, result interface{}, opts ...CallOption) error
	Suggest(ctx context.Context, in *msg.Suggestion, opts ...CallOption) error
	Stream(ctx context.Context) (RESTv1StreamClient, error)
}

// NewRESTv1 returns a properly configured RESTv1Client
func NewRESTv1(ts oauth2.TokenSource, opts ...Option) RESTv1Client {
	o := defaults(ts)
	for _, opt := range opts {
		opt(o)
	}

	if t, ok := o.transport.(*http.Transport); ok {
		_ = http2.ConfigureTransport(t) // #nosec
	}

	return &restV1Client{
		options: o,
		client:  &http.Client{Transport: &transport{options: o}},
	}
}

func (c *restV1Client) GraphQL(ctx context.Context, query string, result interface{}, opts ...CallOption) error {
	url := c.options.restURL(graphQLPath)

	query = `{"query":` + strconv.QuoteToASCII(query) + `}`

	body, err := c.do(ctx, "POST", url, strings.NewReader(query), opts...)
	if err != nil {
		return err
	}
	defer func() { _ = body.Close() }() // #nosec

	if ps, ok := result.(*string); ok {
		s, err := ioutil.ReadAll(body)
		if err != nil {
			return err
		}
		*ps = string(s)
		return nil
	}

	return json.NewDecoder(body).Decode(result)
}

func (c *restV1Client) Query(ctx context.Context, in *msg.QueryRequests, opts ...CallOption) (*msg.QueryReplies, error) {
	url := c.options.restURL(queryV1Path)
	var replies msg.QueryReplies
	if err := c.doPB(ctx, "POST", url, in, &replies, opts...); err != nil {
		return nil, err
	}
	return &replies, nil
}

func (c *restV1Client) Result(ctx context.Context, reqID string, opts ...CallOption) (*msg.QueryResult, error) {
	url := c.options.restURL(queryV1Path, reqID)
	var result msg.QueryResult
	if err := c.doPB(ctx, "GET", url, nil, &result, opts...); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *restV1Client) Suggest(ctx context.Context, in *msg.Suggestion, opts ...CallOption) error {
	url := c.options.restURL(suggestV1Path)
	return c.doPB(ctx, "POST", url, in, nil, opts...)
}

type errorBody struct {
	Error string     `json:"error"`
	Code  codes.Code `json:"code"`
}

func (c *restV1Client) do(ctx context.Context, method, url string, body io.Reader, opts ...CallOption) (io.ReadCloser, error) {
	req, err := http.NewRequest(method, url, body)
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

	for _, opt := range opts {
		opt.before(req)
	}

	resp, err := c.client.Do(req.WithContext(ctx))
	if err != nil {
		return nil, err
	}

	for _, opt := range opts {
		opt.after(resp)
	}

	if resp.StatusCode != http.StatusOK {
		// try to resolve the body as a grpc error
		var eb errorBody
		if err = json.NewDecoder(resp.Body).Decode(&eb); err == nil && eb.Error != "" && eb.Code != 0 {
			_ = resp.Body.Close() // #nosec
			return nil, status.Error(eb.Code, eb.Error)
		}
		_ = resp.Body.Close() // #nosec
		return nil, errors.Errorf("http error: %s", resp.Status)
	}

	return resp.Body, nil
}

func (c *restV1Client) doPB(ctx context.Context, method, url string, in, out proto.Message, opts ...CallOption) error {
	var reqBody io.Reader
	if in != nil {
		var buf bytes.Buffer
		if err := jsonMarshaler.Marshal(&buf, in); err != nil {
			return err
		}
		reqBody = &buf
	}

	body, err := c.do(ctx, method, url, reqBody, opts...)
	if err != nil {
		return err
	}
	defer func() { _ = body.Close() }() // #nosec

	if out == nil {
		return nil
	}

	return jsonUnmarshaler.Unmarshal(body, out)
}

type restV1StreamClient struct {
	io.Closer
	*json.Decoder
}

func (c *restV1Client) Stream(ctx context.Context) (RESTv1StreamClient, error) {
	url := c.options.restURL(streamV1Path)

	ctx = context.WithValue(ctx, debugDumpResponseBodyKey, false)

	body, err := c.do(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	return restV1StreamClient{
		Closer:  body,
		Decoder: json.NewDecoder(body),
	}, nil
}

type streamError struct {
	Code    codes.Code `json:"grpc_code,omitempty"`
	Message string     `json:"message,omitempty"`
}

func (e *streamError) Err() error {
	if e == nil {
		return nil
	}

	return status.Error(e.Code, e.Message)
}

type streamItem struct {
	Result json.RawMessage `json:"result"`
	Error  *streamError
}

func (i streamItem) Err() error {
	return i.Error.Err()
}

func (c restV1StreamClient) Recv() (*msg.QueryResult, error) {
	var item streamItem

	if err := c.Decode(&item); err != nil {
		if err == io.EOF {
			_ = c.Close() // #nosec
		}

		return nil, err
	}

	if err := item.Err(); err != nil {
		return nil, err
	}

	var result msg.QueryResult
	if err := jsonUnmarshaler.Unmarshal(bytes.NewReader(item.Result), &result); err != nil {
		return nil, err
	}

	return &result, nil
}
