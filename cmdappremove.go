//
// Copyright 2019-2023  Pantacor Ltd.
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
	"os"
	"strings"

	"github.com/urfave/cli"
	"gitlab.com/pantacor/pvr/libpvr"
)

// CommandAppRemove : pvr app rm command to remove an app
func CommandAppRemove() cli.Command {
	cmd := cli.Command{
		Name:        "rm",
		Aliases:     []string{"r"},
		ArgsUsage:   "[appname]",
		Usage:       "pvr app rm <appname> : remove app from pvr checkout",
		Description: "Remove app from pvr checkout",
		Action: func(c *cli.Context) error {
			rootPath, err := os.Getwd()
			if err != nil {
				return cli.NewExitError(err, 1)
			}
			if c.NArg() < 1 {
				return cli.NewExitError("'pvr app rm' needs appname argument(eg:app rm <appname> ). See --help", 2)
			}
			session, err := libpvr.NewSession(c.App)
			if err != nil {
				return cli.NewExitError(err, 4)
			}
			pvr, err := libpvr.NewPvr(session, rootPath)
			if err != nil {
				return cli.NewExitError(err, 2)
			}
			appname := c.Args().Get(0)

			// fix up trailing/leading / from appnames
			appname = strings.Trim(appname, "/")

			err = pvr.RemoveApplication(appname)
			if err != nil {
				return cli.NewExitError(err, 3)
			}
			return nil
		},
	}

	return cmd
}
