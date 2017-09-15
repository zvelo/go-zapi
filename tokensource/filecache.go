package tokensource

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"golang.org/x/oauth2"
)

type fileCache struct {
	src      oauth2.TokenSource
	fileName string
}

func FileCache(src oauth2.TokenSource, app, name string, scopes ...string) oauth2.TokenSource {
	hash := sha256.New()

	_, _ = hash.Write([]byte(name))

	// unique
	m := map[string]struct{}{}
	for _, scope := range scopes {
		m[scope] = struct{}{}
	}

	scopes = make([]string, 0, len(m))
	for scope := range m {
		scopes = append(scopes, scope)
	}

	sort.Strings(scopes)

	for _, scope := range scopes {
		_, _ = hash.Write([]byte(scope))
	}

	return fileCache{
		src:      src,
		fileName: filepath.Join(dataDir, app, fmt.Sprintf("token_%x.json", hash.Sum(nil))),
	}
}

func (s fileCache) Token() (*oauth2.Token, error) {
	// 1. check for token cached in filesystem

	// ignore errors since they we can always just go to the src
	if f, err := os.Open(s.fileName); err == nil {
		defer func() { _ = f.Close() }()

		var token oauth2.Token
		if err = json.NewDecoder(f).Decode(&token); err == nil && token.Valid() {
			return &token, nil
		}
	}

	// 2. fetch the token from the src

	token, err := s.src.Token()
	if err != nil {
		return nil, err
	}

	// 3. store the token in the filesystem

	if err = os.MkdirAll(filepath.Dir(s.fileName), 0700); err != nil {
		return nil, err
	}

	var f *os.File
	if f, err = os.OpenFile(s.fileName, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600); err != nil {
		return nil, err
	}

	defer func() { _ = f.Close() }()

	if err = json.NewEncoder(f).Encode(token); err != nil {
		return nil, err
	}

	return token, err
}
