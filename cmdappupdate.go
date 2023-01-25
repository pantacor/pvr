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
	"fmt"
	"os"
	"path"
	"strings"

	"gitlab.com/pantacor/pvr/libpvr"
	"gitlab.com/pantacor/pvr/models"

	"github.com/urfave/cli"
)

func CommandAppUpdate() cli.Command {
	cmd := cli.Command{
		Name:        "update",
		Aliases:     []string{"au"},
		ArgsUsage:   "[appname]",
		Usage:       "update an existing application.",
		Description: "update application files",
		Action: func(c *cli.Context) error {
			wd, err := os.Getwd()
			if err != nil {
				return cli.NewExitError(err, 1)
			}

			if c.NArg() < 1 {
				return cli.NewExitError("app-update needs application argument. See --help", 2)
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

			// fix up trailing/leading / from appnames
			appname := strings.Trim(c.Args().Get(0), "/")
			trackURL, err := pvr.GetTrackURL(appname)
			if err != nil {
				return cli.NewExitError(err, 2)
			}
			err = libpvr.ValidateSourceFlag(c.String("source"))
			if err != nil {
				return cli.NewExitError(err, 3)
			}
			if c.String("from") != "" {
				trackURL = c.String("from")
			}

			appManifest, err := pvr.GetApplicationManifest(appname)
			if err != nil {
				return cli.NewExitError(err, 4)
			}

			app := libpvr.AppData{
				Appmanifest:  appManifest,
				Appname:      appname,
				From:         trackURL,
				Source:       c.String("source"),
				Platform:     c.String("platform"),
				Username:     c.String("username"),
				Password:     c.String("password"),
				SourceType:   c.String("type"),
				TemplateArgs: map[string]interface{}{},
			}

			pvr.SetSourceTypeFromManifest(&app, nil)

			if c.String("runlevel") != "" {
				app.TemplateArgs["PV_RUNLEVEL"] = c.String("runlevel")
			}

			if (c.Bool("patch") || appManifest.DockerOvlDigest != "" || appManifest.Base != "") && !c.Bool("newbase") {
				app.DoOverlay = true
			}

			if c.String("base") != "" && !c.Bool("newbase") {
				app.DoOverlay = true
				app.Base = strings.Trim(c.String("base"), "/")
			}

			err = pvr.UpdateApplication(app)
			if err != nil {
				return cli.NewExitError(err, 3)
			}

			if !app.DoOverlay {
				os.Remove(path.Join(pvr.Dir, appname, libpvr.SQUASH_OVL_FILE+libpvr.DOCKER_DIGEST_SUFFIX))
				os.Remove(path.Join(pvr.Dir, appname, libpvr.SQUASH_OVL_FILE))
			}

			fmt.Println("Application updated")
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
			Name:   "type, t",
			Usage:  fmt.Sprintf("Type of source. available types [%s, %s, %s]", models.SourceTypeDocker, models.SourceTypePvr, models.SourceTypeRootFs),
			EnvVar: "PVR_SOURCE_TYPE",
		},
		cli.StringFlag{
			Name:   "source",
			Usage:  SourceFlagUsage,
			EnvVar: "PVR_SOURCE",
			Value:  "",
		},
		cli.StringFlag{
			Name:   "platform",
			Usage:  "docker platform to resolve",
			EnvVar: "PVR_PLATFORM",
		},
		cli.BoolFlag{
			Name:   "newbase",
			Usage:  "clean patch overlay",
			EnvVar: "PVR_APP_UPDATE_PATCH_CLEAN",
		},
		cli.BoolFlag{
			Name:   "patch",
			Usage:  "update the patch overlay",
			EnvVar: "PVR_APP_UPDATE_PATCH",
		},
		cli.StringFlag{
			Name:   "base",
			Usage:  "Base rootfs to create patch from",
			EnvVar: "PVR_APP_ADD_BASE",
		},
		cli.StringFlag{
			Name:   "from",
			Usage:  "Update docker_name and docker_tag fields before updating according to value",
			EnvVar: "PVR_APP_FROM",
			Value:  "",
		},
	}

	return cmd
}
