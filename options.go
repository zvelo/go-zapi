package zapi

import (
	"net/http"
	"strings"

	opentracing "github.com/opentracing/opentracing-go"

	"golang.org/x/oauth2/clientcredentials"
)

const (
	UserAgent       = "go-zapi v1"
	DefaultEndpoint = "api.zvelo.com"
	DefaultTokenURL = "https://auth.zvelo.com/oauth2/token"
	DefaultScopes   = "zvelo.dataset"
)

type options struct {
	clientCredentials clientcredentials.Config
	endpoint          string
	debug             bool
	transport         http.RoundTripper
	tracer            func() opentracing.Tracer
}

type Option func(*options)

func defaultScopes() []string {
	return strings.Fields(DefaultScopes)
}

func defaults(clientID, clientSecret string) *options {
	return &options{
		endpoint:  DefaultEndpoint,
		transport: http.DefaultTransport,
		clientCredentials: clientcredentials.Config{
			TokenURL:     DefaultTokenURL,
			Scopes:       defaultScopes(),
			ClientID:     clientID,
			ClientSecret: clientSecret,
		},
		tracer: opentracing.GlobalTracer,
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

func WithTokenURL(val string) Option {
	if val == "" {
		val = DefaultTokenURL
	}

	return func(o *options) {
		o.clientCredentials.TokenURL = val
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

func WithScope(val ...string) Option {
	if len(val) == 0 {
		val = defaultScopes()
	}

	return func(o *options) {
		o.clientCredentials.Scopes = val
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
