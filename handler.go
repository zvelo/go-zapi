package zapi

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/golang/protobuf/proto"
)

type handler interface {
	PrepareReq(query interface{}) (*http.Request, error)
	ParseResp(body io.Reader, reply interface{}) error
}

type req struct {
	ContentType string
	Accept      string
	URL         string
	Method      string
}

type jsonHandler struct {
	req
}

type pbHandler struct {
	req
}

func (r req) PrepareReq(data []byte) (*http.Request, error) {
	req, err := http.NewRequest(r.Method, r.URL, bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", r.Accept)
	req.Header.Set("Content-Type", r.ContentType)

	return req, nil
}

func (h jsonHandler) PrepareReq(query interface{}) (*http.Request, error) {
	data, err := json.Marshal(query)
	if err != nil {
		return nil, err
	}

	return h.req.PrepareReq(data)
}

func (h jsonHandler) ParseResp(body io.Reader, reply interface{}) error {
	decoder := json.NewDecoder(body)
	if err := decoder.Decode(reply); err != nil {
		return errDecodeJSON(err.Error())
	}

	return nil
}

func (h pbHandler) PrepareReq(query interface{}) (*http.Request, error) {
	msg, ok := query.(proto.Message)
	if !ok {
		panic("invalid query type passed to pbHandler.PrepareReq")
	}

	data, err := proto.Marshal(msg)
	if err != nil {
		return nil, err
	}

	return h.req.PrepareReq(data)
}

func (h pbHandler) ParseResp(body io.Reader, reply interface{}) error {
	msg, ok := reply.(proto.Message)
	if !ok {
		panic("invalid reply type passed to pbHandler.ParseResp")
	}

	data, err := ioutil.ReadAll(body)
	if err != nil {
		return err
	}

	if err := proto.Unmarshal(data, msg); err != nil {
		return err
	}

	return nil
}
