package callback

import (
	"bytes"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/gogo/protobuf/jsonpb"

	"zvelo.io/go-zapi/internal/zvelo"
	"zvelo.io/httpsig"
	msg "zvelo.io/msg/msgpb"
)

var jsonUnmarshaler = jsonpb.Unmarshaler{
	AllowUnknownFields: true,
}

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

func drainBody(r *http.Request) (io.Reader, error) {
	var buf bytes.Buffer

	if _, err := buf.ReadFrom(r.Body); err != nil {
		return nil, err
	}

	if err := r.Body.Close(); err != nil {
		return nil, err
	}

	r.Body = ioutil.NopCloser(bytes.NewReader(buf.Bytes()))

	return &buf, nil
}

// Middleware returns an http.Handler that can be used with an http.Server
// to receive and process zveloAPI callbacks. If getter is not nil, it will be
// used to validate HTTP Signatures on the incoming request.
func Middleware(getter httpsig.KeyGetter, h Handler, debug io.Writer) http.Handler {
	var handler http.Handler

	handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Body == nil || r.Body == http.NoBody {
			http.Error(w, "no body", http.StatusBadRequest)
			return
		}

		rdr, err := drainBody(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		var result msg.QueryResult
		if err := jsonUnmarshaler.Unmarshal(rdr, &result); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		h.Handle(w, r, &result)
	})

	if getter != nil {
		handler = httpsig.Middleware(httpsig.SignatureHeader, getter, handler)
	}

	return zvelo.DebugHandler(debug, handler)
}
