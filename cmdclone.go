/*
 * Copyright 2017  Alexander Sack <asac129@gmail.com>
 */
package main

import (
	"errors"
	"net/url"
	"os"
	"path"

	"fmt"

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
				return err
			}

			if c.NArg() < 1 {
				return errors.New("clone needs need repository argument. See --help")
			}

			newUrl, err := url.Parse(c.Args().Get(0))
			if err != nil {
				return err
			}

			base := path.Base(newUrl.Path)
			base = path.Join(wd, base)
			if c.NArg() == 2 {
				base = c.Args().Get(1)
			}

			fmt.Println("base: " + base)

			err = os.Mkdir(base, 0755)
			if err != nil {
				return err
			}

			pvr, err := NewPvr(c.App, base)
			if err != nil {
				return err
			}

			err = pvr.Init()
			if err != nil {
				return err
			}

			err = pvr.GetRepo(newUrl.String())
			if err != nil {
				return err
			}

			err = pvr.Reset()

			return err
		},
	}
}
