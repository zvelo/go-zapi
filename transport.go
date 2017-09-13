package zapi

import (
	"net/http"

	opentracing "github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/opentracing/opentracing-go/log"
)

var _ http.RoundTripper = (*transport)(nil)

type transport struct {
	options *options
}

func cloneRequest(r *http.Request) *http.Request {
	// shallow copy of the struct
	r2 := new(http.Request)
	*r2 = *r
	// deep copy of the Header
	r2.Header = make(http.Header, len(r.Header))
	for k, s := range r.Header {
		r2.Header[k] = append([]string(nil), s...)
	}
	return r2
}

func (t *transport) RoundTrip(req *http.Request) (*http.Response, error) {
	req = cloneRequest(req) // per RoundTripper contract

	req.Header.Set("User-Agent", UserAgent)

	var parentCtx opentracing.SpanContext
	if parent := opentracing.SpanFromContext(req.Context()); parent != nil {
		parentCtx = parent.Context()
	}

	clientSpan := opentracing.StartSpan(
		req.URL.Path,
		opentracing.ChildOf(parentCtx),
		ext.SpanKindRPCClient,
	)
	defer clientSpan.Finish()

	ext.Component.Set(clientSpan, "zapi")
	ext.HTTPMethod.Set(clientSpan, req.Method)
	ext.HTTPUrl.Set(clientSpan, req.URL.String())

	tokenSource := t.options.clientCredentials.TokenSource(req.Context())

	token, err := tokenSource.Token()
	if err != nil {
		clientSpan.LogFields(
			log.String("event", "TokenSource.Token() failed"),
			log.Error(err),
		)
		return nil, err
	}

	token.SetAuthHeader(req)

	err = t.options.tracer().Inject(
		clientSpan.Context(),
		opentracing.HTTPHeaders,
		opentracing.HTTPHeadersCarrier(req.Header),
	)
	if err != nil {
		clientSpan.LogFields(
			log.String("event", "Tracer.Inject() failed"),
			log.String("message", err.Error()),
		)
	}

	if t.options.debug {
		debugRequestOut(req)
	}

	res, err := t.options.transport.RoundTrip(req)
	if err != nil {
		clientSpan.LogFields(
			log.String("event", "error"),
			log.Error(err),
		)
		return nil, err
	}

	if t.options.debug {
		debugResponse(res)
	}

	ext.HTTPStatusCode.Set(clientSpan, uint16(res.StatusCode))

	return res, nil
}
