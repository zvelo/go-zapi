package zvelo

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"

	"golang.org/x/oauth2"
)

type mockOAuth2 struct {
	reqs sync.Map
}

type mockOAuth2Handler struct {
	http.Handler
}

func (m mockOAuth2Handler) Endpoint(u string) oauth2.Endpoint {
	return oauth2.Endpoint{
		AuthURL:  u + "/oauth2/auth",
		TokenURL: u + "/oauth2/token",
	}
}

type Handler interface {
	http.Handler
	Endpoint(url string) oauth2.Endpoint
}

func MockOAuth2Handler() Handler {
	var m mockOAuth2

	mux := http.NewServeMux()
	mux.HandleFunc("/oauth2/auth", m.authHandler)
	mux.HandleFunc("/oauth2/token", m.tokenHandler)

	return mockOAuth2Handler{mux}
}

func (m *mockOAuth2) authHandler(w http.ResponseWriter, r *http.Request) {
	redirectURI := r.URL.Query().Get("redirect_uri")
	p, err := url.Parse(redirectURI)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	code := RandString(32)
	m.reqs.Store(code, struct{}{})

	values := p.Query()
	values.Set("code", code)
	values.Set("state", r.URL.Query().Get("state"))
	p.RawQuery = values.Encode()

	if _, err = http.Get(p.String()); err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
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
	switch g := r.FormValue("grant_type"); g {
	case "client_credentials":
	case "authorization_code":
		code := r.FormValue("code")
		if _, ok := m.reqs.Load(code); !ok {
			http.NotFound(w, r)
			return
		}
		m.reqs.Delete(code)
	default:
		http.Error(w, "invalid grant_type: "+g, http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(tokenJSON{
		AccessToken: RandString(32),
		TokenType:   "Bearer",
		ExpiresIn:   expirationTime(time.Hour),
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
