// Copyright 2017-2023  Pantacor Ltd.
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
	"fmt"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"path"

	"github.com/urfave/cli"
	"gitlab.com/pantacor/pvr/libpvr"
)

func CommandClone() cli.Command {
	return cli.Command{
		Name:        "clone",
		Aliases:     []string{"c"},
		ArgsUsage:   "<repository> | <USER_NICK>/<DEVICE_NICK> [directory]",
		Usage:       "clone repository to a new target directory",
		Description: "this combines operations: new, get, checkout",
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

			baseURL := c.App.Metadata["PVR_BASEURL"].(string)

			searchTerm := c.Args()[c.NArg()-1]
			session.SuggestNicks(searchTerm, baseURL)

		},
		Action: func(c *cli.Context) error {
			wd, err := os.Getwd()
			if err != nil {
				return cli.NewExitError(err, 1)
			}
			session, err := libpvr.NewSession(c.App)
			if err != nil {
				return cli.NewExitError(err, 4)
			}

			if c.NArg() < 1 {
				return cli.NewExitError("clone needs need repository argument. See --help", 2)
			}

			origURL, err := url.Parse(c.Args().Get(0))
			if err != nil {
				return cli.NewExitError("clone must have a valid URL as argument", 2)
			}

			deviceString, err := libpvr.FixupRepoRef(c.Args().Get(0))
			if err != nil {
				return cli.NewExitError(err, 7)
			}

			// we use the url as passed in before repo ref to gather the
			// basename to use as the default clone target.
			base := path.Base(origURL.Path)
			base = path.Join(wd, base)
			if c.NArg() == 2 {
				base = c.Args().Get(1)
			}
			if !path.IsAbs(base) {
				base = path.Join(wd, base)
			}
			baseDir := path.Dir(base)

			tempdir, err := ioutil.TempDir(baseDir, "pvr-clone-")
			if err != nil {
				return cli.NewExitError(err, 4)
			}

			defer os.RemoveAll(tempdir)
			libpvr.SetTempFilesInterrupHandler(tempdir)

			session, err = libpvr.NewSession(c.App)

			if err != nil {
				return cli.NewExitError(err, 4)
			}

			pvr, err := libpvr.NewPvr(session, tempdir)
			if err != nil {
				return cli.NewExitError(err, 2)
			}

			objectsDir := c.String("objects")
			if objectsDir == "" {
				objectsDir = path.Join(c.GlobalString("config-dir"), "objects")
			}

			spec := c.String("spec")
			initJson := fmt.Sprintf("{ \"#spec\": \"%s\" }", spec)
			err = pvr.InitCustom(initJson, objectsDir)
			if err != nil {
				return cli.NewExitError(err, 6)
			}

			pvr, err = libpvr.NewPvr(session, tempdir)
			if err != nil {
				return cli.NewExitError(err, 20)
			}

			_, err = pvr.GetRepo(deviceString, false, false)
			if err != nil {
				return cli.NewExitError(err, 7)
			}

			err = pvr.Reset(c.Bool("canonical"))

			if err != nil {
				return cli.NewExitError(err, 8)
			}

			err = os.Rename(tempdir, base)
			if err != nil {
				return cli.NewExitError(err, 9)
			}

			fmt.Println("Successfully cloned: " + base)

			return nil
		},
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:   "objects, o",
				EnvVar: "PVR_OBJECTS_DIR",
				Usage:  "Use `OBJECTS` directory for storing the file objects. Can be absolue or relative to working directory.",
			},
			cli.BoolFlag{
				Name:   "canonical, c",
				Usage:  "clone working copy json files using canonical json format",
				EnvVar: "PVR_CANONICAL_JSON",
			},
			cli.StringFlag{
				Name:  "spec, s",
				Usage: "Use `SPEC` as state format (e.g. pantavisor-service-system@1 or pantavisor-multi-platform@1 (legacy)",
				Value: "pantavisor-service-system@1",
			},
		},
	}

}
