//
// Copyright 2021  Pantacor Ltd.
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
		Aliases:   []string{"a"},
		ArgsUsage: "",
		Usage:     "embed a signature protecting the json document elements of the provided part using matchrule. By default we include all elements startings with _config/parts unless --noconfig is provided",
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
			raw := c.String("raw")

			if part == "" && raw == "" {
				return cli.NewExitError(errors.New("ERROR: no part nor raw name provided; see --help"), 5)
			}

			if part != "" && raw != "" {
				return cli.NewExitError(errors.New("ERROR: part and raw flag cannot be used at same time; see --help"), 5)
			}

			name := part
			if name == "" {
				name = raw
			}

			includes := c.StringSlice("include")

			if includes == nil {
				return cli.NewExitError(errors.New("ERROR: includes must not be nil; see --help"), 5)
			}

			// in 'part' mode we append path; in 'raw' we dont ....
			if name != raw {
				for i, v := range includes {
					includes[i] = path.Join(part, v)
				}
			}

			excludes := c.StringSlice("exclude")

			if excludes == nil {
				return cli.NewExitError(errors.New("ERROR: excludes must not be nil; see --help"), 5)
			}

			// in 'part' mode we append path; in 'raw' we dont ....
			if name != raw {
				for i, v := range excludes {
					excludes[i] = path.Join(part, v)
				}
			}

			// in 'part' mode we support configs; in 'raw' we dont ....
			if raw != name && !c.Bool("noconfig") {
				includes = append(includes, path.Join("_config", part, "**"))
			}

			match := libpvr.PvsMatch{
				Include: includes,
				Exclude: excludes,
			}

			ops := libpvr.PvsOptions{}

			keyPath := c.Parent().String("key")
			if keyPath == "" {
				keyPath = path.Join(pvr.Session.GetConfigDir(), "pvs", "key.default.pem")
				if _, err := os.Stat(keyPath); errors.Is(err, os.ErrNotExist) {
					err := libpvr.DownloadSigningCertWithConfirmation(
						c.App.Metadata["PVS_CERTS_URL"].(string),
						pvr.Session.GetConfigDir(),
					)
					if err != nil {
						return cli.NewExitError(err, 126)
					}
				}
			}

			ops.X5cPath = c.Parent().String("x5c")
			if ops.X5cPath == "" {
				ops.X5cPath = path.Join(pvr.Session.GetConfigDir(), "pvs", "x5c.default.pem")
				if _, err := os.Stat(ops.X5cPath); errors.Is(err, os.ErrNotExist) {
					err := libpvr.DownloadSigningCertWithConfirmation(
						c.App.Metadata["PVS_CERTS_URL"].(string),
						pvr.Session.GetConfigDir(),
					)
					if err != nil {
						return cli.NewExitError(err, 126)
					}
				}
			}

			err = pvr.JwsSign(name, keyPath, &match, &ops)

			if err != nil {
				return cli.NewExitError(err, 127)
			}

			return nil
		},
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "part, p",
				Usage: "select elements of part",
			},
			cli.StringFlag{
				Name:  "raw, r",
				Usage: "select elements using raw include/exclude; with this, noconfig does not apply",
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
			cli.BoolFlag{
				Name:  "noconfig, n",
				Usage: "exclude _config parts from signature",
			},
		},
	}
}
