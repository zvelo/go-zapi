package zapi

import (
	"bytes"
	"context"
	"net/http"
	"testing"
	"text/template"

	opentracing "github.com/opentracing/opentracing-go"

	"zvelo.io/msg"
	"zvelo.io/msg/mock"
)

func TestREST(t *testing.T) {
	span, ctx := opentracing.StartSpanFromContext(context.Background(), "test")
	defer span.Finish()

	client := NewRESTv1(TestTokenSource{}, opts...)

	ctx = mock.QueryContext(ctx, mock.WithCategories(msg.BLOG_4, msg.NEWS_4))

	streamCh := make(chan *msg.QueryResult)
	go func() {
		stream, err := client.Stream(ctx)
		if err != nil {
			t.Error(err)
		}

		result, serr := stream.Recv()
		if serr != nil {
			t.Error(serr)
		}
		streamCh <- result
	}()

	var resp *http.Response
	replies, err := client.Query(ctx, queryRequest, Response(&resp))
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

	result, err := client.Result(ctx, reqID)
	if err != nil {
		t.Fatal(err)
	}

	expect := queryExpect(reqID)
	resultEqual(t, result, expect)
	resultEqual(t, <-streamCh, expect)

	err = client.Suggest(ctx, &msg.Suggestion{
		Url: "http://example.com",
		Dataset: &msg.DataSet{
			Echo: &msg.DataSet_Echo{
				Url: "http://example.com",
			},
		},
	})
	if err != nil {
		t.Error(err)
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
