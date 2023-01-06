//
// Copyright 2018-2023  Pantacor Ltd.
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
	"log"
	"strings"

	"github.com/urfave/cli"
	"gitlab.com/pantacor/pvr/libpvr"
)

func CommandLogsDeprecated() cli.Command {
	return cli.Command{
		Name:        "logs",
		Aliases:     []string{"log"},
		ArgsUsage:   "<deviceid|devicenick>[/source][@Level][#Platform]",
		Usage:       "pvr logs <deviceid|devicenick>[/source][@Level][#Platform]",
		Description: "Get streaming logs of devices you own from pantahub",
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
			filter := c.Args().Get(c.NArg() - 1)
			splits := strings.Split(filter, "/")
			deviceSearchTerm := splits[0]

			baseURL := c.App.Metadata["PVR_BASEURL"].(string)
			session.SuggestDeviceNicks("", deviceSearchTerm, baseURL)
		},
		Before: func(c *cli.Context) error {
			fmt.Print("\nDEPRECATED: the pvr logs command is deprecated and will go away in some future release. It can now be found as a device subcommand:pvr device logs\n")
			return nil
		},
		Action: CommandLogs().Action,
		Flags:  CommandLogs().Flags,
	}
}
