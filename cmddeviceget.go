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

	"gitlab.com/pantacor/pvr/libpvr"

	"github.com/urfave/cli"
)

func CommandDeviceGet() cli.Command {
	cmd := cli.Command{
		Name:        "get",
		Aliases:     []string{"get"},
		ArgsUsage:   "<NICK|ID>",
		Usage:       "pvr device get <NICK|ID>",
		Description: "Get Device details",
		Action: func(c *cli.Context) error {
			session, err := libpvr.NewSession(c.App)
			if err != nil {
				return cli.NewExitError(err, 4)
			}
			baseURL := c.App.Metadata["PVR_BASEURL"].(string)
			deviceNick := ""
			if c.NArg() > 1 {
				return cli.NewExitError(errors.New("Device get command can have at most 1 argument. See --help"), 1)
			} else if c.NArg() == 1 {
				deviceNick = c.Args()[0]
			} else {
				return cli.NewExitError(errors.New("Device ID or Nick is required. See --help"), 2)
			}
			// Get Device Details
			deviceResponse, err := session.GetDevice(baseURL, deviceNick)
			if err != nil {
				return cli.NewExitError(err, 2)
			}
			err = libpvr.HandleNilRestResponse(deviceResponse, err)
			if err != nil {
				return cli.NewExitError(err, 2)
			}
			libpvr.LogPrettyJSON(deviceResponse.Body())
			return nil
		},
	}

	return cmd
}
