/*
 * Copyright 2017  Alexander Sack <asac129@gmail.com>
 */
package main

import (
	"os"

	"github.com/urfave/cli"
)

func CommandGet() cli.Command {
	return cli.Command{
		Name:        "get",
		Aliases:     []string{"g"},
		ArgsUsage:   "<repository> [target-repository]",
		Usage:       "get update target-repository from repository",
		Description: "default target-repository is the local .pvr one",
		Action: func(c *cli.Context) error {
			wd, err := os.Getwd()
			if err != nil {
				return err
			}

			pvr, err := NewPvr(wd)
			if err != nil {
				return err
			}

			err = pvr.GetRepo(c.Args()[0])
			if err != nil {
				return err
			}

			return nil
		},
	}
}
