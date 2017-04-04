/*
 * Copyright 2017  Alexander Sack <asac129@gmail.com>
 */
package main

import (
	"os"

	"github.com/urfave/cli"
)

func CommandPut() cli.Command {
	return cli.Command{
		Name:        "put",
		Aliases:     []string{"p"},
		ArgsUsage:   "[target-repo]",
		Usage:       "put local repository to a target respository.",
		Description: "Can put to local and REST repos. If no repository is provided the previously used one is used.",
		Action: func(c *cli.Context) error {
			wd, err := os.Getwd()
			if err != nil {
				return cli.NewExitError(err, 1)
			}

			var repoPath string

			if c.NArg() > 1 {
				return cli.NewExitError("Push can have at most 1 argument. See --help.", 2)
			} else if c.NArg() == 0 {
				repoPath = ""
			} else {
				repoPath = c.Args()[0]
			}

			pvr, err := NewPvr(c.App, wd)
			if err != nil {
				return cli.NewExitError(err, 3)
			}

			err = pvr.Put(repoPath, c.Bool("force"))
			if err != nil {
				return cli.NewExitError(err, 4)
			}
			return nil
		},
		Flags: []cli.Flag{
			cli.BoolFlag{
				Name:  "force, f",
				Usage: "force reupload of existing objects",
			},
		},
	}
}
