package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/zvelo/go-zapi"
)

type subcommand struct {
	FlagSet *flag.FlagSet
	Setup   func() error
	Action  func() error
	Usage   string
}

const (
	name    = "zvelo-api-example-go"
	version = "0.1.0"
)

var (
	cmd = map[string]subcommand{}

	globalConfig = struct {
		Endpoint string
	}{}

	zClient = zapi.New()
)

func init() {
	zClient.HTTPClient = &http.Client{}

	// global flags
	flag.StringVar(&zClient.UserAgent, "user-agent", getDefaultString("ZVELO_USER_AGENT", name+" "+version), "user-agent to use when making requests to zvelo-api [$ZVELO_USER_AGENT]")
	flag.StringVar(&globalConfig.Endpoint, "endpoint", getDefaultString("ZVELO_ENDPOINT", ""), "URL of the API endpoint [$ZVELO_ENDPOINT]")
	flag.BoolVar(&zClient.Debug, "debug", getDefaultBool("ZVELO_DEBUG", false), "enable debug logging [$ZVELO_DEBUG]")
}

func main() {
	fn, err := parseFlags()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}

	if err := fn(); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}

func usageCmds() string {
	ret := []string{}
	for name, sc := range cmd {
		ret = append(ret, fmt.Sprintf("  %s\n        %s", name, sc.Usage))
	}
	return strings.Join(ret, "\n")
}

func getDefaultString(envVar, fallback string) string {
	ret := os.Getenv(envVar)
	if len(ret) == 0 {
		return fallback
	}
	return ret
}

func getDefaultBool(envVar string, fallback bool) bool {
	val := os.Getenv(envVar)
	if len(val) == 0 {
		return fallback
	}

	ret, err := strconv.ParseBool(val)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error parsing bool: %s\n", err)
		return fallback
	}

	return ret
}

func parseFlags() (func() error, error) {
	// parse global flags
	flag.Parse()
	if err := setupGlobal(); err != nil {
		flag.PrintDefaults()
		return nil, err
	}

	// parse command
	if len(flag.Args()) == 0 {
		flag.PrintDefaults()
		return nil, fmt.Errorf("%s\ncommand is required", usageCmds())
	}

	sc, ok := cmd[flag.Args()[0]]
	if !ok {
		return nil, fmt.Errorf("%s\ninvalid command", usageCmds())
	}

	_ = sc.FlagSet.Parse(flag.Args()[1:])

	if err := sc.Setup(); err != nil {
		sc.FlagSet.PrintDefaults()
		return nil, err
	}

	return sc.Action, nil
}

func setupGlobal() error {
	if len(globalConfig.Endpoint) > 0 {
		if err := zClient.SetEndpoint(globalConfig.Endpoint); err != nil {
			return err
		}
	}

	return nil
}
