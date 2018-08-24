package zapi

import (
	"net/http"

	"zvelo.io/go-zapi/internal/zvelo"
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

	if t.TokenSource != nil {
		token, err := t.Token()
		if err != nil {
			return nil, err
		}

		token.SetAuthHeader(req)
	}

	req = zvelo.DebugRequestTiming(t.debug, req)
	zvelo.DebugRequestOut(t.debug, req)

	res, err := t.transport.RoundTrip(req)
	if err != nil {
		return nil, err
	}

	dumpRespBody := true
	if val, ok := req.Context().Value(debugDumpResponseBodyKey).(bool); ok {
		dumpRespBody = val
	}

	zvelo.DebugResponse(t.debug, res, dumpRespBody)

	return res, nil
}
