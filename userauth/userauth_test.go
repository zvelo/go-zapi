package userauth

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"sync"
	"testing"
	"time"

	"golang.org/x/oauth2"

	"zvelo.io/go-zapi/internal/zvelo"
)

type mockOAuth2 struct {
	reqs sync.Map
}

func NewMockOAuth2() http.Handler {
	m := mockOAuth2{}

	mux := http.NewServeMux()
	mux.HandleFunc("/oauth2/auth", m.authHandler)
	mux.HandleFunc("/oauth2/token", m.tokenHandler)

	return mux
}

func (m *mockOAuth2) authHandler(w http.ResponseWriter, r *http.Request) {
	redirectURI := r.URL.Query().Get("redirect_uri")
	p, err := url.Parse(redirectURI)
	if err != nil {
		log.Fatal(err)
	}

	code := zvelo.RandString(32)
	m.reqs.Store(code, struct{}{})

	values := p.Query()
	values.Set("code", code)
	values.Set("state", r.URL.Query().Get("state"))
	p.RawQuery = values.Encode()

	resp, err := http.Get(p.String())
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		log.Fatalf("unexpected response code %s(%d)", http.StatusText(resp.StatusCode), resp.StatusCode)
	}
}

type expirationTime time.Duration

func (e expirationTime) MarshalJSON() ([]byte, error) {
	return []byte(strconv.FormatInt(int64(time.Duration(e).Seconds()), 10)), nil
}

type tokenJSON struct {
	AccessToken  string         `json:"access_token"`
	TokenType    string         `json:"token_type"`
	RefreshToken string         `json:"refresh_token"`
	ExpiresIn    expirationTime `json:"expires_in"`
}

func (m *mockOAuth2) tokenHandler(w http.ResponseWriter, r *http.Request) {
	code := r.FormValue("code")
	if _, ok := m.reqs.Load(code); !ok {
		log.Fatal("unexpceted code")
	}
	m.reqs.Delete(code)
	w.Header().Set("Content-Type", "application/json")

	err := json.NewEncoder(w).Encode(tokenJSON{
		AccessToken: zvelo.RandString(32),
		TokenType:   "Bearer",
		ExpiresIn:   expirationTime(time.Hour),
	})
	if err != nil {
		log.Fatal(err)
	}
}

func testURLHandler(u string) {
	resp, err := http.Get(u)
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		log.Fatalf("unexpected response code %s(%d)", http.StatusText(resp.StatusCode), resp.StatusCode)
	}
}

func TestUserAuth(t *testing.T) {
	ctx := context.Background()
	const clientID, clientSecret = "", ""

	scopes := []string{"scope0", "scope1"}

	srv := httptest.NewServer(NewMockOAuth2())

	ts := TokenSource(ctx, clientID, clientSecret,
		WithCallbackAddr(""),
		WithCallbackAddr(DefaultCallbackAddr),
		WithDebug(nil),
		WithDebug(ioutil.Discard),
		WithEndpoint(oauth2.Endpoint{}),
		WithEndpoint(oauth2.Endpoint{
			AuthURL:  srv.URL + "/oauth2/auth",
			TokenURL: srv.URL + "/oauth2/token",
		}),
		WithRedirectURL(""),
		WithRedirectURL(DefaultRedirectURL),
		WithScope(),
		WithScope(scopes...),
		WithoutOpen(),
		WithURLFunc(testURLHandler),
	)

	token, err := ts.Token()
	if err != nil {
		t.Fatal(err)
	}

	if !token.Valid() {
		t.Fatal("invalid token")
	}
}
