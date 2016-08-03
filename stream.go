package zapi

import (
	"fmt"
	"net/http"
	"path"

	"zvelo.io/msg"
)

func (c Client) streamsHandler(method string, uuid string) (handler, error) {
	streamsEndpoint, err := c.endpointURL(path.Join(streamsPath, uuid))
	fmt.Println(streamsEndpoint)
	if err != nil {
		return nil, err
	}

	r := req{
		ContentType: "application/vnd.zvelo.stream-request",
		Accept:      "application/vnd.zvelo.stream-reply",
		URL:         streamsEndpoint.String(),
		Method:      method,
	}

	if c.JSON {
		r.ContentType += jsonMIMESuffix
		r.Accept += jsonMIMESuffix

		return jsonHandler{req: r}, nil
	}

	return pbHandler{req: r}, nil
}

func (c Client) StreamsList() (*msg.StreamsReply, error) {
	h, err := c.streamsHandler("GET", "")
	if err != nil {
		return nil, err
	}

	req, err := h.PrepareReq(nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/vnd.zvelo.streams-reply")
	if c.JSON {
		req.Header.Set("Accept", req.Header.Get("Accept")+jsonMIMESuffix)
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

	reply := &msg.StreamsReply{}
	if err = h.ParseResp(resp.Body, reply); err != nil {
		return nil, err
	}

	if err := checkStatus(resp, reply.Status, []int{http.StatusOK}); err != nil {
		return nil, err
	}

	return reply, nil
}

func (c Client) StreamCreate(stream *msg.StreamRequest) (*msg.StreamReply, error) {
	if stream == nil {
		return nil, errNilRequest
	}

	if len(stream.Endpoint) == 0 {
		return nil, errMissingEndpoint
	}

	h, err := c.streamsHandler("POST", "")
	if err != nil {
		return nil, err
	}

	req, err := h.PrepareReq(stream)
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

	reply := &msg.StreamReply{}
	if err = h.ParseResp(resp.Body, reply); err != nil {
		return nil, err
	}

	if err := checkStatus(resp, reply.Status, []int{http.StatusCreated}); err != nil {
		return nil, err
	}

	return reply, nil
}

func (c Client) StreamUpdate(uuid string, stream *msg.StreamRequest) (*msg.StreamReply, error) {
	if stream == nil {
		return nil, errNilRequest
	}

	if len(stream.Endpoint) == 0 {
		return nil, errMissingEndpoint
	}

	h, err := c.streamsHandler("PUT", uuid)
	if err != nil {
		return nil, err
	}

	req, err := h.PrepareReq(stream)
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

	// if ct := resp.Header.Get("Content-Type"); ct != req.Header.Get("Accept") {
	// 	return nil, errContentType(ct)
	// }
	//
	// reply := &msg.QueryReply{}
	// if err = h.ParseResp(resp.Body, reply); err != nil {
	// 	return nil, err
	// }
	//
	// if err := checkStatus(resp, reply.Status, []int{http.StatusCreated}); err != nil {
	// 	return nil, err
	// }
	//
	// if len(reply.RequestId) == 0 {
	// 	return nil, errMissingRequestID
	// }
	//
	// return reply, nil

	return nil, nil
}

func (c Client) StreamDelete(uuid string) (*msg.StreamReply, error) {
	h, err := c.streamsHandler("DELETE", uuid)
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

	// TODO(jrubin)

	return nil, nil
}
