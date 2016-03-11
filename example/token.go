package main

import (
	"flag"
	"fmt"
)

func init() {
	fs := flag.NewFlagSet("get-token", flag.ExitOnError)

	cmd["get-token"] = subcommand{
		FlagSet: fs,
		Action:  getToken,
		Usage:   "get token using username and password",
	}
}

func getToken() error {
	if err := zClient.GetToken(); err != nil {
		return err
	}

	fmt.Printf("ZVELO_TOKEN=%s\n", zClient.Token)

	return nil
}
