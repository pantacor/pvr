/*
 * Copyright 2017  Alexander Sack <asac129@gmail.com>
 */
package main

import (
	"os"

	"github.com/urfave/cli"
)

func CommandCommit() cli.Command {
	return cli.Command{
		Name:      "commit",
		Aliases:   []string{"c"},
		ArgsUsage: "",
		Usage:     "commit status changes",
		Action: func(c *cli.Context) error {
			wd, err := os.Getwd()
			if err != nil {
				return err
			}

			pvr, err := NewPvr(wd)
			if err != nil {
				return err
			}

			commitmsg := c.String("message")
			if commitmsg == "" {
				commitmsg = "** No commit message **"
			}
			err = pvr.Commit(commitmsg)
			if err != nil {
				return err
			}

			return nil
		},
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "message, m",
				Usage: "provide a commit message",
			},
		},
	}
}
