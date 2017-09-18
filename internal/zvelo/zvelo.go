package zvelo

import (
	"crypto/rand"
	"io"
	"math/big"
	"net/http"
	"net/http/httputil"
	"os"
	"strings"

	"github.com/fatih/color"
)

func printDump(w io.Writer, dump []byte, attr color.Attribute, prefix string) {
	write := color.New(attr).FprintfFunc()
	parts := strings.Split(string(dump), "\n")
	for _, line := range parts {
		write(w, "%s%s\n", prefix, line)
	}
	write(w, "\n")
}

// DebugRequest logs incoming http.Requests to stderr
func DebugRequest(req *http.Request) {
	debugHTTP(color.FgYellow, "< ", func() ([]byte, error) { return httputil.DumpRequest(req, true) })
}

// DebugRequestOut logs outgoing http.Requests to stderr
func DebugRequestOut(req *http.Request) {
	debugHTTP(color.FgGreen, "> ", func() ([]byte, error) { return httputil.DumpRequestOut(req, true) })
}

// DebugResponse logs received http.Responses to stderr
func DebugResponse(resp *http.Response) {
	debugHTTP(color.FgYellow, "< ", func() ([]byte, error) { return httputil.DumpResponse(resp, true) })
}

func debugHTTP(attr color.Attribute, prefix string, fn func() ([]byte, error)) {
	dump, err := fn()
	if err != nil {
		_, _ = color.New(color.FgRed).Fprintf(os.Stderr, "%s\n", err)
		return
	}

	printDump(os.Stderr, dump, attr, prefix)
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
