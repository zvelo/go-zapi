package zvelo

import (
	"crypto/rand"
	"io"
	"math/big"
	"net/http"
	"net/http/httputil"
	"strings"

	"github.com/fatih/color"
)

// DebugRequest logs incoming http.Requests to stderr
func DebugRequest(w io.Writer, req *http.Request) {
	debugHTTP(w, color.FgYellow, "< ", func() ([]byte, error) { return httputil.DumpRequest(req, true) })
}

// DebugRequestOut logs outgoing http.Requests to stderr
func DebugRequestOut(w io.Writer, req *http.Request) {
	debugHTTP(w, color.FgGreen, "> ", func() ([]byte, error) { return httputil.DumpRequestOut(req, true) })
}

// DebugResponse logs received http.Responses to stderr
func DebugResponse(w io.Writer, resp *http.Response, body bool) {
	debugHTTP(w, color.FgYellow, "< ", func() ([]byte, error) { return httputil.DumpResponse(resp, body) })
}

func debugHTTP(w io.Writer, attr color.Attribute, prefix string, fn func() ([]byte, error)) {
	if w == nil {
		return
	}

	dump, err := fn()
	if err != nil {
		_, _ = color.New(color.FgRed).Fprintf(w, "%s\n", err)
		return
	}

	write := color.New(attr).FprintfFunc()
	parts := strings.Split(string(dump), "\n")
	for _, line := range parts {
		write(w, "%s%s\n", prefix, line)
	}
}

var chars = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

// RandString returns a random string of length n
func RandString(n int) string {
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

// DebugHandler returns an http.Handler that debugs incoming requests to w. next
// is called after writing to the debug writer.
func DebugHandler(w io.Writer, next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		DebugRequest(w, req)
		next.ServeHTTP(rw, req)
	})
}
