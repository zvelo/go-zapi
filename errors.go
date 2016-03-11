package zapi

import (
	"fmt"
	"net/http"

	"github.com/zvelo/go-zapi/zapitype"
)

var (
	errMissingAccessToken = fmt.Errorf("response did not contain an access token")
	errMissingUsername    = fmt.Errorf("username is required")
	errMissingPassword    = fmt.Errorf("password is required")
	errMissingRequestID   = fmt.Errorf("response did not contain a request id")
	errMissingURL         = fmt.Errorf("at least one url is required")
	errMissingDataSet     = fmt.Errorf("at least one dataset is required")
	errNilRequest         = fmt.Errorf("request was nil")
)

type errStatusCode int

func (e errStatusCode) Error() string {
	return fmt.Sprintf("unexpected status code: %d (%s)", int(e), http.StatusText(int(e)))
}

type errDecodeJSON string

func (e errDecodeJSON) Error() string {
	return fmt.Sprintf("could not decode json response: %s", string(e))
}

type errContentType string

func (e errContentType) Error() string {
	return fmt.Sprintf("unexpected content type: %s", string(e))
}

type ErrIncompleteResult zapitype.QueryResult

func (e ErrIncompleteResult) Error() string {
	return "incomplete result"
}
