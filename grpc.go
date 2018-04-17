package zapi

import (
	"context"
	"crypto/tls"
	"io"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/grpc-ecosystem/grpc-opentracing/go/otgrpc"

	"golang.org/x/oauth2"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/oauth"
	"google.golang.org/grpc/metadata"

	"zvelo.io/go-zapi/internal/zvelo"
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
	dialOpts := append([]grpc.DialOption{}, opts...)

	dialOpts = append(dialOpts, grpc.WithUnaryInterceptor(
		otgrpc.OpenTracingClientInterceptor(d.options.tracer()),
	))

	if d.options.withoutTLS {
		dialOpts = append(dialOpts, grpc.WithInsecure())
	} else {
		// #nosec
		dialOpts = append(dialOpts, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
			InsecureSkipVerify: d.options.tlsInsecureSkipVerify,
		})))
	}

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
	zvelo.DebugContextOut(ctx, c.options.debug)
	header, opts := grpcHeader(opts...)
	resp, err := c.client.Query(ctx, in, opts...)
	zvelo.DebugMD(c.options.debug, *header)
	return resp, err
}

func (c grpcV1Client) Result(ctx context.Context, in *msg.RequestID, opts ...grpc.CallOption) (*msg.QueryResult, error) {
	zvelo.DebugContextOut(ctx, c.options.debug)
	header, opts := grpcHeader(opts...)
	resp, err := c.client.Result(ctx, in, opts...)
	zvelo.DebugMD(c.options.debug, *header)
	return resp, err
}

func (c grpcV1Client) Suggest(ctx context.Context, in *msg.Suggestion, opts ...grpc.CallOption) (*empty.Empty, error) {
	zvelo.DebugContextOut(ctx, c.options.debug)
	header, opts := grpcHeader(opts...)
	resp, err := c.client.Suggest(ctx, in, opts...)
	zvelo.DebugMD(c.options.debug, *header)
	return resp, err
}

func (c grpcV1Client) Stream(ctx context.Context, in *empty.Empty, opts ...grpc.CallOption) (msg.APIv1_StreamClient, error) {
	if in == nil {
		in = &empty.Empty{}
	}

	zvelo.DebugContextOut(ctx, c.options.debug)
	header, opts := grpcHeader(opts...)
	resp, err := c.client.Stream(ctx, in, opts...)
	zvelo.DebugMD(c.options.debug, *header)
	return resp, err
}

func grpcHeader(opts ...grpc.CallOption) (*metadata.MD, []grpc.CallOption) {
	var header *metadata.MD

	for _, o := range opts {
		if h, ok := o.(grpc.HeaderCallOption); ok {
			header = h.HeaderAddr
		}
	}

	if header == nil {
		header = &metadata.MD{}
		opts = append(opts, grpc.Header(header))
	}

	return header, opts
}
