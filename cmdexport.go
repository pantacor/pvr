/*
 * Copyright 2017  Alexander Sack <asac129@gmail.com>
 */
package main

import (
	"os"

	"github.com/urfave/cli"
)

func CommandExport() cli.Command {
	return cli.Command{
		Name:        "export",
		Aliases:     []string{"g"},
		ArgsUsage:   "<export-file>",
		Usage:       "export repo into single file (tarball)",
		Description: "if export file ends with .gz or .tgz it will create a zipped tarball. Otherwise plain",
		Action: func(c *cli.Context) error {
			wd, err := os.Getwd()
			if err != nil {
				return cli.NewExitError(err, 1)
			}

			pvr, err := NewPvr(c.App, wd)
			if err != nil {
				return cli.NewExitError(err, 2)
			}

			err = pvr.Export(c.Args()[0])
			if err != nil {
				return cli.NewExitError(err, 3)
			}

			return nil
		},
	}
}
