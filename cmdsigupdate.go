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

	"github.com/urfave/cli"
	"gitlab.com/pantacor/pvr/libpvr"
)

func CommandSigUpdate() cli.Command {
	return cli.Command{
		Name:      "update",
		Aliases:   []string{"up"},
		ArgsUsage: "",
		Usage:     "update a pvs.json signature by using the encoded matchrule",
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

			pvs := c.String("pvs")

			if pvs == "" {
				return cli.NewExitError(errors.New("ERROR: no pvs provided; see --help"), 5)
			}

			ops := libpvr.PvsOptions{}

			keyPath := c.Parent().String("key")
			if keyPath == "" {
				return cli.NewExitError("needs a --key argument; see --help.", 126)
			}

			err = pvr.JwsSignPvs(keyPath, pvs, &ops)

			if err != nil {
				return cli.NewExitError(err, 126)
			}

			return nil
		},
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "pvs, p",
				Usage: "path to pvs.json file to update",
			},
		},
	}
}
