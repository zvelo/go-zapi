package userauth

import (
	"context"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/pkg/errors"

	"golang.org/x/oauth2"

	"zvelo.io/go-zapi/internal/zvelo"
)

func ensureCode(u string, code int) {
	resp, err := http.Get(u)
	if err != nil {
		panic(err)
	}

	if resp.StatusCode != code {
		panic(errors.Errorf("unexpected status code for %s (%d != %d)", u, resp.StatusCode, code))
	}
}

func testURLHandler(u string) {
	p, err := url.Parse(u)
	if err != nil {
		panic(err)
	}

	state := p.Query().Get("state")
	redirectURI := p.Query().Get("redirect_uri")
	if p, err = url.Parse(redirectURI); err != nil {
		panic(err)
	}

	// test access_denied

	query := p.Query()
	query.Set("error", "access_denied")
	query.Set("error_description", "some error")
	query.Set("state", state)
	p.RawQuery = query.Encode()

	ensureCode(p.String(), http.StatusUnauthorized)

	// test invalid_request

	query.Set("error", "invalid_request")
	p.RawQuery = query.Encode()
	ensureCode(p.String(), http.StatusBadRequest)

	// test unsupported_response_type

	query.Set("error", "unsupported_response_type")
	p.RawQuery = query.Encode()
	ensureCode(p.String(), http.StatusInternalServerError)

	// test server_error

	query.Set("error", "server_error")
	p.RawQuery = query.Encode()
	ensureCode(p.String(), http.StatusServiceUnavailable)

	// test favicon

	p.Path = "/favicon.ico"
	p.RawQuery = ""

	ensureCode(p.String(), http.StatusUnauthorized)

	// finally, just go to the url

	if _, err = http.Get(u); err != nil {
		panic(err)
	}
}

func TestUserAuth(t *testing.T) {
	ctx := context.Background()
	const clientID, clientSecret = "", ""

	scopes := []string{"scope0", "scope1"}

	handler := zvelo.MockOAuth2Handler()
	srv := httptest.NewServer(handler)

	ts := TokenSource(ctx, clientID, clientSecret,
		WithIgnoreErrors(),
		WithCallbackAddr(""),
		WithCallbackAddr(DefaultCallbackAddr),
		WithDebug(nil),
		WithDebug(ioutil.Discard),
		WithEndpoint(oauth2.Endpoint{}),
		WithEndpoint(handler.Endpoint(srv.URL)),
		WithRedirectURL(""),
		WithRedirectURL(DefaultRedirectURL),
		WithScope(),
		WithScope(scopes...),
		WithoutOpen(),
		WithAuthCodeURLHandler(AuthCodeURLHandlerFunc(testURLHandler)),
	)

	token, err := ts.Token()
	if err != nil {
		t.Fatal(err)
	}

	if !token.Valid() {
		t.Fatal("invalid token")
	}
}
