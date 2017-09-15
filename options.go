package zapi

import (
	"net/http"

	opentracing "github.com/opentracing/opentracing-go"

	"golang.org/x/oauth2"
)

const (
	UserAgent       = "go-zapi v1"
	DefaultEndpoint = "api.zvelo.com"
)

type options struct {
	oauth2.TokenSource
	endpoint   string
	debug      bool
	transport  http.RoundTripper
	tracer     func() opentracing.Tracer
	forceTrace bool
}

type Option func(*options)

func defaults(ts oauth2.TokenSource) *options {
	return &options{
		TokenSource: ts,
		endpoint:    DefaultEndpoint,
		transport:   http.DefaultTransport,
		tracer:      opentracing.GlobalTracer,
	}
}

func WithForceTrace() Option {
	return func(o *options) {
		o.forceTrace = true
	}
}

func WithTransport(val http.RoundTripper) Option {
	if val == nil {
		val = http.DefaultTransport
	}

	return func(o *options) {
		o.transport = val
	}
}

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

func WithDebug() Option {
	return func(o *options) {
		o.debug = true
	}
}

func WithEndpoint(val string) Option {
	if val == "" {
		val = DefaultEndpoint
	}

	return func(o *options) {
		o.endpoint = val
	}
}
