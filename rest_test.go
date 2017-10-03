package zapi

import (
	"context"
	"net/http"
	"testing"

	"github.com/google/go-cmp/cmp"
	opentracing "github.com/opentracing/opentracing-go"
)

func TestRest(t *testing.T) {
	span, ctx := opentracing.StartSpanFromContext(context.Background(), "test")
	defer span.Finish()

	client := NewREST(TestTokenSource{}, opts...)

	var resp *http.Response
	replies, err := client.QueryV1(ctx, queryRequest, Response(&resp))
	if err != nil {
		t.Fatal(err)
	}

	if replies == nil || len(replies.Reply) != 1 {
		t.Fatal("unexpected replies")
	}

	reqID := replies.Reply[0].RequestId

	if reqID == "" {
		t.Error("empty request_id")
	}

	result, err := client.QueryResultV1(ctx, reqID)
	if err != nil {
		t.Fatal(err)
	}

	if !cmp.Equal(result, queryExpect(reqID)) {
		t.Log(cmp.Diff(result, queryExpect(reqID)))
		t.Error("got unexpected result")
	}
}
