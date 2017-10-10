package zapi

import (
	"bytes"
	"context"
	"net/http"
	"testing"
	"text/template"

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

	var s string
	if err = client.GraphQL(ctx, graphQLQuery0, &s); err != nil {
		t.Fatal(err)
	}

	var m graphQLReply
	if err = client.GraphQL(ctx, graphQLQuery0, &m); err != nil {
		t.Fatal(err)
	}

	// trigger gzip encoding

	var rids []string
	for i := 0; i < 1000; i++ {
		rids = append(rids, m.Data.URL.RequestID)
	}

	var buf bytes.Buffer
	if err = graphQLQuery1.Execute(&buf, rids); err != nil {
		t.Fatal(err)
	}

	if err = client.GraphQL(ctx, buf.String(), &s); err != nil {
		t.Fatal(err)
	}

}

type graphQLReply struct {
	Data struct {
		URL struct {
			RequestID string
		}
	}
}

var graphQLQuery0 string

func init() {
	graphQLQuery0 = `query {
	url(url: "` + queryURL + `", dataset: [CATEGORIZATION]) {
		...replyFields
	}
}

fragment replyFields on QueryReply {
	requestID
	error {
		code
		message
	}
}`
}

var graphQLQuery1Str = `query {
	{{- range $i, $id := .}}
	r{{$i}}: result(requestID: "{{$id}}") {
		...resultFields
	}
	{{- end}}
}

fragment resultFields on QueryResult {
	responseDataSet {
		categorization
		malicious {
			category
			verdict
		}
		echo
	}
	queryStatus {
		complete
		error {
			code
			message
		}
		fetchCode
		location
	}
}`

var graphQLQuery1 = template.Must(template.New("result").Parse(graphQLQuery1Str))
