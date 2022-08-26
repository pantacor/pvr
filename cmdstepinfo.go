//
// Copyright 2022  Pantacor Ltd.
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
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path"

	"github.com/urfave/cli"
	"gitlab.com/pantacor/pvr/libpvr"
)

func CommandStepInfo() cli.Command {
	return cli.Command{
		Name:        "stepinfo",
		Aliases:     []string{"ri"},
		ArgsUsage:   "<remote-step-ref>",
		Usage:       "display step info from remote repo",
		Description: "display step info from remote repo",
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
				repoUri = pvr.Pvrdir
			} else {
				repoUri = c.Args()[0]
			}

			repoUri, err = libpvr.FixupRepoRef(repoUri)
			if err != nil {
				return cli.NewExitError(err, 7)
			}

			pvrRemote, err := pvr.RemoteInfo(repoUri)

			if err != nil {
				return cli.NewExitError(err, 1)
			}

			stateURL, err := url.Parse(pvrRemote.JsonGetUrl)

			if err != nil {
				return cli.NewExitError(err, 1)
			}

			stateURL.Path = path.Dir(stateURL.Path)
			stateJsonMap, err := pvr.GetJson(stateURL.String())

			if err != nil {
				return cli.NewExitError(err, 1)
			}

			jsonData, err := json.MarshalIndent(stateJsonMap, "", "    ")

			if err != nil {
				return cli.NewExitError(err, 1)
			}

			if c.Bool("canonical") {
				jsonData, err = libpvr.FormatJsonC(jsonData)
				if err != nil {
					cli.NewExitError(err, 4)
				}
			} else {
				jsonData, err = libpvr.FormatJson(jsonData)
				if err != nil {
					cli.NewExitError(err, 4)
				}
			}

			fmt.Println(string(jsonData))

			return nil
		},
		Flags: []cli.Flag{
			cli.BoolFlag{
				Name:  "canonical, c",
				Usage: "Format Output in Canonical JSON",
			},
		},
	}
}
