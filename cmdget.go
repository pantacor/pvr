/*
 * Copyright 2017  Alexander Sack <asac129@gmail.com>
 */
package main

import (
	"errors"
	"os"

	"github.com/urfave/cli"
)

func CommandGet() cli.Command {
	return cli.Command{
		Name:        "get",
		Aliases:     []string{"g"},
		ArgsUsage:   "[repository [target-repository]]",
		Usage:       "get update target-repository from repository",
		Description: "default target-repository is the local .pvr one. If not <repository> is provided the last one is used.",
		Action: func(c *cli.Context) error {
			wd, err := os.Getwd()
			if err != nil {
				return err
			}

			pvr, err := NewPvr(c.App, wd)
			if err != nil {
				return err
			}

			var repoPath string

			if c.NArg() > 1 {
				return errors.New("Get can have at most 1 argument. See --help.")
			} else if c.NArg() == 0 {
				repoPath = ""
			} else {
				repoPath = c.Args()[0]
			}

			err = pvr.GetRepo(repoPath)
			if err != nil {
				return err
			}

			return nil
		},
	}
}
