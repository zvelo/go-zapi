package zapi

import (
	"encoding/json"
	"net/http"
	"path"
	"time"

	"zvelo.io/msg/go-msg"
)

func (c Client) PollOnce(reqID string) (*msg.QueryResult, error) {
	if len(c.Token) == 0 {
		if err := c.GetToken(); err != nil {
			return nil, err
		}
	}

	queryEndpoint, err := c.endpointURL(path.Join(urlPath, reqID))
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("GET", queryEndpoint.String(), nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", c.UserAgent)
	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("Accept", "application/vnd.zvelo.query-result+json")

	c.debugRequestOut(req)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	c.debugResponse(resp)

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		return nil, errStatusCode(resp.StatusCode)
	}

	decoder := json.NewDecoder(resp.Body)
	result := &msg.QueryResult{}
	if err = decoder.Decode(result); err != nil {
		return nil, errDecodeJSON(err.Error())
	}

	if result.Status == nil || int(result.Status.Code) != resp.StatusCode {
		return nil, errStatusCode(int(result.Status.Code))
	}

	if resp.StatusCode != http.StatusOK {
		// implies status code is 202 http.StatusAccepted
		return nil, ErrIncompleteResult(*result)
	}

	return result, nil
}

func (c Client) Poll(reqID string, errCh chan<- error) <-chan *msg.QueryResult {
	ch := make(chan *msg.QueryResult, 1)

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
