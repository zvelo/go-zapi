package main

import (
	"flag"
	"fmt"
	"html"
	"net/http"
)

var callbackConfig = struct {
	URLs                         []string
	ListenAddress, PublicBaseURL string
}{}

func init() {
	fs := flag.NewFlagSet("callback", flag.ExitOnError)

	fs.StringVar(&zClient.Token, "token", getDefaultString("ZVELO_TOKEN", ""), "Token for making the query [$ZVELO_TOKEN]")
	fs.StringVar(&zClient.Username, "username", getDefaultString("ZVELO_USERNAME", ""), "Username to obtain a token as [$ZVELO_USERNAME]")
	fs.StringVar(&zClient.Password, "password", getDefaultString("ZVELO_PASSWORD", ""), "Password to obtain a token with [$ZVELO_PASSWORD]")
	fs.StringVar(&callbackConfig.ListenAddress, "listen-address", getDefaultString("ZVELO_CALLBACK_ADDRESS", "[::1]:8080"), "address and port to listen for callbacks on [$ZVELO_CALLBACK_ADDRESS or [::1]:8080]")
	fs.StringVar(&callbackConfig.PublicBaseURL, "public-base-url", getDefaultString("ZVELO_PUBLIC_BASE_URL", ""), "publicly accessible base URL that routes to the address used by the address flag [$ZVELO_PUBLIC_BASE_URL]")

	cmd["callback"] = subcommand{
		FlagSet: fs,
		Setup:   setupCallback,
		Action:  callbackURL,
		Usage:   "query url using callback",
	}
}

func setupCallback() error {
	if len(zClient.Token) == 0 &&
		(len(zClient.Username) == 0 || len(zClient.Password) == 0) {
		return fmt.Errorf("-token or -username and -password are required")
	}

	callbackConfig.URLs = cmd["callback"].FlagSet.Args()

	if len(callbackConfig.URLs) == 0 {
		return fmt.Errorf("at least one url is required")
	}

	if len(callbackConfig.PublicBaseURL) == 0 {
		return fmt.Errorf("-public-base-url is required")
	}

	return nil
}

func callbackURL() error {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// TODO(jrubin)
		fmt.Fprintf(w, "Hello, %q", html.EscapeString(r.URL.Path))
	})

	return http.ListenAndServe(callbackConfig.ListenAddress, nil)
}
