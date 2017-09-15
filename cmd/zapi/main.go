package main

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"os"
	"runtime"
	"strings"

	"github.com/urfave/cli"
	"zvelo.io/go-zapi"
	"zvelo.io/msg"
)

const name = "zapi"

var (
	endpoint               string
	debug, rest            bool
	clientID, clientSecret string
	restClient             zapi.RESTClient
	grpcClient             zapi.GRPCClient
	datasets               []uint32
	forceTrace             bool

	version        = "1.0.0"
	datasetStrings = cli.StringSlice([]string{"CATEGORIZATION"})
	scopes         = cli.StringSlice(strings.Fields(zapi.DefaultScopes))
	app            = cli.NewApp()
)

func init() {
	app.Name = name
	app.Version = fmt.Sprintf("%s (%s)", version, runtime.Version())
	app.Usage = "client utility for zvelo api"
	app.Authors = []cli.Author{
		{Name: "Joshua Rubin", Email: "jrubin@zvelo.com"},
	}
	app.Before = globalSetup

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:        "endpoint",
			EnvVar:      "ZVELO_ENDPOINT",
			Usage:       "URL of the API endpoint",
			Value:       zapi.DefaultEndpoint,
			Destination: &endpoint,
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
		cli.BoolFlag{
			Name:        "rest",
			EnvVar:      "ZVELO_REST",
			Usage:       "Use REST instead of gRPC for api requests",
			Destination: &rest,
		},
		cli.StringSliceFlag{
			Name:   "datasets",
			EnvVar: "ZVELO_DATASETS",
			Usage:  "list of datasets to retrieve (available options: " + strings.Join(availableDS(), ", ") + ")",
			Value:  &datasetStrings,
		},
		cli.StringSliceFlag{
			Name:   "scopes",
			EnvVar: "ZVELO_SCOPES",
			Usage:  "scopes to request with the token",
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

var chars = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

func randString(n int) string {
	b := make([]rune, n)
	for i := range b {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(chars))))
		if err != nil {
			panic(err)
		}
		b[i] = chars[n.Int64()]
	}
	return string(b)
}

func globalSetup(_ *cli.Context) error {
	tokenSourcer := zapi.ClientCredentials(clientID, clientSecret, scopes...)

	zapiOpts := []zapi.Option{
		zapi.WithEndpoint(endpoint),
	}

	if debug {
		zapiOpts = append(zapiOpts, zapi.WithDebug())
	}

	if forceTrace {
		zapiOpts = append(zapiOpts, zapi.WithForceTrace())
	}

	ts := tokenSourcer.TokenSource(context.Background())

	restClient = zapi.NewREST(ts, zapiOpts...)

	grpcDialer := zapi.NewGRPC(ts, zapiOpts...)

	var err error
	grpcClient, err = grpcDialer.Dial(context.Background())
	if err != nil {
		return err
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
