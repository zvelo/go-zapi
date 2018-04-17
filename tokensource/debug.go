package tokensource

import (
	"io"
	"time"

	"golang.org/x/oauth2"
	"zvelo.io/go-zapi/internal/zvelo"
)

type debugTokenSource struct {
	io.Writer
	src oauth2.TokenSource
}

func (s debugTokenSource) Token() (*oauth2.Token, error) {
	start := time.Now()

	token, err := s.src.Token()

	zvelo.DebugTiming(s, "Token", time.Since(start))

	return token, err
}

// Debug returns an oauth2.TokenSource that will log timing info to w
func Debug(w io.Writer, src oauth2.TokenSource) oauth2.TokenSource {
	return debugTokenSource{
		Writer: w,
		src:    src,
	}
}
