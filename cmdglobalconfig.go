//
// Copyright 2019-2023  Pantacor Ltd.
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
	"fmt"
	"log"
	"os"

	"github.com/urfave/cli"
	"gitlab.com/pantacor/pvr/libpvr"
)

// CommandGlobalConfig read or set configuration
func CommandGlobalConfig() cli.Command {
	return cli.Command{
		Name:      "global-config",
		ArgsUsage: "",
		Usage:     "pvr config",
		Action: func(c *cli.Context) error {
			wd, err := os.Getwd()
			if err != nil {
				return cli.NewExitError(err.Error(), 1)
			}

			session, err := libpvr.NewSession(c.App)
			if err != nil {
				return cli.NewExitError(err.Error(), 4)
			}

			pvr, err := libpvr.NewPvr(session, wd)
			if err != nil {
				return cli.NewExitError(err.Error(), 2)
			}

			config := session.Configuration

			if c.NArg() > 0 {
				config, err = pvr.SetConfiguration(c.Args())
				if err != nil {
					return cli.NewExitError(err.Error(), 2)
				}
			}

			json, err := json.MarshalIndent(config, "", "    ")
			if err != nil {
				log.Fatal(err)
			}

			fmt.Println(string(json))
			return nil
		},
	}
}
