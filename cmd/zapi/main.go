package main

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/urfave/cli"

	"golang.org/x/oauth2"

	"zvelo.io/go-zapi"
	"zvelo.io/go-zapi/tokensource"
	"zvelo.io/go-zapi/userauth"
	"zvelo.io/msg"
)

const name = "zapi"

var (
	host                   string
	debug, rest            bool
	restClient             zapi.RESTClient
	grpcClient             zapi.GRPCClient
	datasets               []uint32
	forceTrace             bool
	pollInterval           time.Duration
	timeout                time.Duration
	noCacheToken           bool
	zapiOpts               []zapi.Option
	tokenSource            oauth2.TokenSource
	useUserCredentials     bool
	scopes                 cli.StringSlice
	datasetStrings         cli.StringSlice
	clientID, clientSecret string
	accessToken            string
	redirectURL            string
	callbackAddr           string
	noOpenBrowser          bool

	version         = "1.0.0"
	app             = cli.NewApp()
	defaultScopes   = strings.Fields(zapi.DefaultScopes)
	defaultDatasets = []string{msg.CATEGORIZATION.String()}
)

func init() {
	app.Name = name
	app.Version = fmt.Sprintf("%s (%s)", version, runtime.Version())
	app.Usage = "client utility for zvelo api"
	app.EnableBashCompletion = true
	app.BashComplete = bashComplete
	app.Authors = []cli.Author{
		{Name: "Joshua Rubin", Email: "jrubin@zvelo.com"},
	}
	app.Before = globalSetup

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:        "host",
			EnvVar:      "ZVELO_HOST",
			Usage:       "hostname of the API endpoint",
			Value:       zapi.DefaultHost,
			Destination: &host,
		},
		cli.BoolFlag{
			Name:        "debug",
			EnvVar:      "ZVELO_DEBUG",
			Usage:       "enable debug logging",
			Destination: &debug,
		},
		cli.StringFlag{
			Name:        "client-id",
			EnvVar:      "ZVELO_CLIENT_ID",
			Usage:       "oauth2 client id",
			Destination: &clientID,
		},
		cli.StringFlag{
			Name:        "client-secret",
			EnvVar:      "ZVELO_CLIENT_SECRET",
			Usage:       "oauth2 client secret",
			Destination: &clientSecret,
		},
		cli.StringFlag{
			Name:        "access-token",
			EnvVar:      "ZVELO_ACCESS_TOKEN",
			Usage:       "explicitly provide an access token. this should rarely be used as it will override client-id, client-secret and user-credentials",
			Destination: &accessToken,
		},
		cli.BoolFlag{
			Name:        "use-user-credentials",
			EnvVar:      "ZVELO_USE_USER_CREDENTIALS",
			Usage:       "use user, 3 legged oauth2, credentials instead of client credentials",
			Destination: &useUserCredentials,
		},
		cli.StringFlag{
			Name:        "oauth2-callback-url",
			EnvVar:      "ZVELO_OAUTH2_CALLBACK_URL",
			Usage:       "url that server will redirect to for oauth2 callbacks",
			Value:       userauth.DefaultRedirectURL,
			Destination: &redirectURL,
		},
		cli.StringFlag{
			Name:        "oauth2-callback-addr",
			EnvVar:      "ZVELO_OAUTH2_CALLBACK_ADDR",
			Usage:       "addr:port that server will listen to for oauth2 callbacks",
			Value:       userauth.DefaultCallbackAddr,
			Destination: &callbackAddr,
		},
		cli.BoolFlag{
			Name:        "oauth2-no-open-in-browser",
			EnvVar:      "ZVELO_OAUTH2_NO_OPEN_IN_BROWSER",
			Usage:       "don't open the auth url in the browser",
			Destination: &noOpenBrowser,
		},
		cli.DurationFlag{
			Name:        "poll-interval",
			EnvVar:      "ZVELO_POLL_INTERVAL",
			Usage:       "repeatedly poll after this much time has elapsed until the request is marked as complete",
			Value:       1 * time.Second,
			Destination: &pollInterval,
		},
		cli.DurationFlag{
			Name:        "timeout",
			EnvVar:      "ZVELO_TIMEOUT",
			Usage:       "maximum amount of time to wait for results to complete",
			Value:       15 * time.Minute,
			Destination: &timeout,
		},
		cli.BoolFlag{
			Name:        "rest",
			EnvVar:      "ZVELO_REST",
			Usage:       "Use REST instead of gRPC for api requests",
			Destination: &rest,
		},
		cli.BoolFlag{
			Name:        "no-cache-token",
			EnvVar:      "ZVELO_NO_CACHE_TOKEN",
			Usage:       "don't cache received oauth2 tokens to the filesystem",
			Destination: &noCacheToken,
		},
		cli.StringSliceFlag{
			Name:   "dataset",
			EnvVar: "ZVELO_DATASETS",
			Usage:  "list of datasets to retrieve, may be repeated (available options: " + strings.Join(availableDS(), ", ") + ", default: " + strings.Join(defaultDatasets, ", ") + ")",
			Value:  &datasetStrings,
		},
		cli.StringSliceFlag{
			Name:   "scope",
			EnvVar: "ZVELO_SCOPES",
			Usage:  "scopes to request with the token, may be repeated (default: " + strings.Join(defaultScopes, ", ") + ")",
			Value:  &scopes,
		},
		cli.BoolFlag{
			Name:        "force-trace",
			EnvVar:      "ZVELO_FORCE_TRACE",
			Usage:       "force a trace to be generated for each request",
			Destination: &forceTrace,
		},
	}
}

