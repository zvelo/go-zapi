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

// A GRPCv1Client implements msg.APIv1Client as well as an io.Closer that, if
// closed, will close the underlying gRPC connection.
type GRPCv1Client interface {
	msg.APIv1Client
	io.Closer
}

type grpcV1Client struct {
	options *options
	client  msg.APIv1Client
	io.Closer
}

// A GRPCv1Dialer is used to simplify connecting to zveloAPI with the correct
// options. grpc DialOptions will override the defaults.
type GRPCv1Dialer interface {
	Dial(context.Context, ...grpc.DialOption) (GRPCv1Client, error)
}

type grpcV1Dialer struct {
	options *options
}

func (d grpcV1Dialer) Dial(ctx context.Context, opts ...grpc.DialOption) (GRPCv1Client, error) {
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

	return grpcV1Client{
		Closer:  conn,
		client:  msg.NewAPIv1Client(conn),
		options: d.options,
	}, nil
}

// NewGRPCv1 returns a properly configured GRPCv1Dialer
func NewGRPCv1(ts oauth2.TokenSource, opts ...Option) GRPCv1Dialer {
	o := defaults(ts)
	for _, opt := range opts {
		opt(o)
	}

	return grpcV1Dialer{options: o}
}

func (c grpcV1Client) Query(ctx context.Context, in *msg.QueryRequests, opts ...grpc.CallOption) (*msg.QueryReplies, error) {
	ctx = c.options.NewOutgoingContext(ctx)
	return c.client.Query(ctx, in, opts...)
}

func (c grpcV1Client) Result(ctx context.Context, in *msg.RequestID, opts ...grpc.CallOption) (*msg.QueryResult, error) {
	ctx = c.options.NewOutgoingContext(ctx)
	return c.client.Result(ctx, in, opts...)
}
