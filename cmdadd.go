/*
 * Copyright 2017  Alexander Sack <asac129@gmail.com>
 */
package main

import (
	"os"

	"github.com/urfave/cli"
)

func CommandAdd() cli.Command {
	return cli.Command{
		Name:        "add",
		Aliases:     []string{"a"},
		ArgsUsage:   "[files]",
		Usage:       "add all or selected files to set of pvr tracked files.",
		Description: "add files to .pvr/tracking.json until the next commit",
		Action: func(c *cli.Context) error {
			wd, err := os.Getwd()
			if err != nil {
				return err
			}

			pvr, err := NewPvr(wd)
			if err != nil {
				return err
			}

			err = pvr.AddFile(os.Args[2:])
			if err != nil {
				return err
			}

			return nil
		},
	}
}
