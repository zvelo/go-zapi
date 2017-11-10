package callback

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	"github.com/gogo/protobuf/jsonpb"
	"github.com/pkg/errors"

	"gopkg.in/square/go-jose.v2"

	"zvelo.io/go-zapi/internal/zvelo"
	"zvelo.io/httpsig"
	"zvelo.io/msg"
)

var jsonUnmarshaler jsonpb.Unmarshaler

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
	app     string
	options *options
}

// KeyGetter returns an httpsig.KeyGetter that will properly fetch and cache
// zvelo public keys
func KeyGetter(app string, opts ...Option) httpsig.KeyGetter {
	o := callbackDefaults()
	for _, opt := range opts {
		opt(o)
	}

	return &keyGetter{
		app:     app,
		options: o,
	}
}

func decodePublicKey(rdr io.Reader) (interface{}, error) {
	var keyset jose.JSONWebKeySet
	if err := json.NewDecoder(rdr).Decode(&keyset); err != nil {
		return nil, err
	}

	keys := keyset.Key("public")

	if len(keys) == 0 {
		return nil, errors.New("no public key")
	}

	return keys[0].Key, nil
}

var keyGetterLock sync.Mutex

func (g *keyGetter) GetKey(keyID string) (interface{}, error) {
	var cacheFile string

	// 1. check for key cached in filesystem

	if !g.options.noCache {
		keyGetterLock.Lock()
		defer keyGetterLock.Unlock()

		cacheFile = filepath.Join(zvelo.DataDir, g.app, fmt.Sprintf("key_%x.json", sha256.Sum256([]byte(keyID))))

		// ignore errors since we can always just fetch the key
		if f, err := os.Open(cacheFile); err == nil {
			defer func() { _ = f.Close() }()
			if key, err := decodePublicKey(f); err == nil {
				return key, nil
			}
		}
	}

	// 2. fetch the key

	req, err := http.NewRequest("GET", keyID, nil)
	if err != nil {
		return nil, err
	}

	zvelo.DebugRequestOut(g.options.debug, req)

	resp, err := g.options.client.Do(req)
	if err != nil {
		return nil, err
	}

	zvelo.DebugResponse(g.options.debug, resp)

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.Errorf("unexpected status fetching key: %s", resp.Status)
	}

	if g.options.noCache {
		return decodePublicKey(resp.Body)
	}

	// 3. write the json key to the cache file as we decode it

	if err = os.MkdirAll(filepath.Dir(cacheFile), 0700); err != nil {
		return nil, err
	}

	var f *os.File
	if f, err = os.OpenFile(cacheFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600); err != nil {
		return nil, err
	}

	defer func() { _ = f.Close() }()

	return decodePublicKey(io.TeeReader(resp.Body, f))
}

type options struct {
	debug      io.Writer
	client     Doer
	noValidate bool
	noCache    bool
}

// An Option is used to configure the HTTPHandler
type Option func(*options)

func callbackDefaults() *options {
	return &options{
		debug:  ioutil.Discard,
		client: http.DefaultClient,
	}
}

// WithoutCache prevents the KeyGetter from caching keys to the filesystem
func WithoutCache() Option {
	return func(o *options) { o.noCache = true }
}

// WithKeyGetterClient causes the HTTPHandler to use the passed in
// http.Client, instead of http.DefaultClient
func WithKeyGetterClient(val Doer) Option {
	if val == nil {
		val = http.DefaultClient
	}

	return func(o *options) { o.client = val }
}

// WithDebug causes the HTTPHandler to emit debug logs to the writer
func WithDebug(val io.Writer) Option {
	if val == nil {
		val = ioutil.Discard
	}

	return func(o *options) { o.debug = val }
}

// WithoutValidation causes the HTTPHandler to skip signature validation
func WithoutValidation() Option {
	return func(o *options) { o.noValidate = true }
}

// HTTPHandler returns an http.Handler that can be used with an http.Server
// to receive and process zveloAPI callbacks
func HTTPHandler(app string, h Handler, opts ...Option) http.Handler {
	o := callbackDefaults()
	for _, opt := range opts {
		opt(o)
	}

	handler := http.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		zvelo.DebugRequest(o.debug, r)

		var result msg.QueryResult
		if err := jsonUnmarshaler.Unmarshal(r.Body, &result); err == nil {
			h.Handle(&result)
		}
	}))

	if !o.noValidate {
		handler = httpsig.Middleware(httpsig.SignatureHeader, KeyGetter(app, opts...), handler)
	}

	return handler
}
