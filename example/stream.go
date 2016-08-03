package main

import (
	"fmt"

	"github.com/urfave/cli"
	"zvelo.io/msg"
)

type streamHandler struct {
	config struct {
		trackingID string
		endpoint   string
		accept     string
	}
}

var (
	sHandler         streamHandler
	updateStreamUUID []string
	deleteStreamUUID []string
)

func init() {
	createUpdateFlags := []cli.Flag{
		cli.StringFlag{
			Name:   "tracking-id, tid",
			EnvVar: "ZVELO_STREAM_TRACKING_ID",
			Usage:  "customer tracking id",
		},
		cli.StringFlag{
			Name:   "endpoint",
			EnvVar: "ZVELO_STREAM_ENDPOINT",
			Usage:  "endpoint towards which the stream is directed",
		},
		cli.StringFlag{
			Name:   "accept",
			EnvVar: "ZVELO_STREAM_ACCEPT",
			Usage:  "The MIME type required for submission to the endpoint",
		},
	}
	app.Commands = append(app.Commands, cli.Command{
		Name:  "stream",
		Usage: "used for stream api",
		Subcommands: []cli.Command{
			cli.Command{
				Name:   "create",
				Usage:  "create a stream",
				Before: setupStreamCreate,
				Action: streamCreate,
				Flags:  createUpdateFlags,
			},
			cli.Command{
				Name:   "list",
				Usage:  "list streams",
				Before: setupStreamList,
				Action: streamList,
			},
			cli.Command{
				Name:   "update",
				Usage:  "update streams",
				Before: setupStreamUpdate,
				Action: streamUpdate,
				Flags: append(createUpdateFlags, cli.StringSliceFlag{
					Name:  "uuid",
					Usage: "list of URLs to query for",
				},
				),
			},
			cli.Command{
				Name:   "delete",
				Usage:  "delete streams",
				Before: setupStreamDelete,
				Action: streamDelete,
				Flags: []cli.Flag{
					cli.StringSliceFlag{
						Name:  "uuid",
						Usage: "list of URLs to query for",
					},
				},
			},
		},
	})
}

func setupStreamCreate(c *cli.Context) error {
	if err := setupClient(c); err != nil {
		return err
	}

	if err := setupDS(c); err != nil {
		return err
	}

	sHandler.config.endpoint = c.String("endpoint")
	sHandler.config.trackingID = c.String("tracking-id")
	sHandler.config.accept = c.String("accept")

	if len(sHandler.config.endpoint) == 0 {
		return fmt.Errorf("endpoint is required")
	}

	return nil
}

func streamCreate(c *cli.Context) error {
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

	//TODO(rnanjegowda)
	//Create a stream listener

	// TODO(jrubin)
	// result.Pretty(os.Stdout)
	fmt.Println(reply)

	return nil
}

func setupStreamList(c *cli.Context) error {
	if err := setupClient(c); err != nil {
		return err
	}

	return nil
}

func streamList(c *cli.Context) error {
	reply, err := zClient.StreamsList()
	if err != nil {
		return err
	}

	// TODO(jrubin)
	// result.Pretty(os.Stdout)
	fmt.Println(reply)

	return nil
}

func setupStreamUpdate(c *cli.Context) error {
	if err := setupClient(c); err != nil {
		return err
	}

	updateStreamUUID = c.StringSlice("uuid")
	if len(updateStreamUUID) == 0 {
		return fmt.Errorf("at least one uuid is required")
	}

	return nil
}

func streamUpdate(c *cli.Context) error {
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
		// result.Pretty(os.Stdout)
		fmt.Println(reply)
	}

	return nil
}

func setupStreamDelete(c *cli.Context) error {
	if err := setupClient(c); err != nil {
		return err
	}

	deleteStreamUUID = c.StringSlice("uuid")
	if len(deleteStreamUUID) == 0 {
		return fmt.Errorf("at least one uuid is required")
	}

	return nil
}

func streamDelete(c *cli.Context) error {
	for _, uuid := range deleteStreamUUID {
		reply, err := zClient.StreamDelete(uuid)
		if err != nil {
			fmt.Println(err)
		}
		// TODO(jrubin)
		// result.Pretty(os.Stdout)
		fmt.Println(reply)
	}
	return nil
}
