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
	"strings"

	"gitlab.com/pantacor/pvr/libpvr"

	"github.com/urfave/cli"
)

func CommandAppUpdate() cli.Command {
	cmd := cli.Command{
		Name:        "update",
		Aliases:     []string{"au"},
		ArgsUsage:   "[appname]",
		Usage:       "update an existing application.",
		Description: "update application files",
		Action: func(c *cli.Context) error {
			wd, err := os.Getwd()
			if err != nil {
				return cli.NewExitError(err, 1)
			}

			if c.NArg() < 1 {
				return cli.NewExitError("app-update needs application argument. See --help", 2)
			}

			session, err := libpvr.NewSession(c.App)

			if err != nil {
				return cli.NewExitError(err, 4)
			}

			pvr, err := libpvr.NewPvr(session, wd)
			if err != nil {
				return cli.NewExitError(err, 2)
			}

			// fix up trailing/leading / from appnames
			appname := strings.Trim(c.Args().Get(0), "/")
			username := c.String("username")
			password := c.String("password")

			err = pvr.UpdateApplication(appname, username, password)
			if err != nil {
				return cli.NewExitError(err, 3)
			}

			fmt.Println("Application updated")

			return nil
		},
	}

	cmd.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   "username, u",
			Usage:  "Use `PVR_REGISTRY_USERNAME` for authorization with docker registrar",
			EnvVar: "PVR_REGISTRY_USERNAME",
		},
		cli.StringFlag{
			Name:   "password, p",
			Usage:  "Use `PVR_REGISTRY_PASSWORD` for authorization with docker registrar",
			EnvVar: "PVR_REGISTRY_PASSWORD",
		},
	}

	return cmd
}
