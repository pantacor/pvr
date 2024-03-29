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

func CommandCommit() cli.Command {
	return cli.Command{
		Name:      "commit",
		Aliases:   []string{"ci"},
		ArgsUsage: "",
		Usage:     "commit status changes",
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

			commitmsg := c.String("message")
			if commitmsg == "" {
				commitmsg = "** No commit message **"
			}
			isCheckpoint := c.Bool("checkpoint")

			err = pvr.Commit(commitmsg, isCheckpoint)
			if err != nil {
				return err
			}

			return nil
		},
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "message, m",
				Usage: "provide a commit message",
			},
			cli.BoolFlag{
				Name:  "checkpoint, c",
				Usage: "commit an updated checkpoint token to ensure this revision will get properly tested and checkpointed as a fallback revision",
			},
		},
	}
}
