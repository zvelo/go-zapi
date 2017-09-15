package zapi

import (
	"context"
	"io"
	"net"
	"net/url"
	"strconv"
	"strings"

	"golang.org/x/oauth2"

	"github.com/grpc-ecosystem/grpc-opentracing/go/otgrpc"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/oauth"

	"zvelo.io/msg"
)

type GRPCClient interface {
	msg.APIClient
	io.Closer
}

type grpcClient struct {
	msg.APIClient
	io.Closer
}

type GRPCDialer interface {
	Dial(context.Context, ...grpc.DialOption) (GRPCClient, error)
}

type grpcDialer struct {
	options *options
}

func grpcEndpoint(val string) (string, error) {
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
	endpoint, err := grpcEndpoint(d.options.endpoint)
	if err != nil {
		return nil, err
	}

	conn, err := grpc.DialContext(
		ctx,
		endpoint,
		append(opts,
			grpc.WithTransportCredentials(credentials.NewTLS(nil)),
			grpc.WithPerRPCCredentials(oauth.TokenSource{
				TokenSource: d.options,
			}),
			grpc.WithUnaryInterceptor(
				otgrpc.OpenTracingClientInterceptor(d.options.tracer()),
			),
		)...,
	)

	if err != nil {
		return nil, err
	}

	return grpcClient{
		Closer:    conn,
		APIClient: msg.NewAPIClient(conn),
	}, nil
}

func NewGRPC(ts oauth2.TokenSource, opts ...Option) GRPCDialer {
	o := defaults(ts)
	for _, opt := range opts {
		opt(o)
	}

	return grpcDialer{options: o}
}
