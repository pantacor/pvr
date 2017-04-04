/*
 * Copyright 2017  Alexander Sack <asac129@gmail.com>
 */
package main

import (
	"net/url"
	"os"
	"path"

	"github.com/urfave/cli"
)

func CommandClone() cli.Command {
	return cli.Command{
		Name:        "clone",
		Aliases:     []string{"c"},
		ArgsUsage:   "<repository> [directory]",
		Usage:       "clone repository to a new target directory",
		Description: "this combines operations: new, get, checkout",
		Action: func(c *cli.Context) error {
			wd, err := os.Getwd()
			if err != nil {
				return cli.NewExitError(err, 1)
			}

			if c.NArg() < 1 {
				return cli.NewExitError("clone needs need repository argument. See --help", 2)
			}

			newUrl, err := url.Parse(c.Args().Get(0))
			if err != nil {
				return cli.NewExitError(err, 3)
			}

			base := path.Base(newUrl.Path)
			base = path.Join(wd, base)
			if c.NArg() == 2 {
				base = c.Args().Get(1)
			}

			err = os.Mkdir(base, 0755)
			if err != nil {
				return cli.NewExitError(err, 4)
			}

			pvr, err := NewPvr(c.App, base)
			if err != nil {
				return cli.NewExitError(err, 5)
			}

			err = pvr.Init()
			if err != nil {
				return cli.NewExitError(err, 6)
			}

			err = pvr.GetRepo(newUrl.String())
			if err != nil {
				return cli.NewExitError(err, 7)
			}

			err = pvr.Reset()

			if err != nil {
				return cli.NewExitError(err, 8)
			}

			return nil
		},
	}
}
