package tokensource

import (
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/oauth2"
	"zvelo.io/go-zapi/internal/zvelo"
)

func testToken(t *testing.T, accessToken, idToken string, token *oauth2.Token) {
	t.Helper()

	if token.AccessToken != accessToken {
		t.Error("access tokens don't match")
	}

	at, ok := token.Extra("id_token").(string)
	if !ok {
		t.Fatal("id_token not a string")
	}

	if at != idToken {
		t.Error("id_tokens don't match")
	}
}

type TestTokenSource struct {
	token *oauth2.Token
	err   error
}

func (ts TestTokenSource) Token() (*oauth2.Token, error) {
	return ts.token, ts.err
}

func TestFileCache(t *testing.T) {
	const app = "testapp"

	var err error

	if err = os.RemoveAll(filepath.Join(zvelo.DataDir, app)); err != nil {
		t.Fatal(err)
	}

	accessToken := zvelo.RandString(32)
	idToken := zvelo.RandString(32)

	token := &oauth2.Token{AccessToken: accessToken}
	token = token.WithExtra(map[string]interface{}{
		"id_token": idToken,
	})

	ts := FileCache(TestTokenSource{token: token}, app, "test", "scope0", "scope1")

	if token, err = ts.Token(); err != nil {
		t.Fatal(err)
	}

	testToken(t, accessToken, idToken, token)

	ts = FileCache(TestTokenSource{token: &oauth2.Token{}}, app, "test", "scope0", "scope1")

	if token, err = ts.Token(); err != nil {
		t.Fatal(err)
	}

	testToken(t, accessToken, idToken, token)
}
