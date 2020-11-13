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
	"os"
	"strings"

	"gitlab.com/pantacor/pvr/libpvr"

	"github.com/urfave/cli"
)

var RunlevelFlagUsageNoDefault = "runlevel to install app to, valid runlevels at this point: root, platform, app [default: \"\"]"

func CommandAppInstall() cli.Command {
	cmd := cli.Command{
		Name:        "install",
		Aliases:     []string{"ai"},
		ArgsUsage:   "[appname]",
		Usage:       "install new applications.",
		Description: "generates application files",
		Action: func(c *cli.Context) error {
			wd, err := os.Getwd()
			if err != nil {
				return cli.NewExitError(err, 1)
			}

			if c.NArg() < 1 {
				return cli.NewExitError("app-install needs application argument. See --help", 2)
			}

			session, err := libpvr.NewSession(c.App)

			if err != nil {
				return cli.NewExitError(err, 4)
			}

			pvr, err := libpvr.NewPvr(session, wd)
			if err != nil {
				return cli.NewExitError(err, 2)
			}

			err = pvr.CheckIfIsRunningAsRoot()
			if err == libpvr.ErrNeedBeRoot {
				err = pvr.RunAsRoot()
				if err != nil {
					return cli.NewExitError(err, 3)
				}
			} else if err != nil {
				return cli.NewExitError(err, 3)
			}

			appname := c.Args().Get(0)
			username := c.String("username")
			password := c.String("password")

			// fix up trailing/leading / from appnames
			appname = strings.Trim(appname, "/")

			appManifest, err := pvr.GetApplicationManifest(appname)
			if err != nil {
				return cli.NewExitError(err, 2)
			}

			source := c.String("source")
			if source == "" {
				source = appManifest.DockerSource
			}

			err = libpvr.ValidateSourceFlag(source)
			if err != nil {
				return cli.NewExitError(err, 3)
			}

			dockerName := appManifest.DockerName
			repoDigest := appManifest.DockerDigest
			from := repoDigest

			if source == "remote" && !strings.Contains(repoDigest, "@") {
				from = dockerName + "@" + repoDigest
			}

			app := libpvr.AppData{
				Appname:      appname,
				From:         from,
				Source:       source,
				Username:     username,
				Password:     password,
				TemplateArgs: map[string]interface{}{},
			}

			err = pvr.FindDockerImage(&app)
			if err != nil {
				fmt.Println("\nSeems like you have an invalid docker digest value in your " + appname + "/src.json file\n")
				fmt.Println("\nPlease run \"pvr app update " + appname + " --source=" + c.String("source") + "\" to auto fix it or update docker_digest field by editing " + appname + "/src.json  to fix it manually\n")
				return cli.NewExitError(err, 3)
			}
			err = pvr.InstallApplication(app)
			if err != nil {
				return cli.NewExitError(err, 3)
			}

			fmt.Println("Application installed")

			return nil
		},
	}

	cmd.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   "username, u",
			Usage:  "Use `PVR_REGISTRY_USERNAME` for authorization with docker registrar",
			EnvVar: "PVR_REGISTRY_USERNAME",
		},
		cli.StringFlag{
			Name:   "password, p",
			Usage:  "Use `PVR_REGISTRY_PASSWORD` for authorization with docker registrar",
			EnvVar: "PVR_REGISTRY_PASSWORD",
		},
		cli.StringFlag{
			Name:   "source",
			Usage:  SourceFlagUsage,
			EnvVar: "PVR_SOURCE",
			Value:  "",
		},
		cli.StringFlag{
			Name:   "runlevel",
			Usage:  RunlevelFlagUsageNoDefault,
			EnvVar: "PVR_RUNLEVEL",
			Value:  "app",
		},
	}

	return cmd
}
