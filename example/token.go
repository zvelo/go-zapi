package main

import (
	"flag"
	"fmt"
)

func init() {
	fs := flag.NewFlagSet("token", flag.ExitOnError)

	fs.StringVar(&zClient.Username, "username", getDefaultString("ZVELO_USERNAME", ""), "Username to obtain a token as [$ZVELO_USERNAME]")
	fs.StringVar(&zClient.Password, "password", getDefaultString("ZVELO_PASSWORD", ""), "Password to obtain a token with [$ZVELO_PASSWORD]")

	cmd["token"] = subcommand{
		FlagSet: fs,
		Setup:   setupToken,
		Action:  getToken,
		Usage:   "get token using username and password",
	}
}

func setupToken() error {
	if len(zClient.Username) == 0 || len(zClient.Password) == 0 {
		return fmt.Errorf("-username and -password are required")
	}

	return nil
}

func getToken() error {
	if err := zClient.GetToken(); err != nil {
		return err
	}

	fmt.Printf("ZVELO_TOKEN=%s\n", zClient.Token)

	return nil
}
