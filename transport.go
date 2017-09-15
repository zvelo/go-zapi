package zapi

import (
	"crypto/rand"
	"math/big"
	"net/http"

	opentracing "github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/opentracing/opentracing-go/log"
)

var _ http.RoundTripper = (*transport)(nil)

type transport struct {
	*options
}

var chars = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

func randString(n int) string {
	b := make([]rune, n)
	for i := range b {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(chars))))
		if err != nil {
			panic(err)
		}
		b[i] = chars[n.Int64()]
	}
	return string(b)
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

	token, err := t.Token()
	if err != nil {
		clientSpan.LogFields(
			log.String("event", "TokenSource.Token() failed"),
			log.Error(err),
		)
		return nil, err
	}

	token.SetAuthHeader(req)

	err = t.tracer().Inject(
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

	if t.forceTrace {
		req.Header.Set("jaeger-debug-id", randString(32))
	}

	if t.debug {
		debugRequestOut(req)
	}

	res, err := t.transport.RoundTrip(req)
	if err != nil {
		clientSpan.LogFields(
			log.String("event", "error"),
			log.Error(err),
		)
		return nil, err
	}

	if t.debug {
		debugResponse(res)
	}

	ext.HTTPStatusCode.Set(clientSpan, uint16(res.StatusCode))

	return res, nil
}
