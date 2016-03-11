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
	endpoint           *url.URL
	Username, Password string
	Token              string
	HTTPClient         Doer
	Debug              bool
	UserAgent          string
}

func New() *Client {
	ret := &Client{
		HTTPClient: &http.Client{},
		UserAgent:  DefaultUserAgent,
	}

	_ = ret.SetEndpoint(DefaultEndpoint)

	return ret
}

func (c *Client) SetEndpoint(endpointURL string) error {
	if len(endpointURL) == 0 {
		return ErrMissingEndpoint
	}

	if strings.Index(endpointURL, "://") == -1 {
		endpointURL = "http://" + endpointURL
	}

	endpoint, err := url.Parse(endpointURL)
	if err != nil {
		return ErrInvalidEndpoint
	}

	endpoint.Path = "/" + APIVersion

	c.endpoint = endpoint

	return nil
}

func (c Client) endpointURL(p string) *url.URL {
	ret := *c.endpoint
	ret.Path = path.Join(ret.Path, p)

	if len(c.Token) > 0 {
		ret.RawQuery = url.Values{"access_token": {c.Token}}.Encode()
	}

	return &ret
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
