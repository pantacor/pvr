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
	"fmt"

	"github.com/urfave/cli"
	"gitlab.com/pantacor/pvr/libpvr"
)

func CommandLogin() cli.Command {
	return cli.Command{
		Name:        "login",
		Aliases:     []string{"lo"},
		ArgsUsage:   "[auth-endpoint]",
		Usage:       "pvr login https://api.pantahub.com/auth/auth_status",
		Description: "Login to pantahub with your username & password with an optional end point",
		Action: func(c *cli.Context) error {
			APIURL := ""
			if c.NArg() > 1 {
				return errors.New("post can have at most 1 argument. See --help.")
			} else if c.NArg() == 0 {
				APIURL = c.App.Metadata["PVR_BASEURL"].(string) + "/auth/auth_status"
			} else {
				APIURL = c.Args()[0]
			}
			session, err := libpvr.NewSession(c.App)

			if err != nil {
				return cli.NewExitError(err, 4)
			}
			response, err := session.Login(APIURL, false)
			if err != nil {
				return cli.NewExitError(err, 4)
			}

			if response != nil {
				fmt.Println("Response of GET " + APIURL)
				err = libpvr.LogPrettyJSON(response.Body())
				if err != nil {
					return cli.NewExitError(err, 2)
				}
			}
			fmt.Println("LoggedIn Successfully!")
			return nil
		},
	}
}
