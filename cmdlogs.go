//
// Copyright 2018  Pantacor Ltd.
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
	"strings"
	"time"

	"github.com/urfave/cli"
	"gitlab.com/pantacor/pvr/libpvr"
)

func CommandLogs() cli.Command {
	return cli.Command{
		Name:        "logs",
		Aliases:     []string{"log"},
		Usage:       "Show Owned Devices",
		Description: "Get a quick overview of devices you manage in Pantahub",
		Action: func(c *cli.Context) error {

			session, err := libpvr.NewSession(c.App)

			if err != nil {
				return cli.NewExitError(err, 4)
			}

			now := time.Now()
			logEntries, cursorID, err := session.DoLogs(c.App.Metadata["PVR_BASEURL"].(string), nil, &now, true)

			if err != nil {
				return cli.NewExitError("Error getting device list: "+err.Error(), 4)
			}

			for _, v := range logEntries {
				cutDeviceStart := strings.LastIndex(v.Device, "/")
				if cutDeviceStart < 0 {
					cutDeviceStart = 0
				} else {
					cutDeviceStart++
				}

				fmt.Printf("%s %s\t%s\n", v.TimeCreated.Format(time.RFC3339),
					v.Device[cutDeviceStart:cutDeviceStart+8]+":"+v.LogSource+":"+v.LogLevel,
					v.LogText)
			}

			for {
				logEntries, cursorID, err = session.DoLogsCursor(c.App.Metadata["PVR_BASEURL"].(string), cursorID)

				if err != nil {
					return cli.NewExitError("Error getting device list: "+err.Error(), 4)
				}

				//log.Printf("logs entries len %d\n", len(logEntries))
				for _, v := range logEntries {
					cutDeviceStart := strings.LastIndex(v.Device, "/")
					if cutDeviceStart < 0 {
						cutDeviceStart = 0
					} else {
						cutDeviceStart++
					}

					fmt.Printf("%s %s\t%s\n", v.TimeCreated.Format(time.RFC822Z),
						v.Device[cutDeviceStart:cutDeviceStart+8]+":"+v.LogSource+":"+v.LogLevel,
						v.LogText)
				}
			}

			return nil
		},
	}
}
