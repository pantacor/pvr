//
// Copyright 2017-2023  Pantacor Ltd.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

package main

import (
	"errors"
	"os"
	"strings"

	"github.com/urfave/cli"
	"gitlab.com/pantacor/pvr/libpvr"
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

			session, err := libpvr.NewSession(c.App)

			if err != nil {
				return cli.NewExitError(err, 4)
			}

			pvr, err := libpvr.NewPvr(session, wd)
			if err != nil {
				return cli.NewExitError(err, 2)
			}

			if c.NArg() > 1 {
				return errors.New("export can have at most 1 argument. See --help")
			}
			if c.NArg() < 1 {
				return errors.New("export-file name is required. See --help")
			}
			var parts []string
			if c.String("parts") != "" {
				parts = strings.Split(c.String("parts"), ",")
			} else {
				parts = []string{}
			}
			err = pvr.Export(parts, c.Args()[0])
			if err != nil {
				return cli.NewExitError(err, 3)
			}

			return nil
		},
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "parts, p",
				Usage: "comma separate list of parts to export; if empty we export all",
			},
		},
	}
}
