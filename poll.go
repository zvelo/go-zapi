package zapi

import (
	"net/http"
	"path"
	"time"

	"zvelo.io/msg"
)

const queryResultType = "application/vnd.zvelo.query-result"

func (c Client) resultHandler(reqID string) (handler, error) {
	if len(c.Token) == 0 {
		if err := c.GetToken(); err != nil {
			return nil, err
		}
	}

	queryEndpoint, err := c.endpointURL(path.Join(urlPath, reqID))
	if err != nil {
		return nil, err
	}

	r := req{
		Accept: queryResultType,
		URL:    queryEndpoint.String(),
		Method: "GET",
	}

	if c.JSON {
		r.Accept += jsonMIMESuffix
		return jsonHandler{req: r}, nil
	}

	return pbHandler{req: r}, nil
}

func (c Client) PollOnce(reqID string) (*msg.QueryResult, error) {
	if len(c.Token) == 0 {
		if err := c.GetToken(); err != nil {
			return nil, err
		}
	}

	h, err := c.resultHandler(reqID)
	if err != nil {
		return nil, err
	}

	req, err := h.PrepareReq(nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", c.UserAgent)
	req.Header.Set("Authorization", "Bearer "+c.Token)

	c.debugRequestOut(req)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	c.debugResponse(resp)

	if ct := resp.Header.Get("Content-Type"); ct != req.Header.Get("Accept") {
		return nil, errContentType(ct)
	}

	result := &msg.QueryResult{}
	if err = h.ParseResp(resp.Body, result); err != nil {
		return nil, err
	}

	if err := checkStatus(resp, result.Status, []int{http.StatusOK, http.StatusAccepted}); err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		// implies status code is 202 http.StatusAccepted
		return nil, ErrIncompleteResult(*result)
	}

	return result, nil
}

func (c Client) Poll(reqID string, ch chan<- *msg.QueryResult, errCh chan<- error) {
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
}
