package zapi

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"path"
	"time"

	"github.com/zvelo/go-zapi/zapitype"
)

func (c Client) PollOnce(reqID []byte) (*zapitype.QueryResult, error) {
	b64ReqID := base64.RawURLEncoding.EncodeToString(reqID[:])

	queryEndpoint, err := c.endpointURL(path.Join(urlPath, b64ReqID))
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("GET", queryEndpoint.String(), nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", c.UserAgent)
	req.Header.Set("Accept", "application/vnd.zvelo.query-result+json")

	c.debugRequest(req)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	c.debugResponse(resp)

	if resp.StatusCode != 200 {
		return nil, errStatusCode(resp.StatusCode)
	}

	decoder := json.NewDecoder(resp.Body)
	result := &zapitype.QueryResult{}
	if err = decoder.Decode(result); err != nil {
		return nil, errDecodeJSON(err.Error())
	}

	// TODO(jrubin) is this the right way to test for poll completion?
	if result.Status == nil {
		return nil, zapitype.ErrIncompleteResult(*result)
	}

	return result, nil
}

func (c Client) Poll(reqID []byte, errCh chan<- error) <-chan *zapitype.QueryResult {
	ch := make(chan *zapitype.QueryResult, 1)

	poll := func() bool {
		result, err := c.PollOnce(reqID)
		if err != nil {
			if errCh != nil {
				errCh <- err
			}
			return false
		}

		ch <- result
		return true
	}

	go func() {
		if poll() {
			return
		}

		tick := time.Tick(c.PollInterval)
		timeout := time.After(c.PollTimeout)

		for {
			select {
			case <-tick:
				if poll() {
					return
				}
			case <-timeout:
				close(ch)
				return
			}
		}
	}()

	return ch
}
