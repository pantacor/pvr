/*
 * Copyright 2017  Alexander Sack <asac129@gmail.com>
 */
package main

import (
	"errors"
	"os"

	"github.com/urfave/cli"
)

func CommandPut() cli.Command {
	return cli.Command{
		Name:        "put",
		Aliases:     []string{"p"},
		ArgsUsage:   "[target-repo]",
		Usage:       "put local repository to a target respository.",
		Description: "Can put to local and REST repos",
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

			err = pvr.Put(c.Args()[0])
			if err != nil {
				return err
			}

			return nil
		},
	}
}
