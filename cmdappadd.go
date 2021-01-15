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

	"gitlab.com/pantacor/pvr/libpvr"

	"github.com/urfave/cli"
)

// SourceFlagUsage : Source Flag Usage Description
var SourceFlagUsage = "Comma separated priority list of source (valid sources: local and remote)"
var RunlevelFlagUsage = "runlevel to install app to, valid runlevels at this point: root, platform, app [default: app]"

func CommandAppAdd() cli.Command {
	cmd := cli.Command{
		Name:        "add",
		Aliases:     []string{"aa"},
		ArgsUsage:   "[appname]",
		Usage:       "add new applications.",
		Description: "add new application and generates files",
		Action: func(c *cli.Context) error {
			wd, err := os.Getwd()
			if err != nil {
				return cli.NewExitError(err, 1)
			}

			if c.NArg() < 1 {
				return cli.NewExitError("app-add needs application name argument. See --help", 2)
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
			// fix up trailing/leading / from appnames
			appname = strings.Trim(appname, "/")

			templateArgs := map[string]interface{}{}

			varSlice := c.StringSlice("arg")
			for _, v := range varSlice {
				va := strings.SplitN(v, "=", 2)
				if len(va) == 2 {
					templateArgs[va[0]] = va[1]
				} else {
					templateArgs[va[0]] = ""
				}
			}

			err = libpvr.ValidateSourceFlag(c.String("source"))
			if err != nil {
				return cli.NewExitError(err, 3)
			}
			if c.String("from") == "" {
				return cli.NewExitError(errors.New(" --from flag is required (e.g. --from=nginx)"), 4)
			}
			app := libpvr.AppData{
				Appname:       appname,
				Username:      c.String("username"),
				Password:      c.String("password"),
				From:          c.String("from"),
				Source:        c.String("source"),
				ConfigFile:    c.String("config-json"),
				Volumes:       c.StringSlice("volume"),
				FormatOptions: c.String("format-options"),
				TemplateArgs:  templateArgs,
			}

			if c.String("runlevel") != "" {
				app.TemplateArgs["PV_RUNLEVEL"] = c.String("runlevel")
			}

			err = pvr.FindDockerImage(&app)
			if err != nil {
				return cli.NewExitError(err, 3)
			}
			err = pvr.AddApplication(app)
			if err != nil {
				return cli.NewExitError(err, 3)
			}

			fmt.Println("Application added")

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
			Name:   "from",
			Usage:  "Container image to add",
			EnvVar: "PVR_FROM",
		},
		cli.StringFlag{
			Name:   "source",
			Usage:  SourceFlagUsage,
			EnvVar: "PVR_SOURCE",
			Value:  "remote,local",
		},
		cli.StringFlag{
			Name:   "runlevel",
			Usage:  RunlevelFlagUsage,
			EnvVar: "PVR_RUNLEVEL",
			Value:  "app",
		},
		cli.StringFlag{
			Name:   "config-json",
			Usage:  "Docker image config",
			EnvVar: "PVR_CONFIG_JSON",
		},
		cli.StringSliceFlag{
			Name:   "arg",
			Usage:  "Template Arguments",
			EnvVar: "PVR_TEMPLATE_ARG",
		},
		cli.StringSliceFlag{
			Name:   "volume",
			Usage:  "Persistence volume",
			EnvVar: "PVR_PERSISTENCE_VOLUME",
		},
		cli.StringFlag{
			Name:   "format-options,o",
			Usage:  "Format Options for Target FS (e.g. \"-comp gzip\")",
			EnvVar: "PVR_FORMAT_OPTIONS",
		},
	}

	return cmd
}
