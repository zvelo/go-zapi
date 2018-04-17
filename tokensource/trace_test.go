package tokensource

import (
	"sync/atomic"
	"testing"

	"golang.org/x/oauth2"
)

func TestTrace(t *testing.T) {
	token := oauth2.Token{}

	var trace Trace

	ts := Tracer(trace, TestTokenSource{token: &token})

	got, err := ts.Token()
	if err != nil {
		t.Fatal(err)
	}

	if *got != token {
		t.Error("unexpected token")
	}

	var getToken, gotToken uint32

	trace = Trace{
		GetToken: func() {
			atomic.StoreUint32(&getToken, 1)
		},
		GotToken: func(got *oauth2.Token, err error) {
			t.Helper()

			if err != nil {
				t.Fatal(err)
			}

			if *got != token {
				t.Error("unexpected token")
			}

			atomic.StoreUint32(&gotToken, 1)
		},
	}

	ts = Tracer(trace, TestTokenSource{token: &token})

	got, err = ts.Token()
	if err != nil {
		t.Fatal(err)
	}

	if *got != token {
		t.Error("unexpected token")
	}

	if getToken != 1 || gotToken != 1 {
		t.Error("callbacks not called")
	}
}
