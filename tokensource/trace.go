package tokensource

import (
	"golang.org/x/oauth2"
)

type traceTokenSource struct {
	Trace
	src oauth2.TokenSource
}

func (s traceTokenSource) Token() (*oauth2.Token, error) {
	if s.GetToken != nil {
		s.GetToken()
	}

	token, err := s.src.Token()

	if s.GotToken != nil {
		s.GotToken(token, err)
	}

	return token, err
}

// Trace is a set of hooks to run befor and after a token is retrieved
type Trace struct {
	GetToken func()
	GotToken func(token *oauth2.Token, err error)
}

// Tracer returns an oauth2.TokenSource that will call non-nil hooks on trace
func Tracer(trace Trace, src oauth2.TokenSource) oauth2.TokenSource {
	return traceTokenSource{
		Trace: trace,
		src:   src,
	}
}
