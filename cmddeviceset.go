// Copyright 2019-2023  Pantacor Ltd.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
//	Unless required by applicable law or agreed to in writing, software
//	distributed under the License is distributed on an "AS IS" BASIS,
//	WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//	See the License for the specific language governing permissions and
//	limitations under the License.
package main

import (
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/urfave/cli"
	"gitlab.com/pantacor/pvr/libpvr"
	"gitlab.com/pantacor/pvr/utils/pvjson"
)

func CommandDeviceSet() cli.Command {
	cmd := cli.Command{
		Name:        "set",
		Aliases:     []string{"se"},
		ArgsUsage:   "<NICK|ID> <KEY1>=<VALUE1> [KEY2]=[VALUE2]...[KEY-N]=[VALUE-N]",
		Usage:       "pvr device set <NICK|ID> <KEY1>=<VALUE1> [KEY2]=[VALUE2]...[KEY-N]=[VALUE-N]",
		Description: "Set or Update device user-meta & device-meta fields (Note:If you are logged in as USER then you can update user-meta field but if you are logged in as DEVICE then you can update device-meta field)",
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
			session.SuggestDeviceNicks("", searchTerm, baseURL)
		},
		Action: func(c *cli.Context) error {
			session, err := libpvr.NewSession(c.App)
			if err != nil {
				return cli.NewExitError(err, 1)
			}
			baseURL := c.App.Metadata["PVR_BASEURL"].(string)
			deviceNick := ""
			if c.NArg() >= 2 {
				deviceNick = c.Args()[0]
			} else if c.NArg() == 1 {
				return cli.NewExitError(errors.New("<KEY1>=<VALUE1> [KEY2]=[VALUE2].. is required. See --help"), 2)
			} else {
				return cli.NewExitError(errors.New("<NICK|ID> is required. See --help"), 2)
			}
			data := map[string]interface{}{}
			for k, v := range c.Args() {
				if k > 0 {
					splits := strings.SplitN(v, "=", 2)
					if len(splits) == 2 {
						data[splits[0]] = splits[1]
					} else if len(splits) == 1 {
						data[splits[0]] = nil
					} else {
						return cli.NewExitError(errors.New("<KEY1>=<VALUE1> [KEY2]=[VALUE2].. is required. See --help"), 2)
					}
				}
			}
			authResponse, err := session.GetAuthStatus(baseURL)
			if err != nil {
				return cli.NewExitError(err, 2)
			}
			authResponseData := map[string]interface{}{}
			err = pvjson.Unmarshal(authResponse.Body(), &authResponseData)
			if err != nil {
				return cli.NewExitError(err, 2)
			}
			//Update Device
			if authResponseData["type"] == "USER" {
				updateResponse, err := session.UpdateDevice(baseURL, deviceNick, data, "user-meta")
				if err != nil {
					return cli.NewExitError(err, 2)
				}
				libpvr.LogPrettyJSON(updateResponse.Body())
				fmt.Println("user-meta field Updated Successfully")
			} else if authResponseData["type"] == "DEVICE" {
				updateResponse, err := session.UpdateDevice(baseURL, deviceNick, data, "device-meta")
				if err != nil {
					return cli.NewExitError(err, 2)
				}
				libpvr.LogPrettyJSON(updateResponse.Body())
				fmt.Println("device-meta field Updated Successfully")
			}
			return nil
		},
	}

	return cmd
}
