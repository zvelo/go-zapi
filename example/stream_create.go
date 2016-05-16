package main

import (
	"flag"
	"fmt"

	"zvelo.io/msg"
)

type streamHandler struct {
	config struct {
		trackingID string
		endpoint   string
		accept     string
	}
}

var sHandler streamHandler

func init() {
	fs := flag.NewFlagSet("stream-create", flag.ExitOnError)
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

	cmd["stream-create"] = subcommand{
		FlagSet: fs,
		Setup:   setupStreamCreate,
		Action:  streamCreate,
		Usage:   "create a stream",
	}
}

func setupStreamCreate() error {
	if len(sHandler.config.endpoint) == 0 {
		return fmt.Errorf("endpoint is required")
	}

	return nil
}

func streamCreate() error {
	req := &msg.StreamRequest{
		CustomerTrackingId: sHandler.config.trackingID,
		Endpoint:           sHandler.config.endpoint,
		Accept:             sHandler.config.accept,
		Dataset:            datasets,
	}

	reply, err := zClient.StreamCreate(req)
	if err != nil {
		return err
	}

	// TODO(jrubin)
	// pretty.Println(reply)
	fmt.Println(reply)

	return nil
}
