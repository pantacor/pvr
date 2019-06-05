//
// Copyright 2019  Pantacor Ltd.
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
	"fmt"
	"os"

	"github.com/urfave/cli"
	"gitlab.com/pantacor/pvr/libpvr"
)

// CommandSelfUpdate update the PVR CLI using the latest version of the docker container
func CommandSelfUpdate() cli.Command {
	return cli.Command{
		Name:      "self-upgrade",
		ArgsUsage: "",
		Usage:     "Update pvr command to the latest version",
		Action: func(c *cli.Context) error {
			wd, err := os.Getwd()
			if err != nil {
				return cli.NewExitError(err.Error(), 1)
			}

			session, err := libpvr.NewSession(c.App)
			if err != nil {
				return cli.NewExitError(err.Error(), 4)
			}

			pvr, err := libpvr.NewPvr(session, wd)
			if err != nil {
				return cli.NewExitError(err.Error(), 2)
			}

			username := c.String("username")
			password := c.String("password")

			err = pvr.UpdatePvr(&username, &password, false)
			if err != nil {
				fmt.Println(err.Error())
				cli.NewExitError(err.Error(), 2)
			}
			return nil
		},
	}
}
