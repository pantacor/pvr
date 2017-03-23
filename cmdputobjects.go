/*
 * Copyright 2017  Alexander Sack <asac129@gmail.com>
 */
package main

import (
	"errors"
	"os"

	"github.com/urfave/cli"
)

func CommandPutObjects() cli.Command {
	return cli.Command{
		Name:      "putobjects",
		Aliases:   []string{"po"},
		ArgsUsage: "[objects-endpoint]",
		Usage:     "put objects from local repository to objects-endpoint",
		Action: func(c *cli.Context) error {
			wd, err := os.Getwd()
			if err != nil {
				return err
			}

			if c.NArg() != 1 {
				return errors.New("Push requires exactly 1 argument. See --help.")
			}

			pvr, err := NewPvr(c.App, wd)
			if err != nil {
				return err
			}

			err = pvr.PutObjects(c.Args()[0], c.Bool("force"))
			if err != nil {
				return err
			}

			return nil
		},
		Flags: []cli.Flag{
			cli.BoolFlag{
				Name:  "force, f",
				Usage: "force reupload of existing objects",
			},
		},
	}
}
