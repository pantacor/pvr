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

	duration "github.com/ChannelMeter/iso8601duration"
	"github.com/urfave/cli"
	"gitlab.com/pantacor/pvr/libpvr"
)

func CommandLogs() cli.Command {
	return cli.Command{
		Name:        "logs",
		Aliases:     []string{"log"},
		ArgsUsage:   "<deviceid|devicenick>[/source][@Level]",
		Usage:       "pvr device logs <deviceid|devicenick>[/source][@Level]",
		Description: "Get streaming logs of devices you own from pantahub",
		Action: func(c *cli.Context) error {

			session, err := libpvr.NewSession(c.App)

			if err != nil {
				return cli.NewExitError(err, 4)
			}

			from := time.Now().Add(time.Duration(-1 * time.Minute))
			if c.String("from") != "" {
				from, err = libpvr.ParseRFC3339(c.String("from"))
				if err != nil {
					parsedDuration, err := duration.FromString(c.String("from"))
					if err != nil {
						return cli.NewExitError(err, 5)
					}
					from = time.Now().Local().Add(-parsedDuration.ToDuration())
				}
			}
			var to time.Time
			if c.String("to") != "" {
				to, err = libpvr.ParseRFC3339(c.String("to"))
				if err != nil {
					parsedDuration, err := duration.FromString(c.String("to"))
					if err != nil {
						return cli.NewExitError(err, 5)
					}
					to = from.Add(parsedDuration.ToDuration())
				}
			}

			splits := []string{}
			devices := []string{}
			source := ""
			level := ""
			filter := c.Args().Get(0)

			if filter != "" {
				splits = strings.Split(filter, "/")
			}
			if len(splits) > 0 {
				devices = []string{splits[0]}
			}

			if len(splits) > 1 {
				splits2 := strings.Split(splits[1], "@")
				source = splits2[0]
				if len(splits2) > 1 {
					level = splits2[1]
				}
			}
			logFilter := libpvr.LogFilter{
				Devices: strings.Join(devices, ","),
				Levels:  level,
				Sources: source,
			}

			for {
				logEntries, cursorID, err := session.DoLogs(c.App.Metadata["PVR_BASEURL"].(string), nil, &from, &to, true, logFilter)

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

					// advance "from" cursor
					from = v.TimeCreated
				}

				for {
					logEntries, cursorID, err = session.DoLogsCursor(c.App.Metadata["PVR_BASEURL"].(string), cursorID)

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

						fmt.Printf("%s %s\t%s\n", v.TimeCreated.Format(time.RFC822Z),
							v.Device[cutDeviceStart:cutDeviceStart+8]+":"+v.LogSource+":"+v.LogLevel,
							v.LogText)

						// advance "from" cursor
						from = v.TimeCreated

					}
					// if we reach end of cursor we have exhausted it and will sleep
					// before trying to get new logs starting from last timestamp
					if len(logEntries) == 0 {
						time.Sleep(time.Duration(1 * time.Second))
						break
					}
				}
			}

			return nil
		},
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:   "from,f",
				Usage:  "Datetime in RFC3339 format, e.g.: --from=2006-01-02T15:04:05+06:00",
				EnvVar: "PVR_LOGS_FROM_DATE",
			},
			cli.StringFlag{
				Name:   "to,t",
				Usage:  "Datetime in RFC3339 format, e.g.:--to=2006-01-02T15:04:05+06:00",
				EnvVar: "PVR_LOGS_TO_DATE",
			},
		},
	}
}
