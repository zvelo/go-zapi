package zapi // import "zvelo.io/go-zapi"

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"os"
	"strings"
)

func printDump(w io.Writer, dump []byte, prefix string) {
	parts := strings.Split(string(dump), "\n")
	for _, line := range parts {
		fmt.Fprintf(w, "%s%s\n", prefix, line)
	}
	fmt.Fprintf(w, "\n")
}

func debugRequest(req *http.Request) {
	debugHTTP("< ", func() ([]byte, error) { return httputil.DumpRequest(req, true) })
}

func debugRequestOut(req *http.Request) {
	debugHTTP("> ", func() ([]byte, error) { return httputil.DumpRequestOut(req, true) })
}

func debugResponse(resp *http.Response) {
	debugHTTP("< ", func() ([]byte, error) { return httputil.DumpResponse(resp, true) })
}

func debugHTTP(prefix string, fn func() ([]byte, error)) {
	dump, err := fn()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		return
	}

	printDump(os.Stderr, dump, prefix)
}
