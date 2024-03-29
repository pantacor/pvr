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
	"strconv"

	"github.com/urfave/cli"
	"gitlab.com/pantacor/pvr/libpvr"
)

func CommandMerge() cli.Command {
	return cli.Command{
		Name:        "merge",
		Aliases:     []string{"m"},
		ArgsUsage:   "[repository [target-repository]]",
		Usage:       "merge content of repository into target-directory",
		Description: "default target-repository is the local .pvr one. If not <repository> is provided the last one is used.",
		Action: func(c *cli.Context) error {
			wd, err := os.Getwd()
			if err != nil {
				return cli.NewExitError(err, 1)
			}

			session, err := libpvr.NewSession(c.App)

			if err != nil {
				return cli.NewExitError(err, 4)
			}

			pvr, err := libpvr.NewPvr(session, wd)
			if err != nil {
				return cli.NewExitError(err, 2)
			}

			var repoPath string

			if c.NArg() > 1 {
				return errors.New("Merge can have at most 1 argument. See --help.")
			} else if c.NArg() == 0 {
				repoPath = ""
			} else {
				repoPath = c.Args()[0]
			}

			objectsCount, err := pvr.GetRepo(repoPath, true, true, nil)
			if err != nil {
				return cli.NewExitError(err, 3)
			}

			fmt.Println("\nImported " + strconv.Itoa(objectsCount) + " objects to " + pvr.Objdir)
			fmt.Println("\n\nRun pvr checkout to checkout the changed files into the workspace.")

			return nil
		},
	}
}
