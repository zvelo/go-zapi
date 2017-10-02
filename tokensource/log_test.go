package tokensource

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"golang.org/x/oauth2"
)

func testLog(t *testing.T, buf *bytes.Buffer, ts oauth2.TokenSource, expectToken oauth2.Token, expectError, expectPrefix string) {
	t.Helper()

	token, err := ts.Token()
	if err != nil {
		if expectError == "" {
			t.Error(err)
		} else if !strings.HasPrefix(err.Error(), expectError) {
			t.Error(err)
		}
	}
	if *token != expectToken {
		t.Error("unexpected token")
	}

	if !strings.HasPrefix(buf.String(), expectPrefix) {
		t.Errorf("unexpected output: %s", buf.String())
	}
}

func TestLog(t *testing.T) {
	token := oauth2.Token{}

	var buf bytes.Buffer
	ts := Log(&buf, TestTokenSource{token: &token})

	testLog(t, &buf, ts, token, "", "got token (")

	buf.Reset()
	ts = Log(&buf, TestTokenSource{
		token: &token,
		err:   errors.New("token error"),
	})

	testLog(t, &buf, ts, token, "token error", "error getting token: token error (")
}
