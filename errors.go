package zapi

import (
	"fmt"
	"net/http"
)

var (
	ErrMissingEndpoint    = fmt.Errorf("endpoint is required")
	ErrInvalidEndpoint    = fmt.Errorf("invalid endpoint")
	ErrMissingAccessToken = fmt.Errorf("response did not contain an access token")
	ErrInvalidAccessToken = fmt.Errorf("access token was not a string")
	ErrMissingUsername    = fmt.Errorf("username is required")
	ErrMissingPassword    = fmt.Errorf("password is required")
	ErrMissingRequestID   = fmt.Errorf("response did not contain a request id")
	ErrMissingURL         = fmt.Errorf("at least one url is required")
	ErrMissingDataSet     = fmt.Errorf("at least one dataset is required")
	ErrNilRequest         = fmt.Errorf("request was nil")
)

type ErrStatusCode int

func (e ErrStatusCode) Error() string {
	return fmt.Sprintf("unexpected status code: %d (%s)", int(e), http.StatusText(int(e)))
}

type ErrDecodeJSON string

func (e ErrDecodeJSON) Error() string {
	return fmt.Sprintf("could not decode json response: %s", string(e))
}

type ErrContentType string

func (e ErrContentType) Error() string {
	return fmt.Sprintf("unexpected content type: %s", string(e))
}

type ErrIncompleteResult QueryResult

func (e ErrIncompleteResult) Error() string {
	return "incomplete result"
}
