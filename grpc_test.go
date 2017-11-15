package zapi

import (
	"context"
	"io/ioutil"
	"net"
	"net/http"
	"testing"

	"github.com/google/go-cmp/cmp"
	opentracing "github.com/opentracing/opentracing-go"

	"golang.org/x/oauth2"

	"google.golang.org/grpc/metadata"

	"zvelo.io/msg"
	"zvelo.io/msg/mock"
)

var (
	opts         []Option
	queryRequest *msg.QueryRequests
	queryURL     = "http://example.com"
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
		if err = mock.APIv1(mock.WhenReady(mockReady)).ListenAndServeTLS(context.Background(), mockAddr); err != nil {
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

	dialer := NewGRPCv1(TestTokenSource{}, opts...)
	client, err := dialer.Dial(ctx)
	if err != nil {
		t.Fatal(err)
	}

	ctx = mock.QueryContext(ctx, mock.WithCategories(msg.BLOG_4, msg.NEWS_4))

	stream, err := client.Stream(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}

	streamCh := make(chan *msg.QueryResult)
	go func() {
		result, serr := stream.Recv()
		if serr != nil {
			t.Error(serr)
		}
		streamCh <- result
	}()

	replies, err := client.Query(ctx, queryRequest)
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

	result, err := client.Result(ctx, &msg.RequestID{
		RequestId: reqID,
	})
	if err != nil {
		t.Fatal(err)
	}

	expect := queryExpect(reqID)
	resultEqual(t, result, expect)
	resultEqual(t, <-streamCh, expect)

	_, err = client.Suggest(ctx, &msg.Suggestion{
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

	if err = client.Close(); err != nil {
		t.Fatal(err)
	}
}

func resultEqual(t *testing.T, result, expect *msg.QueryResult) {
	t.Helper()

	if !cmp.Equal(result, expect) {
		t.Log(cmp.Diff(result, expect))
		t.Error("got unexpected result")
	}
}
