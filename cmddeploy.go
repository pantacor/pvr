//
// Copyright 2021  Pantacor Ltd.
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
	"path"

	"github.com/urfave/cli"
	"gitlab.com/pantacor/pvr/libpvr"
)

func CommandDeploy() cli.Command {
	return cli.Command{
		Name:        "deploy",
		Aliases:     []string{"d"},
		ArgsUsage:   "<deploy-dir> [<repository>[#<part>] | [<USER_NICK>/<DEVICE_NICK>[#<part>]] ...",
		Usage:       "deploy the-repository to 'deploy-dir'",
		Description: "Deploy will make objects hard linked to the object pool and will canonical bsp links in .pv directory",
		Action: func(c *cli.Context) error {
			wd, err := os.Getwd()
			if err != nil {
				return cli.NewExitError(err, 1)
			}

			session, err := libpvr.NewSession(c.App)

			if err != nil {
				return cli.NewExitError(err, 4)
			}

			var deployDir string
			var repoPath string
			var sourceSpec string

			if c.NArg() == 0 {
				return cli.NewExitError("Deploy needs at least 1 argument. See --help.", 101)
			} else if c.NArg() == 1 {
				// if we are in a pvr working copy dir, lets remember that to find default
				// repo in case user didnt provide any in args
				sourcePvr, _ := libpvr.NewPvr(session, wd)
				if err != nil {
					return cli.NewExitError("Deploy to have a repository as second argument or be run from inside a local pvr working copy. See --help.", 102)
				}
				repoPath = sourcePvr.Pvrdir
				sourceSpec = sourcePvr.PristineJsonMap["#spec"].(string)
			} else {
				repoPath = c.Args()[1]
			}

			deployDir = c.Args()[0]
			_, err = url.Parse(repoPath)
			if err != nil {
				return cli.NewExitError(err, 3)
			}

			fmt.Println("Starting deployment ...")

			deployPvr, err := libpvr.NewPvr(session, deployDir)
			if err != nil {
				return cli.NewExitError(err, 5)
			}
			if !deployPvr.Initialized {

				fmt.Println("... initializing deploy repository ...")
				objectsDir := c.String("objects")
				if objectsDir == "" {
					objectsDir = path.Join(c.GlobalString("config-dir"), "objects")
				}

				initJSON := fmt.Sprintf("{ \"#spec\": \"%s\" }", sourceSpec)
				err = deployPvr.InitCustom(initJSON, objectsDir)

				if err != nil {
					cli.NewExitError(err, 3)
				}
				deployPvr, err = libpvr.NewPvr(session, deployDir)
				if err != nil {
					return cli.NewExitError(err, 5)
				}
			}

			if c.NArg() > 1 {
				for _, repoPath = range c.Args()[1:] {
					fmt.Println("   - deploying " + repoPath)
					_, err = deployPvr.GetRepo(repoPath, false, true)
					if err != nil {
						return cli.NewExitError(err, 6)
					}
				}
			} else {
				// wd repo gets deployed in this branch
				fmt.Println("   - deploying " + repoPath)
				_, err = deployPvr.GetRepo(repoPath, false, true)
				if err != nil {
					return cli.NewExitError(err, 6)
				}
			}

			fmt.Println("... deploying hardlinks ...")

			deployPvr.ResetWithHardlink()
			if err != nil {
				return cli.NewExitError(err, 8)
			}

			fmt.Println("... deploying .pv/ links...")

			err = deployPvr.DeployPvLinks()

			if err != nil {
				return cli.NewExitError(err, 7)
			}

			fmt.Println("Deployment finished. Now now available at: " + deployDir)

			return nil
		},
	}
}
