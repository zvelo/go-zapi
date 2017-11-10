package callback

import (
	"encoding/json"
	"net/http"

	"github.com/pkg/errors"

	"gopkg.in/square/go-jose.v2"

	"zvelo.io/httpsig"
)

// KeyCache is a simple interface for caching JSON Web Keys
type KeyCache interface {
	Get(string) *jose.JSONWebKeySet
	Set(string, *jose.JSONWebKeySet)
}

type keyGetter struct {
	cache KeyCache
}

// KeyGetter returns an httpsig.KeyGetter that will properly fetch zvelo public
// keys, if cache is non nil, it will be used to cache keys.
func KeyGetter(cache KeyCache) httpsig.KeyGetter {
	return &keyGetter{cache: cache}
}

func extractKey(keyset *jose.JSONWebKeySet) (interface{}, error) {
	keys := keyset.Key("public")

	if len(keys) == 0 {
		return nil, errors.New("no public key")
	}

	return keys[0].Key, nil
}

func (g *keyGetter) GetKey(keyID string) (interface{}, error) {
	// 1. check for key cached in filesystem

	if g.cache != nil {
		if keyset := g.cache.Get(keyID); keyset != nil {
			return extractKey(keyset)
		}
	}

	// 2. fetch the key

	resp, err := http.Get(keyID)
	if err != nil {
		return nil, err
	}

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.Errorf("unexpected status fetching key: %s", resp.Status)
	}

	var keyset jose.JSONWebKeySet
	if err := json.NewDecoder(resp.Body).Decode(&keyset); err != nil {
		return nil, err
	}

	// 3. write the json key to the cache file as we decode it

	if g.cache != nil {
		g.cache.Set(keyID, &keyset)
	}

	return extractKey(&keyset)
}
