package zapi

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path"
	"time"

	"zvelo.io/msg/go-msg"
)

func (c Client) PollOnce(reqID []byte, dsts []msg.DataSetType) (*msg.QueryResult, error) {
	if len(c.Token) == 0 {
		if err := c.GetToken(); err != nil {
			return nil, err
		}
	}

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
	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("Accept", "application/vnd.zvelo.query-result+json")

	c.debugRequestOut(req)

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
	result := &msg.QueryResult{}
	if err = decoder.Decode(result); err != nil {
		return nil, errDecodeJSON(err.Error())
	}

	complete := true
	for _, dst := range dsts {
		i, err := DataSetByType(result.ResponseDataset, dst)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error checking result: %s\n", err)
			continue
		}

		if i == nil {
			fmt.Fprintf(os.Stderr, "still missing %s dataset\n", dst)
			complete = false
		} else {
			fmt.Fprintf(os.Stderr, "got %s dataset\n", dst)
		}
	}

	// TODO(jrubin) is this the right way to test for poll completion?
	if !complete {
		return nil, ErrIncompleteResult(*result)
	}

	return result, nil
}

func (c Client) Poll(reqID []byte, dsts []msg.DataSetType, errCh chan<- error) <-chan *msg.QueryResult {
	ch := make(chan *msg.QueryResult, 1)

	poll := func() bool {
		result, err := c.PollOnce(reqID, dsts)
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
