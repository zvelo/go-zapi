package callback

import (
	"sync"

	"gopkg.in/square/go-jose.v2"
)

// MemKeyCache returns a KeyCache that stores keys in memory
func MemKeyCache() KeyCache {
	return &memKeyCache{
		cache: map[string]*jose.JSONWebKeySet{},
	}
}

type memKeyCache struct {
	sync.RWMutex
	cache map[string]*jose.JSONWebKeySet
}

func (c *memKeyCache) Get(keyID string) *jose.JSONWebKeySet {
	c.RLock()
	defer c.RUnlock()

	return c.cache[keyID]
}

func (c *memKeyCache) Set(keyID string, keyset *jose.JSONWebKeySet) {
	c.Lock()
	defer c.Unlock()

	c.cache[keyID] = keyset
}
