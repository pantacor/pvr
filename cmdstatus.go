/*
 * Copyright 2017  Alexander Sack <asac129@gmail.com>
 */
package main

import (
	"fmt"
	"os"

	"github.com/urfave/cli"
)

func CommandStatus() cli.Command {
	return cli.Command{
		Name:      "status",
		Aliases:   []string{"s"},
		ArgsUsage: "",
		Usage:     "Display status of working dir compared to pristine state.",
		Action: func(c *cli.Context) error {
			wd, err := os.Getwd()
			if err != nil {
				return cli.NewExitError(err, 1)
			}

			pvr, err := NewPvr(c.App, wd)
			if err != nil {
				return cli.NewExitError(err, 2)
			}

			status, err := pvr.Status()
			if err != nil {
				return cli.NewExitError(err, 3)
			}

			fmt.Println(status)

			return nil
		},
	}
}
