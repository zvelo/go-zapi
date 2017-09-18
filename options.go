package zapi

import (
	"context"
	"net/http"

	"google.golang.org/grpc/metadata"
	"zvelo.io/go-zapi/internal/zvelo"

	opentracing "github.com/opentracing/opentracing-go"

	"golang.org/x/oauth2"
)

// UserAgent is the user agent that will be provided by the RESTClient. It can
// be overridden by providing a custom transport using the WithTransport Option.
const UserAgent = "go-zapi v1"

// DefaultHost is used by both GRPCClient and RESTClient as the default
// host for all zveloAPI calls. It can be overridden using the WithHost
// Option.
const DefaultHost = "api.zvelo.com"

type options struct {
	oauth2.TokenSource
	host       string
	debug      bool
	transport  http.RoundTripper
	tracer     func() opentracing.Tracer
	forceTrace bool
}

// An Option is used to configure different parts of this package. Not every
// Option is useful with every function that takes Options.
type Option func(*options)

func defaults(ts oauth2.TokenSource) *options {
	return &options{
		TokenSource: ts,
		host:        DefaultHost,
		transport:   http.DefaultTransport,
		tracer:      opentracing.GlobalTracer,
	}
}

func (o options) NewOutgoingContext(ctx context.Context) context.Context {
	if !o.forceTrace {
		return ctx
	}

	md := metadata.Pairs("jaeger-debug-id", zvelo.RandString(32))

	if oc, ok := metadata.FromOutgoingContext(ctx); ok {
		md = metadata.Join(md, oc.Copy())
	}

	return metadata.NewOutgoingContext(ctx, md)
}

// WithForceTrace returns an Option that will cause all requests to be traced
// by the api server
func WithForceTrace() Option {
	return func(o *options) {
		o.forceTrace = true
	}
}

// WithTransport returns an Option that will cause all requests from the
// RESTClient to be processed by the given http.RoundTripper. If not specified,
// http.DefaultTransport will be used.
func WithTransport(val http.RoundTripper) Option {
	if val == nil {
		val = http.DefaultTransport
	}

	return func(o *options) {
		o.transport = val
	}
}

// WithTracer returns an Option that will cause requests to be instrumented by
// the given tracer. If not specified, opentracing.GlobalTracer will be used.
func WithTracer(val opentracing.Tracer) Option {
	return func(o *options) {
		if val == nil {
			o.tracer = opentracing.GlobalTracer
			return
		}

		o.tracer = func() opentracing.Tracer {
			return val
		}
	}
}

// WithDebug returns an Option that will cause requests from the RESTClient and
// callbacks processed by the CallbackHandler to emit debug logs to stderr
func WithDebug() Option {
	return func(o *options) {
		o.debug = true
	}
}

// WithHost returns an Option that overrides the default host for all
// zveloAPI requests
func WithHost(val string) Option {
	if val == "" {
		val = DefaultHost
	}

	return func(o *options) {
		o.host = val
	}
}
