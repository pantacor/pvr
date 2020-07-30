//
// Copyright 2020  Pantacor Ltd.
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
	"io/ioutil"
	"os"

	"fmt"

	"github.com/urfave/cli"
	"gitlab.com/pantacor/pvr/libpvr"
)

func CommandFastCopy() cli.Command {
	return cli.Command{
		Name:        "fastcopy",
		Aliases:     []string{"fcp"},
		ArgsUsage:   "<SRCPVR>[#<source-fragment>] <DESTPVR>[#<dest-fragment>]",
		Usage:       "Fast copy from source pvr URL to DESTPVR url",
		Description: "<source-fragement> can be used to select a specific app/folder in state to copy from. <dest-fragement> can be used to use a fragment name other than <source-fragement> as destination name.",
		Action: func(c *cli.Context) error {
			var src string
			var dest string

			if c.NArg() != 2 {
				return errors.New("fastcopy must have exacty 2 arguments. See --help")
			} else {
				src = c.Args()[0]
				dest = c.Args()[1]
			}

			session, err := libpvr.NewSession(c.App)

			if err != nil {
				return cli.NewExitError(err, 4)
			}

			wd, err := ioutil.TempDir(os.TempDir(), "pvr-fastcopy-")

			if err != nil {
				return cli.NewExitError(err, 3)
			}

			pvr, err := libpvr.NewPvr(session, wd)
			if err != nil {
				return cli.NewExitError(err, 2)
			}

			err = pvr.RemoteCopy(src, dest, false, c.String("envelope"), c.String("commit-msg"),
				c.Int("rev"), c.Bool("force"))

			if err != nil {
				fmt.Println("ERROR: " + err.Error())
				return cli.NewExitError(err, 3)
			}

			return nil
		},
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "envelope, e",
				Usage: "provide the json envelope to wrap around the pvr post. use {} when not provided",
			},
			cli.StringFlag{
				Name: "commit-msg, m",
				Usage: "add 'commit-msg' field 	to envelope",
			},
			cli.StringFlag{
				Name:  "rev",
				Usage: "add 'rev' fieldcall to envelope",
				Value: "-1",
			},
			cli.BoolFlag{
				Name:  "force, f",
				Usage: "force reupload of existing objects",
			},
		},
	}
}
