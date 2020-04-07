//
// Copyright 2017  Pantacor Ltd.
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
	"log"
	"os"
	"strings"

	"github.com/urfave/cli"
	"gitlab.com/pantacor/pvr/libpvr"
)

func CommandGet() cli.Command {
	return cli.Command{
		Name:        "get",
		Aliases:     []string{"g"},
		ArgsUsage:   "[repository [target-repository]] | [<USER_NICK>/<DEVICE_NICK>]",
		Usage:       "get update target-repository from repository",
		Description: "default target-repository is the local .pvr one. If not <repository> is provided the last one is used.",
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
			session.SuggestNicks(searchTerm, baseURL)
		},
		Action: func(c *cli.Context) error {
			wd, err := os.Getwd()
			if err != nil {
				return cli.NewExitError(err, 1)
			}

			session, err := libpvr.NewSession(c.App)

			if err != nil {
				return cli.NewExitError(err, 4)
			}

			pvr, err := libpvr.NewPvr(session, wd)
			if err != nil {
				return cli.NewExitError(err, 2)
			}

			var repoPath string

			if c.NArg() > 1 {
				return errors.New("Get can have at most 1 argument. See --help.")
			} else if c.NArg() == 0 {
				repoPath = ""
			} else {
				repoPath = c.Args()[0]
				if !libpvr.IsValidUrl(repoPath) {
					//Get owner nick & Device nick & make device repo URL
					userNick := ""
					deviceNick := ""
					splits := strings.Split(repoPath, "/")
					if len(splits) == 1 {
						return cli.NewExitError("Device nick is missing. (syntax:pvr get <USER_NICK>/<DEVICE_NICK>). See --help", 2)
					} else if len(splits) == 2 {
						userNick = splits[0]
						deviceNick = splits[1]
					}
					repoPath = "https://pvr.pantahub.com/" + userNick + "/" + deviceNick
				}
			}

			err = pvr.GetRepo(repoPath, false)
			if err != nil {
				return cli.NewExitError(err, 3)
			}

			return nil
		},
	}
}
