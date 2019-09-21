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
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/urfave/cli"
	"gitlab.com/pantacor/pvr/libpvr"
	resty "gopkg.in/resty.v1"
)

// CommandLocalPost : pvr local post command
func CommandLocalPost() cli.Command {
	cmd := cli.Command{
		Name:        "post",
		Aliases:     []string{"po"},
		ArgsUsage:   "<DEVICE_IP|HOSTNAME> [CGI_PORT]",
		Usage:       "pvr local post <DEVICE_IP|HOSTNAME> [CGI_PORT]",
		Description: "Post a local device updates",
		Action: func(c *cli.Context) error {
			wd, err := os.Getwd()
			if err != nil {
				return cli.NewExitError(err, 1)
			}
			if _, err := os.Stat(wd + "/.pvr/json"); os.IsNotExist(err) {
				return cli.NewExitError(errors.New("Please cd to a device folder and try again"), 2)
			}
			deviceIP := ""
			deviceCGIPort := "2005"
			if c.NArg() < 1 {
				return cli.NewExitError(errors.New("Device ip or hostname is required for pvr local post <DEVICE_IP>. See --help"), 3)
			} else if c.NArg() == 1 {
				deviceIP = c.Args().Get(0) + ":" + deviceCGIPort
			} else if c.NArg() == 2 {
				deviceIP = c.Args().Get(0) + ":" + c.Args().Get(1)
			}
			if !strings.HasPrefix(deviceIP, "http://") {
				deviceIP = "http://" + deviceIP
			}
			session, err := libpvr.NewSession(c.App)

			if err != nil {
				return cli.NewExitError(err, 4)
			}

			pvr, err := libpvr.NewPvr(session, wd)
			if err != nil {
				return cli.NewExitError(err, 2)
			}
			pvr.Session.GetConfigDir()
			//Generate a tar file
			tarfilename := "device.tar.gz"
			err = pvr.Export(tarfilename)
			if err != nil {
				return cli.NewExitError(err, 3)
			}
			//get sha of tar file
			sha, err := libpvr.FiletoSha(wd + "/" + tarfilename)
			if err != nil {
				return cli.NewExitError(err, 4)
			}
			//read tar file content
			fileContent, err := ioutil.ReadFile(wd + "/" + tarfilename)
			if err != nil {
				return cli.NewExitError(err, 3)
			}
			req := resty.R()
			req.SetBody(fileContent)
			res, err := req.Post(deviceIP + "/cgi-bin/pvrlocal?sha=" + sha)
			if err != nil {
				return cli.NewExitError(err, 4)
			}
			if res.StatusCode() == http.StatusOK {
				//removing tar file
				err = os.Remove(wd + "/" + tarfilename)
				if err != nil {
					return cli.NewExitError(err, 4)
				}
				fmt.Println("\nPosted Successfully to local device:" + deviceIP + "\n")
			} else {
				fmt.Println("\nError posting to local device:" + deviceIP + "\n")
				log.Print(res)
			}

			return nil
		},
	}

	return cmd
}
