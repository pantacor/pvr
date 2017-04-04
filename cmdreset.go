/*
 * Copyright 2017  Alexander Sack <asac129@gmail.com>
 */
package main

import (
	"os"

	"github.com/urfave/cli"
)

func CommandReset() cli.Command {
	return cli.Command{
		Name:        "reset",
		Aliases:     []string{"r", "checkout", "co"},
		ArgsUsage:   "",
		Usage:       "reset working directory to match the repo state",
		Description: "reset/checkout also forgets about added files; pvr status and diff will yield empty",
		Action: func(c *cli.Context) error {
			wd, err := os.Getwd()
			if err != nil {
				return cli.NewExitError(err, 1)
			}

			pvr, err := NewPvr(c.App, wd)
			if err != nil {
				return cli.NewExitError(err, 2)
			}

			err = pvr.Reset()
			if err != nil {
				return cli.NewExitError(err, 3)
			}

			return nil
		},
	}
}
