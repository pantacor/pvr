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
	"errors"
	"fmt"
	"os"

	"github.com/urfave/cli"
	"gitlab.com/pantacor/pvr/libpvr"
)

// CommandLocalGet : pvr local get command
func CommandLocalGet() cli.Command {
	cmd := cli.Command{
		Name:        "get",
		Aliases:     []string{"ge"},
		ArgsUsage:   "[device_ip] [revision]",
		Usage:       "pvr local get [device_ip] [revision]",
		Description: "Get a local device updates",
		Action: func(c *cli.Context) error {
			wd, err := os.Getwd()
			if err != nil {
				return cli.NewExitError(err, 1)
			}
			if _, err := os.Stat(wd + "/.pvr/json"); os.IsNotExist(err) {
				return cli.NewExitError(errors.New("Please cd to a device folder and try again"), 2)
			}

			revision := "0"
			deviceIP := "http://localhost:2005"

			if c.NArg() == 1 {
				deviceIP = c.Args().Get(0)
			} else if c.NArg() == 2 {
				deviceIP = c.Args().Get(0)
				revision = c.Args().Get(1)
			}
			//downloading tar file
			filename, err := libpvr.DownloadFile(deviceIP + "/cgi-bin/pvrlocal/" + revision)
			if err != nil {
				return cli.NewExitError(err, 2)
			}
			//unpack tarball
			err = libpvr.Untar(wd+"/.pvr", wd+"/"+filename)
			if err != nil {
				return cli.NewExitError(err, 3)
			}
			//removing tar file
			err = os.Remove(wd + "/" + filename)
			if err != nil {
				return cli.NewExitError(err, 4)
			}
			fmt.Println("\nDevice Updated Successfully from local\n")

			return nil
		},
	}

	return cmd
}
