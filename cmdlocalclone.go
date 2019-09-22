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
	"errors"
	"fmt"

	"github.com/urfave/cli"
	"gitlab.com/pantacor/pvr/libpvr"
)

// CommandLocalClone : pvr local clone command
func CommandLocalClone() cli.Command {
	cmd := cli.Command{
		Name:        "clone",
		Aliases:     []string{"cl"},
		ArgsUsage:   "[http://]{deviceip|hostname}[:PORT][/revision] [DIR]",
		Usage:       "pvr local clone [http://]{deviceip|hostname}[:PORT][/revision] [DIR]",
		Description: "Clone a local device",
		Action: func(c *cli.Context) error {
			deviceURL := ""
			deviceDir := ""
			if c.NArg() < 1 {
				return cli.NewExitError(errors.New("Device ip or hostname is required for pvr local clone [http://]{deviceip|hostname}[:PORT][/revision]. See --help"), 1)
			} else if c.NArg() == 2 {
				deviceURL = c.Args().Get(0)
				deviceDir = c.Args().Get(1)
			} else if c.NArg() == 1 {
				deviceURL = c.Args().Get(0)
			}
			session, err := libpvr.NewSession(c.App)
			if err != nil {
				return cli.NewExitError(err, 2)
			}
			err = session.CloneLocalDevice(deviceURL, deviceDir)
			if err != nil {
				return cli.NewExitError(err, 3)
			}
			fmt.Println("\n\nCloned Successfully from local device:" + deviceURL + "\n")
			return nil

		},
	}

	return cmd
}
