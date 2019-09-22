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
	"os"

	"github.com/urfave/cli"
	"gitlab.com/pantacor/pvr/libpvr"
)

func CommandLocalLogs() cli.Command {
	return cli.Command{
		Name:        "logs",
		Aliases:     []string{"log"},
		Usage:       "pvr local logs [http://][deviceip|hostname][:PORT]",
		ArgsUsage:   "[http://][deviceip|hostname][:PORT]",
		Description: "Get streaming logs of local devices you own",
		Action: func(c *cli.Context) error {
			wd, err := os.Getwd()
			if err != nil {
				return cli.NewExitError(err, 1)
			}
			session, err := libpvr.NewSession(c.App)
			if err != nil {
				return cli.NewExitError(err, 1)
			}
			pvr, err := libpvr.NewPvr(session, wd)
			if err != nil {
				return cli.NewExitError(err, 2)
			}
			deviceURL := ""
			if c.NArg() == 1 {
				deviceURL = c.Args().Get(0)
			}
			if deviceURL == "" {
				deviceURL = pvr.Pvrconfig.DefaultLocalDeviceURL
				if deviceURL == "" {
					return cli.NewExitError(errors.New("Default Local Device URL doesn't exist"), 2)
				}
			}
			err = pvr.GetLocalDeviceLogs(deviceURL)
			if err != nil {
				return cli.NewExitError("Error getting device logs: "+err.Error(), 4)
			}
			return nil
		},
	}
}
