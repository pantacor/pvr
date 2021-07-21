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
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"

	"github.com/urfave/cli"
	"gitlab.com/pantacor/pvr/libpvr"
)

func CommandSigLs() cli.Command {
	return cli.Command{
		Name:      "ls",
		ArgsUsage: "",
		Usage:     "verify and list content protected by pvs.json files for a part",
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

			pubkey := c.Parent().String("pubkey")

			if pubkey == "" {
				return cli.NewExitError(errors.New("ERROR: no signing key provided; see --help"), 10)
			}

			keyFs, err := os.Stat(pubkey)

			if err != nil {
				return cli.NewExitError(("ERROR: errors accessing signing key; see --help: " + err.Error()), 11)
			}

			if keyFs.IsDir() {
				return cli.NewExitError(("ERROR: signing key is not a file; see --help: " + err.Error()), 12)
			}

			part := c.String("part")

			if part == "" {
				return cli.NewExitError(errors.New("ERROR: no part provided; see --help"), 5)
			}

			partPvs := path.Join(part, "pvs.json")

			partPvsFs, err := os.Stat(partPvs)

			if err != nil {
				return cli.NewExitError(err, 13)
			}

			if partPvsFs.IsDir() {
				return cli.NewExitError(errors.New("ERROR: pvs.json is a directory; see --help"), 13)
			}

			verifySummary, err := pvr.JwsVerify(pubkey, part)

			if err != nil {
				return cli.NewExitError(err, 13)
			}

			jsonBuf, err := json.MarshalIndent(verifySummary, "", "    ")

			if err != nil {
				return cli.NewExitError(err, 13)
			}

			fmt.Println(string(jsonBuf))

			return nil
		},
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:   "part, p",
				Usage:  "select elements of part",
				EnvVar: "PVR_SIG_ADD_PART",
			},
		},
	}
}
