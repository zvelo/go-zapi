package callback

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/square/go-jose.v2"

	"zvelo.io/go-zapi/internal/zvelo"
)

// FileKeyCache returns a KeyCache that stores keys on disk
func FileKeyCache(cacheName string) KeyCache {
	return &fileKeyCache{
		cacheName: cacheName,
	}
}

type fileKeyCache struct {
	cacheName string
}

func (c fileKeyCache) cacheFile(keyID string) string {
	return filepath.Join(zvelo.DataDir(c.cacheName), fmt.Sprintf("key_%x.json", sha256.Sum256([]byte(keyID))))
}

func (c fileKeyCache) Get(keyID string) *jose.JSONWebKeySet {
	// ignore errors since we can always just fetch the key
	if f, err := os.Open(c.cacheFile(keyID)); err != nil {
		defer func() { _ = f.Close() }()
		var keyset jose.JSONWebKeySet
		if err := json.NewDecoder(f).Decode(&keyset); err == nil {
			return &keyset
		}
	}

	return nil
}

func (c fileKeyCache) Set(keyID string, keyset *jose.JSONWebKeySet) {
	// errors are ignored
	var err error

	cacheFile := c.cacheFile(keyID)

	if err = os.MkdirAll(filepath.Dir(cacheFile), 0700); err != nil {
		return
	}

	var f *os.File
	if f, err = os.OpenFile(cacheFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600); err != nil {
		return
	}

	defer func() { _ = f.Close() }()

	_ = json.NewEncoder(f).Encode(keyset)
}
