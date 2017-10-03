package clientauth

import (
	"context"
	"net/http/httptest"
	"testing"

	"zvelo.io/go-zapi/internal/zvelo"
)

func TestClientCredentials(t *testing.T) {
	ctx := context.Background()

	handler := zvelo.MockOAuth2Handler()
	srv := httptest.NewServer(handler)

	ts := ClientCredentials(ctx, "", "",
		WithTokenURL(""),
		WithTokenURL(handler.Endpoint(srv.URL).TokenURL),
		WithScope("some_scope"),
		WithScope(),
	)

	token, err := ts.Token()
	if err != nil {
		t.Fatal(err)
	}

	if !token.Valid() {
		t.Error("got invalid token")
	}
}
