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
	"os"
	"strings"

	"gitlab.com/pantacor/pvr/libpvr"
	"gitlab.com/pantacor/pvr/models"

	"github.com/urfave/cli"
)

// SourceFlagUsage : Source Flag Usage Description
var SourceFlagUsage = "Comma separated priority list of source (valid sources: \"local\" and \"remote\")"
var RunlevelFlagUsage = "Runlevel to install container to (valid runlevels: \"data\", \"root\", \"platform\" and \"app\") (deprecated) (default: \"app\")"
var GroupFlagUsage = "Group to install container to (default: last group in groups.json, if exists)"
var RestartPolicyFlagUsage = "Restart policy in case of container modification (valid policies: \"system\" and \"container\")"
var StatusGoalFlagUsage = "Status goal for container after bootup (valid goals: \"MOUNTED\", \"STARTED\" and \"READY\")"

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
			app := &libpvr.AppData{
				Appname:       appname,
				Platform:      c.String("platform"),
				Username:      c.String("username"),
				Password:      c.String("password"),
				From:          c.String("from"),
				Source:        c.String("source"),
				ConfigFile:    c.String("config-json"),
				Volumes:       c.StringSlice("volume"),
				FormatOptions: c.String("format-options"),
				SourceType:    c.String("type"),
				TemplateArgs:  templateArgs,
			}

			if c.IsSet("group") && c.IsSet("runlevel") {
				return cli.NewExitError(errors.New("ERROR: you must not use --runlevel and --group at the same time"), 5)
			}

			if c.String("group") != "" {
				app.TemplateArgs["PV_GROUP"] = c.String("group")
			}
			if c.String("runlevel") != "" {
				app.TemplateArgs["PV_RUNLEVEL"] = c.String("runlevel")
			}
			if c.String("restart-policy") != "" {
				app.TemplateArgs["PV_RESTART_POLICY"] = c.String("restart-policy")
			}
			if c.String("status-goal") != "" {
				app.TemplateArgs["PV_STATUS_GOAL"] = c.String("status-goal")
			}
			if c.String("base") != "" {
				app.DoOverlay = true
				app.Base = strings.Trim(c.String("base"), "/")
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
			Name:   "platform",
			Usage:  "docker platform to resolve",
			EnvVar: "PVR_PLATFORM",
		},
		cli.StringFlag{
			Name:   "source",
			Usage:  SourceFlagUsage,
			EnvVar: "PVR_SOURCE",
			Value:  "remote,local",
		},
		cli.StringFlag{
			Name:   "type, t",
			Usage:  fmt.Sprintf("Type of source. available types [%s, %s, %s]", models.SourceTypeDocker, models.SourceTypePvr, models.SourceTypeRootFs),
			EnvVar: "PVR_SOURCE_TYPE",
			Value:  models.SourceTypeDocker,
		},
		cli.StringFlag{
			Name:   "runlevel",
			Usage:  RunlevelFlagUsage,
			EnvVar: "PVR_RUNLEVEL",
		},
		cli.StringFlag{
			Name:   "group",
			Usage:  GroupFlagUsage,
			EnvVar: "PVR_GROUP",
		},
		cli.StringFlag{
			Name:   "restart-policy",
			Usage:  RestartPolicyFlagUsage,
			EnvVar: "PVR_RESTART_POLICY",
		},
		cli.StringFlag{
			Name:   "status-goal",
			Usage:  StatusGoalFlagUsage,
			EnvVar: "PVR_STATUS_GOAL",
		},
		cli.StringFlag{
			Name:   "base",
			Usage:  "Base rootfs to create patch from",
			EnvVar: "PVR_APP_ADD_BASE",
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
