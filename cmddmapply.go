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
	"os"

	"github.com/urfave/cli"
	"gitlab.com/pantacor/pvr/libpvr"
)

func CommandDmApply() cli.Command {
	return cli.Command{
		Name:      "dm-apply",
		Aliases:   []string{"dma"},
		ArgsUsage: "[<component-prefix>]",
		Usage:     "apply device mapper state transform of a normalized pantavisor state; only apply for _dm matches for platforms starting with '<component-prefix>'",
		Action: func(c *cli.Context) error {
			wd, err := os.Getwd()
			if err != nil {
				return err
			}

			session, err := libpvr.NewSession(c.App)

			if err != nil {
				return cli.NewExitError(err, 4)
			}

			pvr, err := libpvr.NewPvr(session, wd)
			if err != nil {
				return cli.NewExitError(err, 2)
			}

			if len(c.Args()) > 0 {
				err = pvr.DmCVerityApply(c.Args()[0])
			} else {
				err = pvr.DmCVerityApply("")

			}
			if err != nil {
				return cli.NewExitError(err, 3)
			}

			return nil
		},
		Flags: []cli.Flag{},
	}
}
