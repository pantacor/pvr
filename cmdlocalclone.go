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
	"fmt"
	"net/url"
	"os"

	"github.com/urfave/cli"
	"gitlab.com/pantacor/pvr/libpvr"
)

// CommandLocalClone : pvr local clone command
func CommandLocalClone() cli.Command {
	cmd := cli.Command{
		Name:        "clone",
		Aliases:     []string{"cl"},
		ArgsUsage:   "[device_ip] [revision]",
		Usage:       "pvr local clone [device_ip] [revision]",
		Description: "Clone a local device",
		Action: func(c *cli.Context) error {
			wd, err := os.Getwd()
			if err != nil {
				return cli.NewExitError(err, 1)
			}
			revision := "0"
			deviceIP := "http://localhost:2005"

			if c.NArg() == 1 {
				deviceIP = c.Args().Get(0)
			} else if c.NArg() == 2 {
				deviceIP = c.Args().Get(0)
				revision = c.Args().Get(1)
			}

			filename, err := libpvr.DownloadFile(deviceIP + "/cgi-bin/pvrlocal/" + revision)
			if err != nil {
				return cli.NewExitError(err, 2)
			}

			u, err := url.Parse(deviceIP)
			if err != nil {
				return cli.NewExitError(err, 2)
			}
			folderName := u.Hostname()
			//Make device root directory
			err = libpvr.CreateFolder(wd + "/" + folderName + "/.pvr")
			if err != nil {
				return cli.NewExitError(err, 3)
			}
			//unpack tarball
			err = libpvr.Untar(wd+"/"+folderName+"/.pvr", wd+"/"+filename)
			if err != nil {
				return cli.NewExitError(err, 3)
			}
			//removing tar file
			err = os.Remove(wd + "/" + filename)
			if err != nil {
				return cli.NewExitError(err, 4)
			}
			//pvr checkout
			session, err := libpvr.NewSession(c.App)

			if err != nil {
				return cli.NewExitError(err, 4)
			}

			pvr, err := libpvr.NewPvr(session, wd+"/"+folderName)
			if err != nil {
				return cli.NewExitError(err, 2)
			}

			err = pvr.Reset()
			if err != nil {
				return cli.NewExitError(err, 3)
			}
			fmt.Println("\nDevice Cloned Successfully from local\n")

			return nil
		},
	}

	return cmd
}
