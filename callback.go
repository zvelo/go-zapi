package zapi

import (
	"net/http"

	"zvelo.io/go-zapi/internal/zvelo"
	"zvelo.io/msg"
)

// A Handler responds to a zveloAPI callback
type Handler interface {
	Handle(*msg.QueryResult)
}

// The HandlerFunc type is an adapter to allow the use of ordinary functions as
// zveloAPI handlers. If f is a function with the appropriate signature,
// HandlerFunc(f) is a Handler that calls f.
type HandlerFunc func(*msg.QueryResult)

// Handle calls f(in)
func (f HandlerFunc) Handle(in *msg.QueryResult) {
	f(in)
}

var _ Handler = (*HandlerFunc)(nil)

// CallbackHandler returns an http.Handler that can be used with an http.Server
// to receive and process zveloAPI callbacks
func CallbackHandler(h Handler, opts ...Option) http.Handler {
	o := defaults(nil)
	for _, opt := range opts {
		opt(o)
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		zvelo.DebugRequest(o.debug, r)

		var result msg.QueryResult
		if err := jsonUnmarshaler.Unmarshal(r.Body, &result); err == nil {
			h.Handle(&result)
		}
	})
}
