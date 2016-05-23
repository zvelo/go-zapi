package main

import (
	"flag"
	"fmt"

	"zvelo.io/msg"
)

var updateStreamUUID []string

func init() {
	fs := flag.NewFlagSet("stream-update", flag.ExitOnError)
	// fs.Usage = cmdUsage(fs, "url [url...]") // TODO(jrubin)

	fs.StringVar(
		&sHandler.config.trackingID,
		"tracking-id",
		getDefaultString("ZVELO_STREAM_TRACKING_ID", ""),
		"customer tracking id [$ZVELO_STREAM_TRACKING_ID]",
	)

	fs.StringVar(
		&sHandler.config.endpoint,
		"endpoint",
		getDefaultString("ZVELO_STREAM_ENDPOINT", ""),
		"endpoint towards which the stream is directed [$ZVELO_STREAM_ENDPOINT]",
	)

	fs.StringVar(
		&sHandler.config.accept,
		"accept",
		getDefaultString("ZVELO_STREAM_ACCEPT", ""), // TODO(jrubin) set default
		"endpoint towards which the stream is directed [$ZVELO_STREAM_ACCEPT]",
	)

	cmd["stream-update"] = subcommand{
		FlagSet: fs,
		Setup:   setupStreamUpdate,
		Action:  streamUpdate,
		Usage:   "update streams",
	}
}

func setupStreamUpdate() error {
	updateStreamUUID = cmd["stream-update"].FlagSet.Args()
	if len(updateStreamUUID) == 0 {
		return fmt.Errorf("it least one uuid is required")
	}

	return nil
}

func streamUpdate() error {
	for _, uuid := range updateStreamUUID {
		req := &msg.StreamRequest{
			CustomerTrackingId: sHandler.config.trackingID,
			Endpoint:           sHandler.config.endpoint,
			Accept:             sHandler.config.accept,
			Dataset:            datasets,
		}

		reply, err := zClient.StreamUpdate(uuid, req)
		if err != nil {
			return err
		}

		// TODO(jrubin)
		// pretty.Println(reply)
		fmt.Println(reply)
	}

	return nil
}
