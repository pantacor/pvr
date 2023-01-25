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
	"errors"
	"log"
	"strings"

	"gitlab.com/pantacor/pvr/libpvr"

	"github.com/urfave/cli"
)

func CommandDeviceGet() cli.Command {
	cmd := cli.Command{
		Name:        "get",
		Aliases:     []string{"get"},
		ArgsUsage:   "<NICK|ID> | <USER_NICK>/<NICK|ID> ",
		Usage:       "pvr device get <NICK|ID> | <USER_NICK>/<NICK|ID> ",
		Description: "Get Device details",
		BashComplete: func(c *cli.Context) {
			if c.GlobalString("baseurl") != "" {
				c.App.Metadata["PVR_BASEURL"] = c.GlobalString("baseurl")
			}
			session, err := libpvr.NewSession(c.App)
			if err != nil {
				log.Fatal(err.Error())
				return
			}
			if c.NArg() == 0 {
				return
			}
			searchTerm := c.Args()[c.NArg()-1]
			baseURL := c.App.Metadata["PVR_BASEURL"].(string)

			splits := strings.Split(searchTerm, "/")

			if len(splits) == 1 {
				session.SuggestDeviceNicks("", searchTerm, baseURL)
			} else {
				userNick := splits[1]
				session.SuggestDeviceNicks(userNick, searchTerm, baseURL)
			}
		},
		Action: func(c *cli.Context) error {
			session, err := libpvr.NewSession(c.App)
			if err != nil {
				return cli.NewExitError(err, 4)
			}
			baseURL := c.App.Metadata["PVR_BASEURL"].(string)
			deviceNick := ""
			ownerNick := ""
			if c.NArg() > 1 {
				return cli.NewExitError(errors.New("Device get command can have at most 1 argument. See --help"), 1)
			} else if c.NArg() == 1 {
				splits := strings.Split(c.Args()[0], "/")
				deviceNick = splits[0]

				if len(splits) > 1 {
					ownerNick = splits[0]
					deviceNick = splits[1]
				}

			} else {
				return cli.NewExitError(errors.New("Device ID or Nick is required. See --help"), 2)
			}
			// Get Device Details
			deviceResponse, err := session.GetDevice(baseURL, deviceNick, ownerNick)
			if err != nil {
				return cli.NewExitError(err, 2)
			}
			libpvr.LogPrettyJSON(deviceResponse.Body())
			return nil
		},
	}

	return cmd
}
