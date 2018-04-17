package zapi

import (
	"net/http"

	"zvelo.io/go-zapi/internal/zvelo"

	opentracing "github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/opentracing/opentracing-go/log"
)

var _ http.RoundTripper = (*transport)(nil)

type transport struct {
	*options
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

type key int

const debugDumpResponseBodyKey key = 0

func (t *transport) RoundTrip(req *http.Request) (*http.Response, error) {
	req = cloneRequest(req) // per RoundTripper contract

	req.Header.Set("User-Agent", UserAgent)

	// prevent "double" gzip decoding
	req.Header.Set("Accept-Encoding", "")

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

	if t.TokenSource != nil {
		token, err := t.Token()
		if err != nil {
			clientSpan.LogFields(
				log.String("event", "TokenSource.Token() failed"),
				log.Error(err),
			)
			return nil, err
		}

		token.SetAuthHeader(req)
	}

	err := t.tracer().Inject(
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

	req = zvelo.DebugRequestTiming(t.debug, req)
	zvelo.DebugRequestOut(t.debug, req)

	res, err := t.transport.RoundTrip(req)
	if err != nil {
		clientSpan.LogFields(
			log.String("event", "error"),
			log.Error(err),
		)
		return nil, err
	}

	dumpRespBody := true
	if val, ok := req.Context().Value(debugDumpResponseBodyKey).(bool); ok {
		dumpRespBody = val
	}

	zvelo.DebugResponse(t.debug, res, dumpRespBody)

	ext.HTTPStatusCode.Set(clientSpan, uint16(res.StatusCode))

	return res, nil
}
