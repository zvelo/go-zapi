package zapi

import (
	"context"
	"crypto/tls"
	"io"

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

func (d grpcDialer) Dial(ctx context.Context, opts ...grpc.DialOption) (GRPCClient, error) {
	var tc tls.Config
	if d.options.tlsInsecureSkipVerify {
		tc.InsecureSkipVerify = true
	}

	dialOpts := append([]grpc.DialOption{
		grpc.WithTransportCredentials(credentials.NewTLS(&tc)),
		grpc.WithUnaryInterceptor(
			otgrpc.OpenTracingClientInterceptor(d.options.tracer()),
		),
	}, opts...)

	if d.options.TokenSource != nil {
		dialOpts = append(dialOpts,
			grpc.WithPerRPCCredentials(oauth.TokenSource{
				TokenSource: d.options,
			}),
		)
	}

	conn, err := grpc.DialContext(ctx, d.options.grpcTarget, dialOpts...)
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

func (c grpcClient) QueryV1(ctx context.Context, in *msg.QueryRequests, opts ...grpc.CallOption) (*msg.QueryReplies, error) {
	ctx = c.options.NewOutgoingContext(ctx)
	return c.client.QueryV1(ctx, in, opts...)
}

func (c grpcClient) QueryResultV1(ctx context.Context, in *msg.QueryPollRequest, opts ...grpc.CallOption) (*msg.QueryResult, error) {
	ctx = c.options.NewOutgoingContext(ctx)
	return c.client.QueryResultV1(ctx, in, opts...)
}
