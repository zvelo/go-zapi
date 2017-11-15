package zvelo

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDebug(t *testing.T) {
	var buf bytes.Buffer

	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		DebugRequest(&buf, r)
	}))

	req, err := http.NewRequest("GET", s.URL, nil)
	if err != nil {
		t.Fatal(err)
	}

	DebugRequestOut(&buf, req)

	if buf.Len() == 0 {
		t.Error("DebugRequestOut didn't write any data")
	}

	buf.Reset()

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}

	if buf.Len() == 0 {
		t.Error("DebugRequest didn't write any data")
	}

	buf.Reset()

	DebugResponse(&buf, resp, true)

	if buf.Len() == 0 {
		t.Error("DebugResponse didn't write any data")
	}
}

func TestRandString(t *testing.T) {
	val0 := RandString(32)
	if len(val0) != 32 {
		t.Errorf("len(val0)[%d] != 32", len(val0))
	}

	val1 := RandString(32)
	if len(val1) != 32 {
		t.Errorf("len(val1)[%d] != 32", len(val1))
	}

	if val0 == val1 {
		t.Error("random strings matched")
	}
}
