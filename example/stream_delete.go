package main

import (
	"flag"
	"fmt"
)

var deleteStreamUUID []string

func init() {
	fs := flag.NewFlagSet("stream-delete", flag.ExitOnError)
	// fs.Usage = cmdUsage(fs, "url [url...]") // TODO(jrubin)

	cmd["stream-delete"] = subcommand{
		FlagSet: fs,
		Setup:   setupStreamDelete,
		Action:  streamDelete,
		Usage:   "delete streams",
	}
}

func setupStreamDelete() error {
	deleteStreamUUID = cmd["stream-delete"].FlagSet.Args()
	if len(deleteStreamUUID) == 0 {
		return fmt.Errorf("it least one uuid is required")
	}
	return nil
}

func streamDelete() error {
	for _, uuid := range deleteStreamUUID {
		reply, err := zClient.StreamDelete(uuid)
		if err != nil {
			return err
		}

		// TODO(jrubin)
		// pretty.Println(reply)
		fmt.Println(reply)
	}

	return nil
}
