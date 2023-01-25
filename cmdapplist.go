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
	"os"

	"github.com/urfave/cli"
	"gitlab.com/pantacor/pvr/libpvr"
)

// CommandAppList : pvr app ls command to list all apps
func CommandAppList() cli.Command {
	cmd := cli.Command{
		Name:        "ls",
		Aliases:     []string{"l"},
		ArgsUsage:   "",
		Usage:       "pvr app ls :list applications in pvr checkout",
		Description: "List applications in pvr checkout",
		Action: func(c *cli.Context) error {
			rootPath, err := os.Getwd()
			if err != nil {
				return cli.NewExitError(err, 1)
			}
			session, err := libpvr.NewSession(c.App)
			if err != nil {
				return cli.NewExitError(err, 4)
			}
			pvr, err := libpvr.NewPvr(session, rootPath)
			if err != nil {
				return cli.NewExitError(err, 2)
			}
			err = pvr.ListApplications()
			if err != nil {
				return cli.NewExitError(err, 3)
			}
			return nil
		},
	}

	return cmd
}
