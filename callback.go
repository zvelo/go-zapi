package zapi

import (
	"net/http"

	"zvelo.io/msg"
)

func (c Client) ProcessCallback(req *http.Request) (*msg.QueryResult, error) {
	defer func() { _ = req.Body.Close() }()

	c.debugRequest(req)

	var h handler

	ct := req.Header.Get("Content-Type")
	if ct == queryResultType {
		h = pbHandler{}
	} else if ct == queryResultType+jsonMIMESuffix {
		h = jsonHandler{}
	} else {
		return nil, errContentType(ct)
	}

	result := &msg.QueryResult{}
	if err := h.ParseResp(req.Body, result); err != nil {
		return nil, err
	}

	// TODO(jrubin) both OK and Accepted here or just OK?
	if err := checkStatus(nil, result.Status, []int{http.StatusOK, http.StatusAccepted}); err != nil {
		return nil, err
	}

	return result, nil
}
