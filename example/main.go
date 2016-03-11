package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"

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
	cmd     = map[string]subcommand{}
	zClient = zapi.New()
)

func init() {
	// global flags

	flag.StringVar(
		&zClient.UserAgent,
		"user-agent",
		getDefaultString("ZVELO_USER_AGENT", name+" "+version),
		"user-agent to use when making requests to zvelo-api [$ZVELO_USER_AGENT]",
	)

	flag.StringVar(
		&zClient.Endpoint,
		"endpoint",
		getDefaultString("ZVELO_ENDPOINT", zapi.DefaultEndpoint),
		"URL of the API endpoint [$ZVELO_ENDPOINT]",
	)

	flag.BoolVar(
		&zClient.Debug,
		"debug",
		getDefaultBool("ZVELO_DEBUG"),
		"enable debug logging [$ZVELO_DEBUG]",
	)

	flag.StringVar(
		&zClient.Token,
		"token",
		getDefaultString("ZVELO_TOKEN", ""),
		"Token for making the query [$ZVELO_TOKEN]",
	)

	flag.StringVar(
		&zClient.Username,
		"username",
		getDefaultString("ZVELO_USERNAME", ""),
		"Username to obtain a token as [$ZVELO_USERNAME]",
	)

	flag.StringVar(
		&zClient.Password,
		"password",
		getDefaultString("ZVELO_PASSWORD", ""),
		"Password to obtain a token with [$ZVELO_PASSWORD]",
	)

	flag.DurationVar(
		&zClient.PollTimeout,
		"timeout",
		zapi.DefaultPollTimeout,
		"timeout after this much time has elapsed",
	)

	flag.DurationVar(
		&zClient.PollInterval,
		"interval",
		zapi.DefaultPollInterval,
		"amount of time between polling requests",
	)
}

func main() {
	// parse flags and run necessary setup
	fn, err := parseFlags()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}

	// execute the command
	if fn != nil {
		if err := fn(); err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", err)
			os.Exit(1)
		}
	}
}

func printCmdUsage() {
	for name, sc := range cmd {
		fmt.Fprintf(os.Stderr, "  %s\n        %s\n", name, sc.Usage)
	}
}

func getDefaultString(envVar, fallback string) string {
	ret := os.Getenv(envVar)
	if len(ret) == 0 {
		return fallback
	}
	return ret
}

func getDefaultBool(envVar string) bool {
	val := os.Getenv(envVar)
	if len(val) == 0 {
		return false
	}

	ret, err := strconv.ParseBool(val)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error parsing bool: %s\n", err)
		return false
	}

	return ret
}

func parseFlags() (func() error, error) {
	// parse global flags
	flag.Parse()

	// run global setup
	if err := setupGlobal(); err != nil {
		flag.Usage()
		printCmdUsage()
		return nil, err
	}

	// ensure command exists
	if len(flag.Args()) == 0 {
		flag.Usage()
		printCmdUsage()
		return nil, fmt.Errorf("command is required")
	}

	// ensure command is valid
	sc, ok := cmd[flag.Args()[0]]
	if !ok {
		printCmdUsage()
		return nil, fmt.Errorf("invalid command")
	}

	// parse command flags
	if sc.FlagSet != nil {
		_ = sc.FlagSet.Parse(flag.Args()[1:])
	}

	// run command setup
	if sc.Setup != nil {
		if err := sc.Setup(); err != nil {
			if sc.FlagSet != nil {
				if sc.FlagSet.Usage != nil {
					sc.FlagSet.Usage()
				} else {
					sc.FlagSet.PrintDefaults()
				}
			}
			return nil, err
		}
	}

	return sc.Action, nil
}

func setupGlobal() error {
	if len(zClient.Token) == 0 &&
		(len(zClient.Username) == 0 || len(zClient.Password) == 0) {
		return fmt.Errorf("-token or -username and -password are required")
	}

	return nil
}

func cmdUsage(fs *flag.FlagSet, usage string) func() {
	exec := os.Args[0]

	return func() {
		fmt.Fprintf(os.Stderr, "Usage of %s %s: %s\n", exec, flag.Args()[0], usage)
		fs.PrintDefaults()
	}
}
