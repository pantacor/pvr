/*
 * Copyright 2017  Alexander Sack <asac129@gmail.com>
 */
package main

import (
	"os"

	"github.com/urfave/cli"
)

func CommandImport() cli.Command {
	return cli.Command{
		Name:        "import",
		Aliases:     []string{"i"},
		ArgsUsage:   "<repo-tarball>",
		Usage:       "import repo tarball (like the one produced by 'pvr export') into pvr in current working dir",
		Description: "can import files with.gz or .tgz extension as well as plain .tar. Will not do pvr checkout, so working directory stays untouched.",
		Action: func(c *cli.Context) error {
			wd, err := os.Getwd()
			if err != nil {
				return cli.NewExitError(err, 1)
			}

			pvr, err := NewPvr(c.App, wd)
			if err != nil {
				return cli.NewExitError(err, 2)
			}

			err = pvr.Import(c.Args()[0])
			if err != nil {
				return cli.NewExitError(err, 3)
			}

			return nil
		},
	}
}
