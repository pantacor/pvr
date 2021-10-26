//
// Copyright 2018-2020  Pantacor Ltd.
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
	"log"
	"strings"
	"time"

	duration "github.com/ChannelMeter/iso8601duration"
	"github.com/urfave/cli"
	"gitlab.com/pantacor/pvr/libpvr"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func CommandLogs() cli.Command {
	return cli.Command{
		Name:        "logs",
		Aliases:     []string{"log"},
		ArgsUsage:   "<deviceid|devicenick>[/source][@Level][#Platform]",
		Usage:       "pvr device logs <deviceid|devicenick>[/source][@Level][#Platform]",
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
		Action: func(c *cli.Context) error {

			session, err := libpvr.NewSession(c.App)

			if err != nil {
				return cli.NewExitError(err, 4)
			}

			rev := c.Int("rev")

			var from *time.Time
			if rev < 0 {
				_t := time.Now().Add(time.Duration(-1 * time.Minute))
				from = &_t
			}
			if c.String("from") != "" {
				_t, err := libpvr.ParseRFC3339(c.String("from"))
				if err != nil {
					parsedDuration, err := duration.FromString(c.String("from"))
					if err != nil {
						return cli.NewExitError(err, 5)
					}
					_t = time.Now().Local().Add(-parsedDuration.ToDuration())
				}
				from = &_t
			}
			var to *time.Time
			if c.String("to") != "" {
				_t, err := libpvr.ParseRFC3339(c.String("to"))
				if err != nil {
					parsedDuration, err := duration.FromString(c.String("to"))
					if err != nil {
						return cli.NewExitError(err, 5)
					}
					_t = from.Add(parsedDuration.ToDuration())
				}
				to = &_t
			}

			splits := []string{}
			devices := []string{}
			source := ""
			level := ""
			platforms := ""
			filter := c.Args().Get(0)

			if filter != "" {
				splits = strings.Split(filter, "/")
			}
			if len(splits) > 0 {
				devices = strings.Split(splits[0], ",")
			}

			if len(splits) > 1 {
				splits2 := strings.Split(splits[1], "@")
				source = splits2[0]

				if len(splits2) > 1 {
					splits3 := strings.Split(splits2[1], "#")
					level = splits3[0]
					if len(splits3) > 1 {
						platforms = splits3[1]
					}

				}
			}

			if c.String("device") != "" {
				devices = strings.Split(c.String("device"), ",")
			}

			for k, v := range devices {
				_, err := primitive.ObjectIDFromHex(v)
				if err == nil {
					devices[k] = "prn:::devices:/" + v
				}
			}

			if c.String("source") != "" {
				source = c.String("source")
			}

			if c.String("level") != "" {
				level = c.String("level")
			}

			if c.String("platform") != "" {
				platforms = c.String("platform")
			}

			logFilter := libpvr.LogFilter{
				Devices:   strings.Join(devices, ","),
				Levels:    level,
				Sources:   source,
				Platforms: platforms,
			}

			var logFormatter libpvr.LogFormatter
			logFormat := c.String("template")
			switch logFormat {
			case "json":
				logFormatter = &libpvr.LogFormatterJson{}
				break
			default:
				logFormatter = &libpvr.LogFormatterTemplate{}
				logFormatter.Init("{{ .Device | prn2id " +
					"| sprintf \"%10.10s\" }}" +
					"({{ .LogRev | sprintf \"%3s\" }}) " +
					"{{ .TimeCreated | timeformat \"Stamp\" }}" +
					"{{ .LogPlat | sprintf \"%12s\" }}(" +
					"{{ .LogSource |  basename | sprintf \"%-15.15s\"}})" +
					": {{ .LogText }}")
				break
			}

			for {
				logEntries, cursorID, err := session.DoLogs(c.App.Metadata["PVR_BASEURL"].(string), nil, rev, from, to, true, logFilter)

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
					logFormatter.DoLog(v)
					// advance "from" cursor
					from = &v.TimeCreated
				}

				for {
					logEntries, cursorID, err = session.DoLogsCursor(c.App.Metadata["PVR_BASEURL"].(string), cursorID)

					if err != nil {
						return cli.NewExitError("Error getting device list: "+err.Error(), 4)
					}

					for _, v := range logEntries {
						logFormatter.DoLog(v)
						// advance "from" cursor
						from = &v.TimeCreated
					}
					// if we reach end of cursor we have exhausted it and will sleep
					// before trying to get new logs starting from last timestamp
					if len(logEntries) == 0 {
						time.Sleep(time.Duration(1 * time.Second))
						break
					}
				}
			}
		},
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:   "from,f",
				Usage:  "Datetime in RFC3339 format, e.g.: --from=2006-01-02T15:04:05+06:00",
				EnvVar: "PVR_LOGS_FROM_DATE",
			},
			cli.IntFlag{
				Name:   "rev,r",
				Usage:  "Get logs for specific revision (rev) number, e.g. --rev=1 or -rev=-1 (disabled), with this flag --from will not be defaulting to NOW",
				EnvVar: "PVR_LOGS_REV",
				Value:  -1,
			},
			cli.StringFlag{
				Name:   "template,s",
				Usage:  "template for log output formatting: short(default), json, <golang-time-format>",
				EnvVar: "PVR_LOGS_TEMPLATE",
			},
			cli.StringFlag{
				Name:   "to,t",
				Usage:  "Datetime in RFC3339 format, e.g.:--to=2006-01-02T15:04:05+06:00",
				EnvVar: "PVR_LOGS_TO_DATE",
			},
			cli.StringFlag{
				Name:   "platform,p",
				Usage:  "Platform, e.g.: --platform=linux,windows",
				EnvVar: "PVR_LOGS_PLATFORM",
			},
			cli.StringFlag{
				Name:   "device,d",
				Usage:  "device, e.g.: --device=5ee13a6087dfb60008ab8f5c,7ee13a6087dfb60008ab8f5e",
				EnvVar: "PVR_LOGS_DEVICE",
			},
			cli.StringFlag{
				Name:   "source,src",
				Usage:  "source, e.g.: --source=/pantavisor.log,/updater,/pvr-sdk-lxc",
				EnvVar: "PVR_LOGS_SOURCE",
			},
			cli.StringFlag{
				Name:   "level,l",
				Usage:  "level, e.g.: --level=DEBUG,INFO",
				EnvVar: "PVR_LOGS_LEVEL",
			},
		},
	}
}
