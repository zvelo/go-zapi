package main

import (
	"flag"
	"fmt"
	"html"
	"net/http"
	"time"

	"github.com/zvelo/go-zapi/zapitype"
)

const (
	callbackDefaultListenAddress = "[::1]:8080"
	callbackDefaultTimeout       = 15 * time.Minute
)

var callbackConfig = struct {
	URLs                       []string
	ListenAddress, CallbackURL string
	Timeout                    time.Duration
	PartialResults             bool
}{}

func init() {
	fs := flag.NewFlagSet("callback", flag.ExitOnError)

	fs.StringVar(&callbackConfig.ListenAddress, "listen-address", getDefaultString("ZVELO_LISTEN_ADDRESS", callbackDefaultListenAddress), "address and port to listen for callbacks on [$ZVELO_LISTEN_ADDRESS]")
	fs.StringVar(&callbackConfig.CallbackURL, "callback-url", getDefaultString("ZVELO_CALLBACK_URL", ""), "publicly accessible base URL that routes to the address used by the address flag [$ZVELO_CALLBACK_URL]")
	fs.DurationVar(&callbackConfig.Timeout, "timeout", callbackDefaultTimeout, "maximum amount of time to wait for the callback to be called")
	fs.BoolVar(&callbackConfig.PartialResults, "partial-results", false, "request that datasets be delivered as soon as they become available instead of waiting for all datasets to become available before responding")

	cmd["callback"] = subcommand{
		FlagSet: fs,
		Setup:   setupCallback,
		Action:  callbackURL,
		Usage:   "query url using callback",
	}
}

func setupCallback() error {
	callbackConfig.URLs = cmd["callback"].FlagSet.Args()

	if len(callbackConfig.CallbackURL) == 0 {
		return fmt.Errorf("-callback-url is required")
	}

	if len(callbackConfig.URLs) == 0 {
		return fmt.Errorf("at least one url is required")
	}

	return nil
}

func callbackURL() error {
	fmt.Println("URLs", callbackConfig.URLs)

	doneCh := make(chan struct{}, 1)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// TODO(jrubin)
		fmt.Fprintf(w, "Hello, %q", html.EscapeString(r.URL.Path))
		doneCh <- struct{}{}
	})

	go http.ListenAndServe(callbackConfig.ListenAddress, nil)

	reply, err := zClient.Query(&zapitype.QueryURLRequests{
		URLs: callbackConfig.URLs,
		DataSets: []zapitype.DataSetType{
			zapitype.DataSetTypeCategorization,
			zapitype.DataSetTypeAdFraud,
		},
		CallbackURL:    callbackConfig.CallbackURL,
		PartialResults: callbackConfig.PartialResults,
	})
	if err != nil {
		return err
	}

	// TODO(jrubin) assert(len(reply.RequestIDs) > 0)
	fmt.Println(reply) // TODO(jrubin)

	select {
	case <-doneCh:
		// TODO(jrubin)
		return nil
	case <-time.After(callbackConfig.Timeout):
		// TODO(jrubin)
		return nil
	}
}
