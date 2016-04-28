package zapi

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"zvelo.io/msg"
)

const jsonMIMESuffix = "+json"

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

func (c Client) queryHandler() (handler, error) {
	queryEndpoint, err := c.endpointURL(urlPath)
	if err != nil {
		return nil, err
	}

	r := req{
		ContentType: "application/vnd.zvelo.query",
		Accept:      "application/vnd.zvelo.query-reply",
		URL:         queryEndpoint.String(),
		Method:      "POST",
	}

	if c.JSON {
		r.ContentType += jsonMIMESuffix
		r.Accept += jsonMIMESuffix

		return jsonHandler{req: r}, nil
	}

	return pbHandler{req: r}, nil
}

func checkStatus(resp *http.Response, s *msg.Status, expectedCodes []int) error {
	if s == nil {
		fmt.Fprintf(os.Stderr, "missing status code in message\n")
		// return nil, errStatusCode(int(s.Code))
		return nil // TODO(jrubin)
	}

	foundExpectedCode := false
	for _, code := range expectedCodes {
		if int(s.Code) == code {
			foundExpectedCode = true
		}
	}

	if !foundExpectedCode {
		expected := make([]string, len(expectedCodes))
		for i, code := range expectedCodes {
			expected[i] = strconv.Itoa(code)
		}

		fmt.Fprintf(os.Stderr, "unexpected status code in message: %d (%s) => %s [expected: %s]\n", s.Code, http.StatusText(int(s.Code)), s.Message, strings.Join(expected, ", "))
		// return nil, errStatusCode(int(s.Code))
		return nil // TODO(jrubin)
	}

	if resp != nil {
		if int(s.Code) != resp.StatusCode {
			expected := make([]string, len(expectedCodes))
			for i, code := range expectedCodes {
				expected[i] = strconv.Itoa(code)
			}

			fmt.Fprintf(os.Stderr, "unexpected http status code: %d (%s) => %s [expected: %s]\n", resp.StatusCode, http.StatusText(resp.StatusCode), resp.Status, strings.Join(expected, ", "))
			// return nil, errStatusCode(resp.StatusCode)
			return nil // TODO(jrubin)
		}
	}

	return nil
}

func (c Client) Query(query *msg.QueryURLRequests) (*msg.QueryReply, error) {
	if query == nil {
		return nil, errNilRequest
	}

	var err error
	query.Url, err = massageURLs(query.Url)
	if err != nil {
		return nil, err
	}

	if len(query.Url) == 0 {
		return nil, errMissingURL
	}

	if len(query.Dataset) == 0 {
		return nil, errMissingDataSet
	}

	if len(c.Token) == 0 {
		if err = c.GetToken(); err != nil {
			return nil, err
		}
	}

	h, err := c.queryHandler()
	if err != nil {
		return nil, err
	}

	req, err := h.PrepareReq(query)
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

	reply := &msg.QueryReply{}
	if err = h.ParseResp(resp.Body, reply); err != nil {
		return nil, err
	}

	if err := checkStatus(resp, reply.Status, []int{http.StatusCreated}); err != nil {
		return nil, err
	}

	if len(reply.RequestId) == 0 {
		return nil, errMissingRequestID
	}

	return reply, nil
}