func main() {
	if err := app.Run(os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}

func setupTokenSource() {
	if len(scopes) == 0 {
		scopes = defaultScopes
	}

	var cacheName string

	if accessToken != "" {
		tokenSource = oauth2.StaticTokenSource(&oauth2.Token{
			AccessToken: accessToken,
		})
	} else if useUserCredentials {
		cacheName = "user"
		userOpts := []userauth.Option{
			userauth.WithRedirectURL(redirectURL),
			userauth.WithScope(scopes...),
			userauth.WithCallbackAddr(callbackAddr),
		}

		if noOpenBrowser {
			userOpts = append(userOpts, userauth.WithoutOpen())
		}

		if debug {
			userOpts = append(userOpts, userauth.WithDebug())
		}

		tokenSource = userauth.TokenSource(context.Background(), clientID, clientSecret, userOpts...)
	} else {
		cacheName = "client"
		tokenSource = zapi.ClientCredentials(
			context.Background(),
			clientID,
			clientSecret,
			scopes...,
		)
	}

	if accessToken == "" {
		if !noCacheToken {
			tokenSource = tokensource.FileCache(tokenSource, "zapi", cacheName, scopes...)
		}

		tokenSource = oauth2.ReuseTokenSource(nil, tokenSource)
	}

	if debug {
		tokenSource = tokensource.Log(tokenSource)
	}
}

func setupZapiOpts() {
	zapiOpts = append(zapiOpts, zapi.WithHost(host))

	if debug {
		zapiOpts = append(zapiOpts, zapi.WithDebug())
	}

	if forceTrace {
		zapiOpts = append(zapiOpts, zapi.WithForceTrace())
	}
}

func setupDataSets() error {
	if len(datasetStrings) == 0 {
		datasetStrings = defaultDatasets
	}

	for _, dsName := range datasetStrings {
		dsName = strings.TrimSpace(dsName)

		dst, err := msg.NewDataSetType(dsName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "invalid dataset type: %s\n", dsName)
			continue
		}

		datasets = append(datasets, uint32(dst))
	}

	if len(datasets) == 0 {
		return errors.New("at least one valid dataset is required")
	}

	return nil
}

func globalSetup(_ *cli.Context) error {
	setupTokenSource()
	setupZapiOpts()

	restClient = zapi.NewREST(tokenSource, zapiOpts...)
	grpcDialer := zapi.NewGRPC(tokenSource, zapiOpts...)

	var err error
	if grpcClient, err = grpcDialer.Dial(context.Background()); err != nil {
		return err
	}

	return setupDataSets()
}

func availableDS() []string {
	var ds []string
	for dst, name := range msg.DataSetType_name {
		if dst == int32(msg.ECHO) {
			continue
		}

		ds = append(ds, name)
	}
	return ds
}
