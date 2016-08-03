package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/urfave/cli"
	"zvelo.io/go-zapi"
	"zvelo.io/msg"
)

const (
	name    = "zvelo-api-example-go"
	version = "0.1.0"
)

var (
	zClient  = zapi.New()
	datasets = []msg.DataSetType{}
	app      = initApp()
)

func initApp() *cli.App {
	nApp := cli.NewApp()
	nApp.Name = name
	nApp.Version = version
	nApp.Usage = "example client for zvelo api"
	nApp.Authors = []cli.Author{
		{Name: "Joshua Rubin", Email: "jrubin@zvelo.com"},
		{Name: "RJ Nanjegowda", Email: "rnanjegowda@zvelo.com"},
	}

	// Global flags
	nApp.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   "user-agent, ua",
			EnvVar: "ZVELO_USER_AGENT",
			Usage:  "user-agent to use when making requests to zvelo-api",
			Value:  name + " " + version,
		},
		cli.StringFlag{
			Name:   "endpoint, ep",
			EnvVar: "ZVELO_ENDPOINT",
			Usage:  "URL of the API endpoint",
			Value:  zapi.DefaultEndpoint,
		},
		cli.BoolFlag{
			Name:   "debug, d",
			EnvVar: "ZVELO_DEBUG",
			Usage:  "enable debug logging",
		},
		cli.StringFlag{
			Name:   "token, tk",
			EnvVar: "ZVELO_TOKEN",
			Usage:  "Token for making the query",
		},
		cli.StringFlag{
			Name:   "username, u",
			EnvVar: "ZVELO_USERNAME",
			Usage:  "Username to obtain a token",
		},
		cli.StringFlag{
			Name:   "password, p",
			EnvVar: "ZVELO_PASSWORD",
			Usage:  "Password to obtain a token",
		},
		cli.DurationFlag{
			Name:  "timeout, to",
			Usage: "timeout after this much time has elapsed",
			Value: zapi.DefaultPollTimeout,
		},
		cli.DurationFlag{
			Name:  "interval, in",
			Usage: "amount of time between polling requests",
			Value: zapi.DefaultPollInterval,
		},
		cli.BoolFlag{
			Name:   "json, j",
			EnvVar: "ZVELO_JSON",
			Usage:  "Use json instead of protocol buffers for api requests",
		},
		cli.StringSliceFlag{
			Name:   "dataset, ds",
			EnvVar: "ZVELO_DATASET",
			Usage:  "list of datasets to retrieve (available options: " + strings.Join(availableDS(), ", ") + ")",
		},
	}
	return nApp
}

func main() {
	if err := app.Run(os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
	}
}

func setupClient(c *cli.Context) error {
	zClient.Endpoint, zClient.UserAgent, zClient.Debug = c.GlobalString("endpoint"), c.GlobalString("useragent"), c.GlobalBool("debug")
	zClient.Token, zClient.Username, zClient.Password = c.GlobalString("token"), c.GlobalString("username"), c.GlobalString("password")
	zClient.PollTimeout, zClient.PollInterval = c.GlobalDuration("timeout"), c.GlobalDuration("interval")
	zClient.JSON = c.GlobalBool("json")

	if len(zClient.Token) == 0 &&
		(len(zClient.Username) == 0 || len(zClient.Password) == 0) {
		return fmt.Errorf("-token or -username and -password are required")
	}

	return nil
}

func setupDS(c *cli.Context) error {
	for _, dsName := range c.GlobalStringSlice("dataset") {
		dst, err := msg.NewDataSetType(strings.TrimSpace(dsName))
		if err != nil {
			fmt.Fprintf(os.Stderr, "invalid dataset type: %s\n", dsName)
			continue
		}
		datasets = append(datasets, dst)
	}

	if len(datasets) == 0 {
		return fmt.Errorf("at least one valid dataset is required")
	}

	return nil
}

func availableDS() []string {
	allDatasets := make([]string, len(msg.DataSetType_name)-1)
	i := 0
	for dst, name := range msg.DataSetType_name {
		if dst == int32(msg.DataSetType_ECHO) {
			continue
		}

		allDatasets[i] = name
		i++
	}
	return allDatasets
}
