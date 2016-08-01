package main

import (
	"fmt"

	"github.com/urfave/cli"
)

func init() {
	app.Commands = append(app.Commands, cli.Command{
		Name:   "get-token",
		Usage:  "get token using username and password",
		Action: getToken,
		Before: setupGetToken,
	})
}

func setupGetToken(c *cli.Context) error {
	if err := setupClient(c); err != nil {
		return err
	}
	
	return nil
}

func getToken(c *cli.Context) error {
	if err := zClient.GetToken(); err != nil {
		return err
	}

	fmt.Printf("ZVELO_TOKEN=%s\n", zClient.Token)

	return nil
}
