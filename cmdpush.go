/*
 * Copyright 2017  Alexander Sack <asac129@gmail.com>
 */
package main

import (
	"errors"
	"os"

	"github.com/urfave/cli"
)

func CommandPush() cli.Command {
	return cli.Command{
		Name:        "push",
		Aliases:     []string{"p"},
		ArgsUsage:   "[remote-location]",
		Usage:       "push to a remote location.",
		Description: "Pointed to local pvr repository or pvr REST endpoint",
		Action: func(c *cli.Context) error {
			wd, err := os.Getwd()
			if err != nil {
				return err
			}

			if c.NArg() != 1 {
				return errors.New("Push requires exactly 1 argument. See --help.")
			}

			pvr, err := NewPvr(wd)
			if err != nil {
				return err
			}

			err = pvr.Push(c.Args()[0])
			if err != nil {
				return err
			}

			return nil
		},
	}
}
