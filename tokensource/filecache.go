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

type fileToken struct {
	*oauth2.Token `json:"token"`
	IDToken       string `json:"id_token"`
}

// FileCache returns an oauth2.TokenSource that will cache tokens in the
// filesystem. On unix systems this will be in $XDG_DATA_HOME/<app> (or
// ~/.local/share/<app>). On windows systems this will be in
// %%LOCALAPPDATA%%/<app> (or C:\Users\<username>\AppData\Local\<app>).
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

		var token fileToken
		if err = json.NewDecoder(f).Decode(&token); err == nil && token.Valid() {
			if token.IDToken != "" {
				return token.WithExtra(map[string]interface{}{
					"id_token": token.IDToken,
				}), nil
			}
			return token.Token, nil
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

	ft := fileToken{Token: token}
	if extra := token.Extra("id_token"); extra != nil {
		if idToken, ok := extra.(string); ok {
			ft.IDToken = idToken
		}
	}

	if err = json.NewEncoder(f).Encode(ft); err != nil {
		return nil, err
	}

	return token, err
}
