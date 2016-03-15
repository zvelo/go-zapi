package zapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"

	"zvelo.io/go-zapi/zapitype"
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

func (c Client) Query(query *zapitype.QueryURLRequests) (*zapitype.QueryReply, error) {
	if query == nil {
		return nil, errNilRequest
	}

	var err error
	query.URLs, err = massageURLs(query.URLs)
	if err != nil {
		return nil, err
	}

	if len(query.URLs) == 0 {
		return nil, errMissingURL
	}

	if len(query.DataSets) == 0 {
		return nil, errMissingDataSet
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
		return nil, errStatusCode(resp.StatusCode)
	}

	if ct := resp.Header.Get("Content-Type"); ct != contentTypeQueryResp {
		return nil, errContentType(ct)
	}

	decoder := json.NewDecoder(resp.Body)
	reply := &zapitype.QueryReply{}
	if err = decoder.Decode(reply); err != nil {
		return nil, errDecodeJSON(err.Error())
	}

	if len(reply.RequestIDs) == 0 {
		return nil, errMissingRequestID
	}

	return reply, nil
}
