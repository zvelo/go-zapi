package zapi

import (
	"crypto/tls"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"
	"strings"

	"golang.org/x/oauth2"
)

// UserAgent is the user agent that will be provided by the RESTv1Client. It can
// be overridden by providing a custom transport using the WithTransport Option.
const UserAgent = "go-zapi v1"

// DefaultRestBaseURL is used by the RESTv1Client as the default base URL for all
// zveloAPI calls. It can be overridden using the WithRestBaseURL Option.
const DefaultRestBaseURL = "https://api.zvelo.com/"

// DefaultGrpcTarget is used by the GRPCv1Client as the default
// scheme://authority/endpoint_name for all zveloAPI calls. It can be
// overridden using the WithGrpcTarget Option.
const DefaultGrpcTarget = "dns:///api.zvelo.com"

type options struct {
	oauth2.TokenSource
	grpcTarget            string
	restBaseURL           *url.URL
	debug                 io.Writer
	noHTTP2               bool
	transport             http.RoundTripper
	tlsInsecureSkipVerify bool
	withoutTLS            bool
}

// An Option is used to configure different parts of this package. Not every
// Option is useful with every function that takes Options.
type Option func(*options)

func defaults(ts oauth2.TokenSource) *options {
	o := options{
		TokenSource: ts,
		transport:   http.DefaultTransport,
		debug:       ioutil.Discard,
	}
	WithRestBaseURL(DefaultRestBaseURL)(&o)
	WithGrpcTarget(DefaultGrpcTarget)(&o)
	return &o
}

func (o options) restURL(dir string, elem ...string) string {
	u := *o.restBaseURL

	parts := []string{u.Path, dir}
	parts = append(parts, elem...)
	u.Path = path.Join(parts...)

	return u.String()
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
			// #nosec
			t.TLSClientConfig = &tls.Config{
				InsecureSkipVerify: true,
			}
		}
	}
}

// WithoutTLS disables TLS when connecting to zveloAPI
func WithoutTLS() Option {
	return func(o *options) {
		o.withoutTLS = true
		o.restBaseURL.Scheme = "http"
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

// WithoutHTTP2 disables the http/2 client for REST queries
func WithoutHTTP2() Option {
	return func(o *options) {
		o.noHTTP2 = true
	}
}

// WithRestBaseURL returns an Option that overrides the default base URL for all
// zveloAPI requests
func WithRestBaseURL(val string) Option {
	if val == "" {
		val = DefaultRestBaseURL
	}

	return func(o *options) {
		if !strings.Contains(val, "://") {
			if o.restBaseURL != nil {
				val = o.restBaseURL.Scheme + "://" + val
			} else {
				val = "https://" + val
			}
		}

		if p, err := url.Parse(val); err == nil {
			if p.Path == "" {
				p.Path = "/"
			}
			o.restBaseURL = p
		}
	}
}

// WithGrpcTarget returns an Option that overrides the default gRPC target for
// all zveloAPI requests
func WithGrpcTarget(val string) Option {
	if val == "" {
		val = DefaultGrpcTarget
	}

	return func(o *options) {
		o.grpcTarget = val
	}
}
