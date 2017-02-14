/*
 * Copyright 2017  Alexander Sack <asac129@gmail.com>
 */
package main

import (
	"fmt"
	"os"

	"github.com/urfave/cli"
)

func CommandDiff() cli.Command {
	return cli.Command{
		Name:        "diff",
		Aliases:     []string{"d"},
		ArgsUsage:   "[files]",
		Usage:       "Display diff of pristine to working state.",
		Description: "show json diff of working dir to last committed state. filter diff by files if requested.",
		Action: func(c *cli.Context) error {
			wd, err := os.Getwd()
			if err != nil {
				return err
			}

			pvr, err := NewPvr(c.App, wd)
			if err != nil {
				return err
			}

			jsonDiff, err := pvr.Diff()
			if err != nil {
				return err
			}

			jsonDiffF, err := FormatJson(*jsonDiff)
			fmt.Println(string(jsonDiffF))

			return nil
		},
	}
}
