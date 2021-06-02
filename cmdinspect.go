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
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/urfave/cli"
	"gitlab.com/pantacor/pvr/libpvr"
)

func CommandInspect() cli.Command {
	return cli.Command{
		Name:        "inspect",
		Aliases:     []string{"i"},
		ArgsUsage:   "[<repository>[#<part>] [<target-repository>]] | [<USER_NICK>/<DEVICE_NICK>[#<part>]]",
		Usage:       "inspect repository state",
		Description: "default target-repository is the local .pvr one. If not <repository> is provided the last one is used. <part> can be one of 'bsp' or $appname.",
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

			var repoUri string

			if c.NArg() > 1 {
				return errors.New("Get can have at most 1 argument. See --help.")
			} else if c.NArg() == 0 {
				repoUri = pvr.Pvrdir
			} else {
				repoUri = c.Args()[0]
			}

			jsonMap, err := pvr.GetStateJson(repoUri)
			if err != nil {
				return cli.NewExitError(err, 1)
			}

			jsonData, err := json.MarshalIndent(jsonMap, "", "    ")

			if err != nil {
				return cli.NewExitError(err, 1)
			}

			jsonStr := string(jsonData)

			fmt.Println(jsonStr)

			return nil
		},
	}
}
