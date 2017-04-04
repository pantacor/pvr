/*
 * Copyright 2017  Alexander Sack <asac129@gmail.com>
 */
package main

import (
	"fmt"
	"os"

	"github.com/urfave/cli"
)

func CommandJson() cli.Command {
	return cli.Command{
		Name:        "json",
		Aliases:     []string{"j"},
		ArgsUsage:   "",
		Usage:       "Print JSON of working directory",
		Description: "Creates an aggregate json for the current working directory",
		Action: func(c *cli.Context) error {
			wd, err := os.Getwd()
			if err != nil {
				return cli.NewExitError(err, 1)
			}

			pvr, err := NewPvr(c.App, wd)
			if err != nil {
				return cli.NewExitError(err, 2)
			}

			result, err := pvr.GetWorkingJson()
			if err != nil {
				return cli.NewExitError(err, 3)
			}

			resultF, err := FormatJson(result)
			if err != nil {
				cli.NewExitError(err, 4)
			}

			fmt.Println(string(resultF))

			return nil
		},
	}
}
