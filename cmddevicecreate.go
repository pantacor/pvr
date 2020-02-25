//
// Copyright 2019  Pantacor Ltd.
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
	"os"

	"gitlab.com/pantacor/pvr/libpvr"

	"github.com/urfave/cli"
)

func CommandDeviceCreate() cli.Command {
	cmd := cli.Command{
		Name:        "create",
		Aliases:     []string{"cr"},
		ArgsUsage:   "[DEVICE_NICK]",
		Usage:       "pvr device create [DEVICE_NICK]",
		Description: "Creates a new device",
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

			baseURL := c.App.Metadata["PVR_BASEURL"].(string)
			deviceNick := ""

			if c.NArg() > 1 {
				return errors.New("Device create can have at most 1 argument. See --help")
			} else if c.NArg() == 1 {
				deviceNick = c.Args()[0]
			}
			state, err := libpvr.StructToMap(pvr.PristineJsonMap)
			if err != nil {
				return cli.NewExitError(err, 2)
			}
			err = pvr.PutObjects(baseURL+"/objects/", true)
			if err != nil {
				return cli.NewExitError(err, 3)
			}
			// Create device
			deviceResponse, err := session.CreateDevice(baseURL, deviceNick)
			if err != nil {
				return cli.NewExitError(err, 2)
			}
			err = libpvr.HandleNilRestResponse(deviceResponse, err)
			if err != nil {
				return cli.NewExitError(err, 2)
			}
			responseData := map[string]interface{}{}
			err = json.Unmarshal(deviceResponse.Body(), &responseData)
			if err != nil {
				return cli.NewExitError(err, 2)
			}
			// Login device
			dToken, err := libpvr.LoginDevice(
				baseURL,
				responseData["prn"].(string),
				responseData["secret"].(string),
			)
			if err != nil {
				return cli.NewExitError(err, 2)
			}
			//Create trail
			_, err = libpvr.CreateTrail(baseURL, dToken, state)
			if err != nil {
				return cli.NewExitError(err, 2)
			}
			libpvr.LogPrettyJSON(deviceResponse.Body())
			fmt.Println("Device Created Successfully")
			return nil
		},
	}

	return cmd
}
