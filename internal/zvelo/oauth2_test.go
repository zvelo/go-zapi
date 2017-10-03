package zvelo

import (
	"context"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
)

func authCodeHandler(ctx context.Context, state string, c *oauth2.Config, token **oauth2.Token) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if state == "" || state != r.URL.Query().Get("state") {
			http.Error(w, "invalid state", http.StatusUnauthorized)
			return
		}

		var err error

		if _, err = c.Exchange(ctx, "bad_code"); err == nil {
			http.Error(w, "bad_code didn't return error", http.StatusInternalServerError)
		}

		if *token, err = c.Exchange(ctx, r.URL.Query().Get("code")); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	})
}

func TestMockOAuth2(t *testing.T) {
	handler := MockOAuth2Handler()
	srv := httptest.NewServer(handler)
	ctx := context.Background()

	t.Run("client_credentials", func(t *testing.T) {
		t.Parallel()

		c := clientcredentials.Config{
			TokenURL: handler.Endpoint(srv.URL).TokenURL,
		}

		token, err := c.Token(ctx)
		if err != nil {
			t.Fatal(err)
		}

		if !token.Valid() {
			t.Error("invalid token")
		}
	})

	t.Run("authorization_code", func(t *testing.T) {
		t.Parallel()

		state := RandString(32)

		c := oauth2.Config{
			Endpoint: handler.Endpoint(srv.URL),
		}

		var token *oauth2.Token

		// 1. try without RedirectURL

		resp, err := http.Get(c.AuthCodeURL(state))
		if err != nil {
			t.Fatal(err)
		}

		if resp.StatusCode != http.StatusBadGateway {
			t.Error("didn't get expected error")
		}

		// 2. try with bad RedirectURL

		c.RedirectURL = "://bad_url"

		if resp, err = http.Get(c.AuthCodeURL(state)); err != nil {
			t.Fatal(err)
		}

		if resp.StatusCode != http.StatusBadRequest {
			t.Error("didn't get expected error")
		}

		// 3. try with good RedirectURL

		tsrv := httptest.NewServer(authCodeHandler(ctx, state, &c, &token))
		c.RedirectURL = tsrv.URL

		if resp, err = http.Get(c.AuthCodeURL(state)); err != nil {
			log.Fatal(err)
		}

		if resp.StatusCode != http.StatusOK {
			t.Error("server didn't return OK")
		}

		if !token.Valid() {
			t.Error("invalid token")
		}
	})
}
