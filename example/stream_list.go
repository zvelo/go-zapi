package main

import (
	"flag"

	"github.com/kr/pretty"
)

func init() {
	fs := flag.NewFlagSet("stream-list", flag.ExitOnError)
	// fs.Usage = cmdUsage(fs, "url [url...]") // TODO(jrubin)

	cmd["stream-list"] = subcommand{
		FlagSet: fs,
		Action:  streamList,
		Usage:   "list streams",
	}
}

func streamList() error {
	reply, err := zClient.StreamsList()
	if err != nil {
		return err
	}

	// TODO(jrubin)
	pretty.Println(reply)

	return nil
}
