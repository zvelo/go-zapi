package callback

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	hydra "github.com/ory/hydra/sdk"

	"zvelo.io/go-zapi/internal/zvelo"
	"zvelo.io/httpsig"
	"zvelo.io/msg"
)

func handler(m **msg.QueryResult) Handler {
	return HandlerFunc(func(in *msg.QueryResult) {
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

	hc, err := hydra.Connect(
		hydra.ClientID(clientID),
		hydra.ClientSecret(clientSecret),
		hydra.ClusterURL(hydraURL),
		hydra.Scopes("hydra.keys.get"),
	)

	if err != nil {
		t.Fatal(err)
	}

	keys, err := hc.JSONWebKeys.GetKeySet(keyset)
	if err != nil {
		t.Fatal(err)
	}

	key, ok := keys.Key("private")[0].Key.(*ecdsa.PrivateKey)
	if !ok {
		t.Fatal("invalid private key")
	}

	keyID := fmt.Sprintf("%s/%s/public", hc.JSONWebKeys.Endpoint, keyset)

	return keyID, key
}

func TestCallbackHandler(t *testing.T) {
	const app = "testapp"

	if err := os.RemoveAll(filepath.Join(zvelo.DataDir, app)); err != nil {
		t.Fatal(err)
	}

	var m *msg.QueryResult
	srv := httptest.NewServer(HTTPHandler(app, handler(&m)))

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
