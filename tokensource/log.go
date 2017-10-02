package tokensource

import (
	"fmt"
	"io"
	"time"

	"golang.org/x/oauth2"
)

type logTokenSource struct {
	io.Writer
	src oauth2.TokenSource
}

func (s logTokenSource) Token() (*oauth2.Token, error) {
	start := time.Now()

	token, err := s.src.Token()

	if err == nil {
		fmt.Fprintf(s, "got token (%s)\n", time.Since(start))
	} else {
		fmt.Fprintf(s, "error getting token: %s (%s)\n", err, time.Since(start))
	}

	return token, err
}

var _ oauth2.TokenSource = (*logTokenSource)(nil)

// Log returns an oauth2.TokenSource that will log debug information to the
// writer
func Log(w io.Writer, src oauth2.TokenSource) oauth2.TokenSource {
	return logTokenSource{
		Writer: w,
		src:    src,
	}
}
