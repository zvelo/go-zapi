package zvelo

import (
	"context"
	"crypto/rand"
	"io"
	"math/big"
	"net/http"
	"net/http/httputil"
	"strings"

	"github.com/fatih/color"
	"google.golang.org/grpc/metadata"
)

// DebugRequest logs incoming http.Requests to w
func DebugRequest(w io.Writer, req *http.Request) {
	debugHTTP(w, color.FgYellow, "< ", func() ([]byte, error) { return httputil.DumpRequest(req, true) })
}

// DebugRequestOut logs outgoing http.Requests to w
func DebugRequestOut(w io.Writer, req *http.Request) {
	debugHTTP(w, color.FgGreen, "> ", func() ([]byte, error) { return httputil.DumpRequestOut(req, true) })
}

// DebugResponse logs received http.Responses to w
func DebugResponse(w io.Writer, resp *http.Response, body bool) {
	debugHTTP(w, color.FgYellow, "< ", func() ([]byte, error) { return httputil.DumpResponse(resp, body) })
}

func debugHTTP(w io.Writer, attr color.Attribute, prefix string, fn func() ([]byte, error)) {
	if w == nil {
		return
	}

	dump, err := fn()
	if err != nil {
		_, _ = color.New(color.FgRed).Fprintf(w, "%s\n", err) // #nosec
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

// DebugContextOut logs outgoing metadata headers to w
func DebugContextOut(ctx context.Context, w io.Writer) {
	md, ok := metadata.FromOutgoingContext(ctx)
	if !ok {
		return
	}

	debugMD(w, color.FgGreen, "> ", md)
}

// DebugMD logs received metadata headers to w
func DebugMD(w io.Writer, md metadata.MD) {
	debugMD(w, color.FgYellow, "< ", md)
}

func debugMD(w io.Writer, attr color.Attribute, prefix string, md metadata.MD) {
	if w == nil {
		return
	}

	write := color.New(attr).FprintfFunc()

	for k, vs := range md {
		for _, v := range vs {
			write(w, "%s%s: %s\n", prefix, k, v)
		}
	}
}
