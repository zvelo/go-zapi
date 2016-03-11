package zapi

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path"
	"strings"
)

const (
	APIVersion       = "v1"
	DefaultEndpoint  = "https://api.zvelo.com"
	DefaultUserAgent = "go-zapi v1"
	tokenPath        = "auth/token"
	urlPath          = "queries/url"
)

type Doer interface {
	Do(*http.Request) (*http.Response, error)
}

type Client struct {
	Endpoint           string
	Username, Password string
	Token              string
	HTTPClient         Doer
	Debug              bool
	UserAgent          string
}

func New() *Client {
	ret := &Client{
		UserAgent:  DefaultUserAgent,
		Endpoint:   DefaultEndpoint,
		HTTPClient: &http.Client{},
	}

	return ret
}

func (c Client) endpointURL(p string) (*url.URL, error) {
	if strings.Index(c.Endpoint, "://") == -1 {
		c.Endpoint = "https://" + c.Endpoint
	}

	ret, err := url.Parse(c.Endpoint)
	if err != nil {
		return nil, err
	}

	ret.Path = path.Join(ret.Path, APIVersion, p)

	if len(c.Token) > 0 {
		ret.RawQuery = url.Values{"access_token": {c.Token}}.Encode()
	}

	return ret, nil
}

func printDump(w io.Writer, dump []byte, prefix string) {
	parts := strings.Split(string(dump), "\n")
	for _, line := range parts {
		fmt.Fprintf(w, "%s%s\n", prefix, line)
	}
	fmt.Fprintf(w, "\n")
}

func (c Client) debugRequest(req *http.Request) {
	c.debugHTTP("> ", func() ([]byte, error) { return httputil.DumpRequestOut(req, true) })
}

func (c Client) debugResponse(resp *http.Response) {
	c.debugHTTP("< ", func() ([]byte, error) { return httputil.DumpResponse(resp, true) })
}

func (c Client) debugHTTP(prefix string, fn func() ([]byte, error)) {
	if !c.Debug {
		return
	}

	dump, err := fn()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		return
	}

	printDump(os.Stderr, dump, prefix)
}
