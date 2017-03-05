/*
 * Copyright 2017  Alexander Sack <asac129@gmail.com>
 */
package main

import (
	"errors"
	"os"

	"github.com/urfave/cli"
)

func CommandPost() cli.Command {
	return cli.Command{
		Name:        "post",
		Aliases:     []string{"po"},
		ArgsUsage:   "[target-log]",
		Usage:       "Post local repository to a target log",
		Description: "Suitable for POSTNIG this repo to a remote storage that can hold more than one REPO. If not target log is specified the last use remote repo is used",
		Action: func(c *cli.Context) error {
			wd, err := os.Getwd()
			if err != nil {
				return err
			}

			var repoPath string

			if c.NArg() > 1 {
				return errors.New("post can have at most 1 argument. See --help.")
			} else if c.NArg() == 0 {
				repoPath = ""
			} else {
				repoPath = c.Args()[0]
			}

			pvr, err := NewPvr(c.App, wd)
			if err != nil {
				return err
			}

			err = pvr.Post(repoPath, c.String("envelope")) // XXX add surrounding flags etc
			if err != nil {
				return err
			}

			return nil
		},
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "envelope, e",
				Usage: "provide the json envelope to wrap around the pvr post.",
			},
		},
	}
}
