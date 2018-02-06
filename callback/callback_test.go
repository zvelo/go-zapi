package callback

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	jose "gopkg.in/square/go-jose.v2"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"

	"zvelo.io/go-zapi/internal/zvelo"
	"zvelo.io/httpsig"
	"zvelo.io/msg"
)

func handler(m **msg.QueryResult) Handler {
	return HandlerFunc(func(_ http.ResponseWriter, _ *http.Request, in *msg.QueryResult) {
		*m = in
	})
}

const hydraURL = "https://auth.zvelo.com"

var (
	keyset       = os.Getenv("KEYSET")
	clientID     = os.Getenv("APP_CLIENT_ID")
	clientSecret = os.Getenv("APP_CLIENT_SECRET")
)

func getPrivateKey(t *testing.T) (string, *ecdsa.PrivateKey) {
	t.Helper()

	config := clientcredentials.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		TokenURL:     "https://auth.zvelo.com/oauth2/token",
		Scopes:       []string{"hydra.keys.get"},
	}

	client := oauth2.NewClient(context.Background(), config.TokenSource(context.Background()))

	resp, err := client.Get(hydraURL + "/" + path.Join("keys", keyset))
	if err != nil {
		t.Fatal(err)
	}

	defer func() { _ = resp.Body.Close() }()

	var keys jose.JSONWebKeySet

	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		t.Fatalf("Expected status code %d, got %d.\n%s\n", http.StatusOK, resp.StatusCode, body)
	}

	if err = json.NewDecoder(resp.Body).Decode(&keys); err != nil {
		t.Fatal(err)
	}

	key, ok := keys.Key("private")[0].Key.(*ecdsa.PrivateKey)
	if !ok {
		t.Fatal("invalid private key")
	}

	keyID := hydraURL + "/" + path.Join("keys", keyset, "public")

	return keyID, key
}

func TestCallbackHandler(t *testing.T) {
	const app = "testapp"

	if err := os.RemoveAll(filepath.Join(zvelo.DataDir, app)); err != nil {
		t.Fatal(err)
	}

	var m *msg.QueryResult
	srv := httptest.NewServer(Middleware(KeyGetter(MemKeyCache()), handler(&m), nil))

	r := msg.QueryResult{
		ResponseDataset: &msg.DataSet{
			Categorization: &msg.DataSet_Categorization{
				Value: []msg.Category{
					msg.BLOG_4,
					msg.NEWS_4,
				},
			},
		},
		QueryStatus: &msg.QueryStatus{
			Complete:  true,
			FetchCode: http.StatusOK,
		},
	}

	body, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}

	httpClient := &http.Client{
		Transport: httpsig.ECDSASHA256.Transport(getPrivateKey(t)),
		Timeout:   30 * time.Second,
	}

	if _, err = httpClient.Post(srv.URL, "application/json", bytes.NewReader(body)); err != nil {
		t.Fatal(err)
	}

	if !cmp.Equal(&r, m) {
		t.Log(cmp.Diff(&r, m))
		t.Error("got unexpected result")
	}

	if _, err = httpClient.Post(srv.URL, "application/json", bytes.NewReader(body)); err != nil {
		t.Fatal(err)
	}
}
