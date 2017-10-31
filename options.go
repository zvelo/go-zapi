package zapi

import (
	"context"
	"crypto/tls"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"

	opentracing "github.com/opentracing/opentracing-go"

	"golang.org/x/oauth2"

	"google.golang.org/grpc/metadata"

	"zvelo.io/go-zapi/internal/zvelo"
)

// UserAgent is the user agent that will be provided by the RESTv1Client. It can
// be overridden by providing a custom transport using the WithTransport Option.
const UserAgent = "go-zapi v1"

// DefaultAddr is used by both GRPCv1Client and RESTv1Client as the default
// address:port for all zveloAPI calls. It can be overridden using the WithAddr
// Option.
const DefaultAddr = "api.zvelo.com"

type options struct {
	oauth2.TokenSource
	grpcTarget            string
	restBaseURL           *url.URL
	debug                 io.Writer
	transport             http.RoundTripper
	tracerFunc            func() opentracing.Tracer
	forceTrace            bool
	tlsInsecureSkipVerify bool
}

// An Option is used to configure different parts of this package. Not every
// Option is useful with every function that takes Options.
type Option func(*options)

func defaults(ts oauth2.TokenSource) *options {
	o := options{
		TokenSource: ts,
		transport:   http.DefaultTransport,
		tracerFunc:  opentracing.GlobalTracer,
		debug:       ioutil.Discard,
	}
	WithAddr(DefaultAddr)(&o)
	return &o
}

func (o options) tracer() opentracing.Tracer {
	if tracer := o.tracerFunc(); tracer != nil {
		return tracer
	}

	return opentracing.NoopTracer{}
}

func (o options) restURL(dir string, elem ...string) string {
	u := *o.restBaseURL

	parts := []string{u.Path, dir}
	parts = append(parts, elem...)
	u.Path = path.Join(parts...)

	return u.String()
}

func (o options) NewOutgoingContext(ctx context.Context) context.Context {
	if !o.forceTrace {
		return ctx
	}

	var md metadata.MD
	if oc, ok := metadata.FromOutgoingContext(ctx); ok {
		md = oc.Copy()
	}

	return metadata.NewOutgoingContext(ctx, metadata.Join(
		md,
		metadata.Pairs("jaeger-debug-id", zvelo.RandString(32)),
	))
}

// WithForceTrace returns an Option that will cause all requests to be traced
// by the api server
func WithForceTrace() Option {
	return func(o *options) {
		o.forceTrace = true
	}
}

// WithTransport returns an Option that will cause all requests from the
// RESTv1Client to be processed by the given http.RoundTripper. If not
// specified, http.DefaultTransport will be used.
func WithTransport(val http.RoundTripper) Option {
	if val == nil {
		val = http.DefaultTransport
	}

	return func(o *options) {
		o.transport = val
		if o.tlsInsecureSkipVerify {
			WithTLSInsecureSkipVerify()(o)
		}
	}
}

// WithTLSInsecureSkipVerify returns an Option that disables certificate chain
// and host name verification of the connection to zveloAPI. This should
// only be used for testing, e.g. with mocks.
func WithTLSInsecureSkipVerify() Option {
	return func(o *options) {
		o.tlsInsecureSkipVerify = true
		if t, ok := o.transport.(*http.Transport); ok {
			t.TLSClientConfig = &tls.Config{
				InsecureSkipVerify: true,
			}
		}
	}
}

// WithTracer returns an Option that will cause requests to be instrumented by
// the given tracer. If not specified, opentracing.GlobalTracer will be used.
func WithTracer(val opentracing.Tracer) Option {
	return func(o *options) {
		if val == nil {
			o.tracerFunc = opentracing.GlobalTracer
			return
		}

		o.tracerFunc = func() opentracing.Tracer {
			return val
		}
	}
}

// WithDebug returns an Option that will cause requests from the RESTv1Client
// and callbacks processed by the CallbackHandler to emit debug logs to the
// writer
func WithDebug(val io.Writer) Option {
	if val == nil {
		val = ioutil.Discard
	}

	return func(o *options) {
		o.debug = val
	}
}

// WithAddr returns an Option that overrides the default address:port for all
// zveloAPI requests
func WithAddr(val string) Option {
	if val == "" {
		val = DefaultAddr
	}

	if !strings.Contains(val, "://") {
		val = "https://" + val
	}

	p, err := url.Parse(val)
	if err != nil {
		panic(err)
	}

	port := p.Port()
	if port == "" {
		o, err := net.LookupPort("tcp", p.Scheme)
		if err != nil {
			panic(err)
		}
		port = strconv.Itoa(o)
	}

	return func(o *options) {
		o.grpcTarget = net.JoinHostPort(p.Hostname(), port)
		o.restBaseURL = p
	}
}
