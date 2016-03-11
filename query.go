package zapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
)

const (
	contentTypeQueryReq  = "application/vnd.zvelo.query+json"
	contentTypeQueryResp = "application/vnd.zvelo.query-reply+json"
)

func massageURLs(urls []string) ([]string, error) {
	ret := make([]string, 0, len(urls))

	for _, u := range urls {
		if len(u) == 0 {
			continue
		}

		if strings.Index(u, "://") == -1 {
			u = "http://" + u
		}

		tmp, err := url.Parse(u)
		if err != nil {
			return nil, err
		}

		ret = append(ret, tmp.String())
	}

	return ret, nil
}

func (c Client) Query(query *QueryURLRequests) (*QueryReply, error) {
	if query == nil {
		return nil, ErrNilRequest
	}

	var err error
	query.URLs, err = massageURLs(query.URLs)
	if err != nil {
		return nil, err
	}

	if len(query.URLs) == 0 {
		return nil, ErrMissingURL
	}

	if len(query.DataSets) == 0 {
		return nil, ErrMissingDataSet
	}

	if len(c.Token) == 0 {
		if err = c.GetToken(); err != nil {
			return nil, err
		}
	}

	data, err := json.Marshal(query)
	if err != nil {
		return nil, err
	}

	queryEndpoint, err := c.endpointURL(urlPath)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", queryEndpoint.String(), bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", c.UserAgent)
	req.Header.Set("Content-Type", contentTypeQueryReq)
	req.Header.Set("Accept", contentTypeQueryResp)

	c.debugRequest(req)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	c.debugResponse(resp)

	if resp.StatusCode != 201 {
		return nil, ErrStatusCode(resp.StatusCode)
	}

	if ct := resp.Header.Get("Content-Type"); ct != contentTypeQueryResp {
		return nil, ErrContentType(ct)
	}

	decoder := json.NewDecoder(resp.Body)
	reply := &QueryReply{}
	if err = decoder.Decode(reply); err != nil {
		return nil, ErrDecodeJSON(err.Error())
	}

	if len(reply.RequestIDs) == 0 {
		return nil, ErrMissingRequestID
	}

	return reply, nil
}
