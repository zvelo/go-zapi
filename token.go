package zapi

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
)

func (c *Client) GetToken() error {
	if len(c.Username) == 0 {
		return ErrMissingUsername
	}

	if len(c.Password) == 0 {
		return ErrMissingPassword
	}

	c.Token = ""
	tokenEndpoint, err := c.endpointURL(tokenPath)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", tokenEndpoint.String(), strings.NewReader(url.Values{
		"username":   {c.Username},
		"password":   {c.Password},
		"grant_type": {"password"},
	}.Encode()))

	req.Header.Set("User-Agent", c.UserAgent)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	c.debugRequest(req)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}

	defer func() { _ = resp.Body.Close() }()
	c.debugResponse(resp)

	if resp.StatusCode != 200 {
		return ErrStatusCode(resp.StatusCode)
	}

	decoder := json.NewDecoder(resp.Body)
	t := &token{}
	if err = decoder.Decode(t); err != nil {
		return ErrDecodeJSON(err.Error())
	}

	if len(t.AccessToken) == 0 {
		return ErrMissingAccessToken
	}

	c.Token = t.AccessToken

	return nil
}
