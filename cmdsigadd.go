//
// Copyright 2017-2021  Pantacor Ltd.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
//   Unless required by applicable law or agreed to in writing, software
//   distributed under the License is distributed on an "AS IS" BASIS,
//   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//   See the License for the specific language governing permissions and
//   limitations under the License.
//
package main

import (
	"errors"
	"os"
	"path"

	"github.com/urfave/cli"
	"gitlab.com/pantacor/pvr/libpvr"
)

func CommandSigAdd() cli.Command {
	return cli.Command{
		Name:      "add",
		Aliases:   []string{"ci"},
		ArgsUsage: "",
		Usage:     "embed a signature protecting the json document elements by matchrule",
		Action: func(c *cli.Context) error {
			wd, err := os.Getwd()
			if err != nil {
				return err
			}

			session, err := libpvr.NewSession(c.App)

			if err != nil {
				return cli.NewExitError(err, 4)
			}

			pvr, err := libpvr.NewPvr(session, wd)
			if err != nil {
				return cli.NewExitError(err, 2)
			}

			commitmsg := c.String("message")
			if commitmsg == "" {
				commitmsg = "** No commit message **"
			}
			part := c.String("part")

			if part == "" {
				return cli.NewExitError(errors.New("ERROR: no part provided; see --help"), 5)
			}

			includes := c.StringSlice("include")

			if includes == nil {
				return cli.NewExitError(errors.New("ERROR: includes must not be nil; see --help"), 5)
			}

			excludes := c.StringSlice("exclude")

			if excludes == nil {
				return cli.NewExitError(errors.New("ERROR: excludes must not be nil; see --help"), 5)
			}

			match := libpvr.PvsMatch{
				Part:    part,
				Include: includes,
				Exclude: excludes,
			}

			ops := libpvr.PvsOptions{}

			configDir := c.App.Metadata["PVR_CONFIG_DIR"].(string)
			keyPath := path.Join(configDir, "secrets", "priv.pem")

			err = pvr.JwsSign(keyPath, &match, &ops)

			if err != nil {
				return err
			}

			return nil
		},
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "part, p",
				Usage: "select elements of part",
			},
			cli.StringSliceFlag{
				Name:  "include, i",
				Usage: "include files by glob pattern",
				Value: &cli.StringSlice{"**"},
			},
			cli.StringSliceFlag{
				Name:  "exclude, e",
				Usage: "exclude files by glob patterns",
				Value: &cli.StringSlice{"src.json"},
			},
		},
	}
}
