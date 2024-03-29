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
	"os"
	"strings"

	"github.com/urfave/cli"
	"gitlab.com/pantacor/pvr/libpvr"
)

func CommandDmConvert() cli.Command {
	return cli.Command{
		Name:      "dm-convert",
		Aliases:   []string{"dmc"},
		ArgsUsage: "<container[/volume]> <volume>",
		Usage:     "convert a volume to device mappers dm-verity based volume",
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

			if len(c.Args()) < 1 {
				return cli.NewExitError(errors.New("missing arguments, see --help"), 2)
			}

			container := c.Args()[0]
			var volume string

			if len(c.Args()) == 2 {
				volume = c.Args()[1]
			} else {
				_arr := strings.SplitN(container, "/", 2)
				if len(_arr) <= 1 {
					err = errors.New("ERROR: volume could not be found; see --help")
					return cli.NewExitError(err, 3)
				}
				container = _arr[0]
				volume = _arr[1]
			}

			err = pvr.DmCVerityConvert(container, volume)
			if err != nil {
				return cli.NewExitError(err, 4)
			}

			return nil
		},
		Flags: []cli.Flag{},
	}
}
