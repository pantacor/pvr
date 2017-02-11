/*
 * Copyright 2017  Alexander Sack <asac129@gmail.com>
 */
package main

import (
	"fmt"
	"os"

	"github.com/urfave/cli"
)

var (
	hubBaseUrl string
)

func main() {

	hubBaseUrl = "https://pantahub.appspot.com/api"

	app := cli.NewApp()
	app.Name = "pvr"
	app.Usage = "PantaVisor Remote"
	app.Version = "0.0.1"
	app.Action = func(c *cli.Context) error {
		fmt.Println("boom! I say!")
		return nil
	}

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "access-token, a",
			Usage: "Use `ACCESS_TOKEN` for authorization with core services",
		},
		cli.StringFlag{
			Name:  "baseurl, b",
			Usage: "Use `BASEURL` for resolving prn URIs to core service endpoints",
		},
	}

	if os.Getenv("PANTAHUB_BASE") != "" {
		hubBaseUrl = os.Getenv("PANTAHUB_BASE")
	}

	app.Commands = []cli.Command{
		CommandInit(),
		CommandAdd(),
		CommandJson(),
		CommandDiff(),
		CommandStatus(),
		CommandCommit(),
		CommandPush(),
	}
	app.Run(os.Args)
}
