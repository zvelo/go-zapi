package zapi

import (
	"context"
	"io/ioutil"
	"net"
	"net/http"
	"testing"

	"golang.org/x/oauth2"

	"google.golang.org/grpc/metadata"

	"github.com/google/go-cmp/cmp"
	opentracing "github.com/opentracing/opentracing-go"

	"zvelo.io/msg"
	"zvelo.io/msg/mock"
)

var (
	opts         []Option
	queryURL     string
	queryRequest *msg.QueryRequests
)

type TestTokenSource struct {
	token *oauth2.Token
	err   error
}

func (ts TestTokenSource) Token() (*oauth2.Token, error) {
	token := ts.token
	if token == nil {
		token = &oauth2.Token{}
	}
	return token, ts.err
}

func init() {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		panic(err)
	}

	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		panic(err)
	}

	mockAddr := l.Addr().String()
	if err = l.Close(); err != nil {
		panic(err)
	}

	mockReady := make(chan struct{})

	go func() {
		if err = mock.ListenAndServeTLS(context.Background(), mockAddr, mock.WithOnReady(mockReady)); err != nil {
			panic(err)
		}
	}()

	<-mockReady

	opts = []Option{
		WithForceTrace(),
		WithTLSInsecureSkipVerify(),
		WithTransport(http.DefaultTransport),
		WithTransport(nil),
		WithTracer(nil),
		WithTracer(opentracing.GlobalTracer()),
		WithDebug(ioutil.Discard),
		WithDebug(nil),
		WithAddr(""),
		WithAddr(mockAddr),
	}

	queryURL, err = mock.NewQueryURL("http://example.com",
		mock.WithCategories(
			msg.BLOG_4,
			msg.NEWS_4,
		),
	)
	if err != nil {
		panic(err)
	}

	queryRequest = &msg.QueryRequests{
		Url: []string{queryURL},
		Dataset: []msg.DataSetType{
			msg.CATEGORIZATION,
			msg.ECHO,
		},
	}
}

func queryExpect(reqID string) *msg.QueryResult {
	return &msg.QueryResult{
		RequestId: reqID,
		ResponseDataset: &msg.DataSet{
			Categorization: &msg.DataSet_Categorization{
				Value: []msg.Category{
					msg.BLOG_4,
					msg.NEWS_4,
				},
			},
			Echo: &msg.DataSet_Echo{
				Url: queryURL,
			},
		},
		QueryStatus: &msg.QueryStatus{
			Complete:  true,
			FetchCode: http.StatusOK,
		},
	}
}

func TestGRPC(t *testing.T) {
	ctx := metadata.NewOutgoingContext(context.Background(), nil)

	dialer := NewGRPC(TestTokenSource{}, opts...)
	client, err := dialer.Dial(ctx)
	if err != nil {
		t.Fatal(err)
	}

	replies, err := client.QueryV1(ctx, queryRequest)
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

	result, err := client.QueryResultV1(ctx, &msg.QueryPollRequest{
		RequestId: reqID,
	})
	if err != nil {
		t.Fatal(err)
	}

	if !cmp.Equal(result, queryExpect(reqID)) {
		t.Log(cmp.Diff(result, queryExpect(reqID)))
		t.Error("got unexpected result")
	}

	if err = client.Close(); err != nil {
		t.Fatal(err)
	}
}
