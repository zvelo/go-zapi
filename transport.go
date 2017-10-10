package zapi

import (
	"compress/gzip"
	"io"
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

func (t *transport) RoundTrip(req *http.Request) (*http.Response, error) {
	req = cloneRequest(req) // per RoundTripper contract

	req.Header.Set("User-Agent", UserAgent)
	req.Header.Set("Accept-Encoding", "gzip")

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

	if t.forceTrace {
		req.Header.Set("jaeger-debug-id", zvelo.RandString(32))
	}

	zvelo.DebugRequestOut(t.debug, req)

	res, err := t.transport.RoundTrip(req)
	if err != nil {
		clientSpan.LogFields(
			log.String("event", "error"),
			log.Error(err),
		)
		return nil, err
	}

	if res.Header.Get("Content-Encoding") == "gzip" {
		r, err := gzip.NewReader(res.Body)
		if err != nil {
			return nil, err
		}

		res.ContentLength = -1
		res.Header.Del("Content-Encoding")
		res.Header.Del("Content-Length")

		res.Body = gzipReader{
			Reader: r,
			src:    res.Body,
		}
	}

	zvelo.DebugResponse(t.debug, res)

	ext.HTTPStatusCode.Set(clientSpan, uint16(res.StatusCode))

	return res, nil
}

type gzipReader struct {
	*gzip.Reader
	src io.ReadCloser
}

func (r gzipReader) Close() error {
	if err := r.Reader.Close(); err != nil {
		return err
	}

	return r.src.Close()
}
