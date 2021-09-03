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
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/urfave/cli"
	"gitlab.com/pantacor/pvr/libpvr"
)

func CommandGet() cli.Command {
	return cli.Command{
		Name:        "get",
		Aliases:     []string{"g"},
		ArgsUsage:   "[<repository>[#<part>] [<target-repository>]] | [<USER_NICK>/<DEVICE_NICK>[#<part>]]",
		Usage:       "get update target-repository from repository",
		Description: "default target-repository is the local .pvr one. If not <repository> is provided the last one is used. <part> can be one of 'bsp' or $appname.",
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

			var repoUri string

			if c.NArg() > 1 {
				return errors.New("Get can have at most 1 argument. See --help.")
			} else if c.NArg() == 0 {
				repoUri = ""
			} else {
				repoUri, err = libpvr.FixupRepoRef(c.Args()[0])
				if err != nil {
					return cli.NewExitError(err, 1)
				}
			}

			objectsCount, err := pvr.GetRepo(repoUri, false, true)
			if err != nil {
				return cli.NewExitError(err, 3)
			}

			fmt.Println("\nImported " + strconv.Itoa(objectsCount) + " objects to " + pvr.Objdir)
			fmt.Println("\n\nRun pvr checkout to checkout the changed files into the workspace.")

			return nil
		},
	}
}
