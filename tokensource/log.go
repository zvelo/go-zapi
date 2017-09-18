package tokensource

import (
	"fmt"
	"os"
	"time"

	"golang.org/x/oauth2"
)

type logTokenSource struct {
	src oauth2.TokenSource
}

func (s logTokenSource) Token() (*oauth2.Token, error) {
	start := time.Now()

	token, err := s.src.Token()

	if err == nil {
		fmt.Fprintf(os.Stderr, "got token (%s)\n", time.Since(start))
	} else {
		fmt.Fprintf(os.Stderr, "error getting token: %s (%s)\n", err, time.Since(start))
	}

	return token, err
}

var _ oauth2.TokenSource = (*logTokenSource)(nil)

// Log returns an oauth2.TokenSource that will log debug information to stderr
func Log(src oauth2.TokenSource) oauth2.TokenSource {
	return logTokenSource{src: src}
}
