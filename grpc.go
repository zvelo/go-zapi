package zapi

import (
	"context"
	"io"
	"net"
	"net/url"
	"strconv"
	"strings"

	"github.com/grpc-ecosystem/grpc-opentracing/go/otgrpc"

	"golang.org/x/oauth2"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/oauth"

	"zvelo.io/msg"
)

// A GRPCClient implements msg.APIClient as well as an io.Closer that, if
// closed, will close the underlying gRPC connection.
type GRPCClient interface {
	msg.APIClient
	io.Closer
}

type grpcClient struct {
	options *options
	client  msg.APIClient
	io.Closer
}

// A GRPCDialer is used to simplify connecting to zveloAPI with the correct
// options. grpc DialOptions will override the defaults.
type GRPCDialer interface {
	Dial(context.Context, ...grpc.DialOption) (GRPCClient, error)
}

type grpcDialer struct {
	options *options
}

func grpcTarget(val string) (string, error) {
	if !strings.Contains(val, "://") {
		val = "https://" + val
	}

	p, err := url.Parse(val)
	if err != nil {
		return "", err
	}

	port := p.Port()
	if port == "" {
		o, err := net.LookupPort("tcp", p.Scheme)
		if err != nil {
			return "", err
		}
		port = strconv.Itoa(o)
	}

	return net.JoinHostPort(p.Hostname(), port), nil
}

func (d grpcDialer) Dial(ctx context.Context, opts ...grpc.DialOption) (GRPCClient, error) {
	target, err := grpcTarget(d.options.host)
	if err != nil {
		return nil, err
	}

	conn, err := grpc.DialContext(
		ctx,
		target,
		append([]grpc.DialOption{
			grpc.WithTransportCredentials(credentials.NewTLS(nil)),
			grpc.WithPerRPCCredentials(oauth.TokenSource{
				TokenSource: d.options,
			}),
			grpc.WithUnaryInterceptor(
				otgrpc.OpenTracingClientInterceptor(d.options.tracer()),
			),
		}, opts...)...,
	)

	if err != nil {
		return nil, err
	}

	return grpcClient{
		Closer:  conn,
		client:  msg.NewAPIClient(conn),
		options: d.options,
	}, nil
}

// NewGRPC returns a properly configured GRPCDialer
func NewGRPC(ts oauth2.TokenSource, opts ...Option) GRPCDialer {
	o := defaults(ts)
	for _, opt := range opts {
		opt(o)
	}

	return grpcDialer{options: o}
}

func (c grpcClient) QueryURLV1(ctx context.Context, in *msg.QueryURLRequests, opts ...grpc.CallOption) (*msg.QueryReplies, error) {
	ctx = c.options.NewOutgoingContext(ctx)
	return c.client.QueryURLV1(ctx, in, opts...)
}

func (c grpcClient) QueryURLResultV1(ctx context.Context, in *msg.QueryPollRequest, opts ...grpc.CallOption) (*msg.QueryResult, error) {
	ctx = c.options.NewOutgoingContext(ctx)
	return c.client.QueryContentResultV1(ctx, in, opts...)
}

func (c grpcClient) QueryContentV1(ctx context.Context, in *msg.QueryContentRequests, opts ...grpc.CallOption) (*msg.QueryReplies, error) {
	ctx = c.options.NewOutgoingContext(ctx)
	return c.client.QueryContentV1(ctx, in, opts...)
}

func (c grpcClient) QueryContentResultV1(ctx context.Context, in *msg.QueryPollRequest, opts ...grpc.CallOption) (*msg.QueryResult, error) {
	ctx = c.options.NewOutgoingContext(ctx)
	return c.client.QueryContentResultV1(ctx, in, opts...)
}
