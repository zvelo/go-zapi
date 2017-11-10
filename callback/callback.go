package callback

import (
	"net/http"

	"github.com/gogo/protobuf/jsonpb"

	"zvelo.io/httpsig"
	"zvelo.io/msg"
)

var jsonUnmarshaler jsonpb.Unmarshaler

// A Handler responds to a zveloAPI callback
type Handler interface {
	Handle(http.ResponseWriter, *http.Request, *msg.QueryResult)
}

// The HandlerFunc type is an adapter to allow the use of ordinary functions as
// zveloAPI handlers. If f is a function with the appropriate signature,
// HandlerFunc(f) is a Handler that calls f.
type HandlerFunc func(http.ResponseWriter, *http.Request, *msg.QueryResult)

// Handle calls f(w, r, in)
func (f HandlerFunc) Handle(w http.ResponseWriter, r *http.Request, in *msg.QueryResult) {
	f(w, r, in)
}

var _ Handler = (*HandlerFunc)(nil)

// Middleware returns an http.Handler that can be used with an http.Server
// to receive and process zveloAPI callbacks. If getter is not nil, it will be
// used to validate HTTP Signatures on the incoming request.
func Middleware(getter httpsig.KeyGetter, h Handler) http.Handler {
	handler := http.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var result msg.QueryResult
		if err := jsonUnmarshaler.Unmarshal(r.Body, &result); err == nil {
			h.Handle(w, r, &result)
		}
	}))

	if getter != nil {
		handler = httpsig.Middleware(httpsig.SignatureHeader, getter, handler)
	}

	return handler
}
