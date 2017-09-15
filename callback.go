package zapi

import (
	"encoding/json"
	"net/http"

	"zvelo.io/msg"
)

type Handler interface {
	Handle(*msg.QueryResult)
}

type HandlerFunc func(*msg.QueryResult)

func (f HandlerFunc) Handle(in *msg.QueryResult) {
	f(in)
}

var _ Handler = (*HandlerFunc)(nil)

func CallbackHandler(h Handler, opts ...Option) http.Handler {
	o := defaults(nil)
	for _, opt := range opts {
		opt(o)
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if o.debug {
			debugRequest(r)
		}

		var result msg.QueryResult
		if err := json.NewDecoder(r.Body).Decode(&result); err == nil {
			h.Handle(&result)
		}
	})
}
