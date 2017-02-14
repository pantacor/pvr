/*
 * Copyright 2017  Alexander Sack <asac129@gmail.com>
 */
package main

import (
	"fmt"
	"os"

	"github.com/urfave/cli"
)

func CommandInit() cli.Command {
	return cli.Command{
		Name:        "init",
		Aliases:     []string{"i"},
		ArgsUsage:   "",
		Usage:       "pvr'ize the working directory",
		Description: "Creates the .pvr according to default spec. Creates systemc if not exists.",
		Action: func(c *cli.Context) error {
			wd, err := os.Getwd()
			if err != nil {
				return err
			}
			fmt.Println("NEW DIR INIT: " + wd)

			pvr, err := NewPvr(c.App, wd)

			if err != nil {
				return err
			}
			// empty template as starting point; XXX; add Flag to pass custom json
			err = pvr.Init()

			return err
		},
	}
}
