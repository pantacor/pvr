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
	"strings"

	"github.com/urfave/cli"
	"gitlab.com/pantacor/pvr/libpvr"
)

// CommandLocalGet : pvr local get command
func CommandLocalGet() cli.Command {
	cmd := cli.Command{
		Name:        "get",
		Aliases:     []string{"ge"},
		ArgsUsage:   "<DEVICE_IP|HOSTNAME> [REVISION] [CGI_PORT]",
		Usage:       "pvr local get <DEVICE_IP|HOSTNAME> [REVISION] [CGI_PORT]",
		Description: "Get a local device updates",
		Action: func(c *cli.Context) error {
			wd, err := os.Getwd()
			if err != nil {
				return cli.NewExitError(err, 1)
			}
			if _, err := os.Stat(wd + "/.pvr/json"); os.IsNotExist(err) {
				return cli.NewExitError(errors.New("Please cd to a device folder and try again"), 2)
			}

			deviceIP := ""
			revision := "0"
			deviceCGIPort := "2005"
			if c.NArg() < 1 {
				return cli.NewExitError(errors.New("Device ip or hostname is required for pvr local clone <DEVICE_IP|HOSTNAME> [REVISION] [CGI_PORT]. See --help"), 3)
			} else if c.NArg() == 1 {
				deviceIP = c.Args().Get(0) + ":" + deviceCGIPort
			} else if c.NArg() == 2 {
				deviceIP = c.Args().Get(0) + ":" + deviceCGIPort
				revision = c.Args().Get(1)
			} else if c.NArg() == 3 {
				deviceIP = c.Args().Get(0) + ":" + c.Args().Get(2)
				revision = c.Args().Get(1)
			}
			if !strings.HasPrefix(deviceIP, "http://") {
				deviceIP = "http://" + deviceIP
			}

			//downloading tar file
			filename, err := libpvr.DownloadFile(deviceIP + "/cgi-bin/pvrlocal?revision=" + revision)
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
			fmt.Println("\nUpdated Successfully from local device:" + deviceIP + "\n")

			return nil
		},
	}

	return cmd
}
