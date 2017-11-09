package zapi

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"sync"

	"github.com/pkg/errors"

	"gopkg.in/square/go-jose.v2"

	"zvelo.io/go-zapi/internal/zvelo"
	"zvelo.io/httpsig"
	"zvelo.io/msg"
)

// A Handler responds to a zveloAPI callback
type Handler interface {
	Handle(*msg.QueryResult)
}

// The HandlerFunc type is an adapter to allow the use of ordinary functions as
// zveloAPI handlers. If f is a function with the appropriate signature,
// HandlerFunc(f) is a Handler that calls f.
type HandlerFunc func(*msg.QueryResult)

// Handle calls f(in)
func (f HandlerFunc) Handle(in *msg.QueryResult) {
	f(in)
}

var _ Handler = (*HandlerFunc)(nil)

// Doer is an abstraction that is satisfied by http.Client
type Doer interface {
	Do(*http.Request) (*http.Response, error)
}

type keyGetter struct {
	options *callbackOptions

	sync.RWMutex
	cache map[string]interface{}
}

func (g *keyGetter) GetKey(keyID string) (key interface{}, err error) {
	g.RLock()
	if key, ok := g.cache[keyID]; ok {
		g.RUnlock()
		return key, nil
	}

	g.RUnlock()
	g.Lock()
	defer g.Unlock()

	if key, ok := g.cache[keyID]; ok {
		return key, nil
	}

	req, err := http.NewRequest("GET", keyID, nil)
	if err != nil {
		return nil, err
	}

	zvelo.DebugRequestOut(g.options.debug, req)

	resp, err := g.options.client.Do(req)
	if err != nil {
		return nil, nil
	}

	zvelo.DebugResponse(g.options.debug, resp)

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.Errorf("unexpected status fetching key: %s", resp.Status)
	}

	var keyset jose.JSONWebKeySet
	if err = json.NewDecoder(resp.Body).Decode(&keyset); err != nil {
		return nil, err
	}

	keys := keyset.Key("public")
	if len(keys) == 0 {
		return nil, errors.New("no public key in response")
	}

	return keys[0].Key, nil
}

// KeyGetter returns an httpsig.KeyGetter that will properly fetch and cache
// zvelo public keys
func KeyGetter(opts ...CallbackOption) httpsig.KeyGetter {
	o := callbackDefaults()
	for _, opt := range opts {
		opt(o)
	}

	return &keyGetter{options: o}
}

type callbackOptions struct {
	debug  io.Writer
	client Doer
}

// A CallbackOption is used to configure the CallbackHandler
type CallbackOption func(*callbackOptions)

func callbackDefaults() *callbackOptions {
	return &callbackOptions{
		debug:  ioutil.Discard,
		client: http.DefaultClient,
	}
}

// WithKeyGetterClient causes the CallbackHandler to use the passed in
// http.Client, instead of http.DefaultClient
func WithKeyGetterClient(val Doer) CallbackOption {
	if val == nil {
		val = http.DefaultClient
	}

	return func(o *callbackOptions) { o.client = val }
}

// WithCallbackDebug causes the CallbackHandler to emit debug logs to the writer
func WithCallbackDebug(val io.Writer) CallbackOption {
	if val == nil {
		val = ioutil.Discard
	}

	return func(o *callbackOptions) { o.debug = val }
}

// CallbackHandler returns an http.Handler that can be used with an http.Server
// to receive and process zveloAPI callbacks
func CallbackHandler(h Handler, opts ...CallbackOption) http.Handler {
	o := callbackDefaults()
	for _, opt := range opts {
		opt(o)
	}

	return httpsig.Middleware(httpsig.SignatureHeader, KeyGetter(opts...), http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		zvelo.DebugRequest(o.debug, r)

		var result msg.QueryResult
		if err := jsonUnmarshaler.Unmarshal(r.Body, &result); err == nil {
			h.Handle(&result)
		}
	}))
}
