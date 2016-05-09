package zapi

import "zvelo.io/msg"

func (c Client) streamsHandler(method string) (handler, error) {
	streamsEndpoint, err := c.endpointURL(streamsPath)
	if err != nil {
		return nil, err
	}

	r := req{
		ContentType: "application/vnd.zvelo.stream-request",
		Accept:      "application/vnd.zvelo.stream",
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

func (c Client) StreamsList() ([]*msg.StreamReply, error) {
	h, err := c.streamsHandler("GET")
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

func (c Client) StreamCreate(stream *msg.StreamRequest) (*msg.StreamReply, error) {
	return nil, nil
}
