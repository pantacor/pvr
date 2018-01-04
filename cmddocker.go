//
// Copyright 2018  Pantacor Ltd.
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
	"log"
	"os"

	"github.com/urfave/cli"
)

func CommandDocker() cli.Command {
	return cli.Command{
		Name:    "docker",
		Aliases: []string{"dock"},
		Usage:   "import or sync blobs from docker repository.",
		Before: func(c *cli.Context) error {
			return nil
		},
		Subcommands: []cli.Command{
			cli.Command{
				Name:  "add",
				Usage: "add a new docker app/platform",
				Flags: []cli.Flag{
					cli.StringFlag{Name: "arch, a"},
					cli.StringFlag{Name: "registry, r"},
					cli.StringFlag{Name: "username, u"},
					cli.StringFlag{Name: "password, p"},
				},
				Action: func(c *cli.Context) error {
					wd, err := os.Getwd()
					if err != nil {
						return cli.NewExitError(err, 1)
					}

					pvr, err := NewPvr(c.App, wd)
					if err != nil {
						return cli.NewExitError(err, 2)
					}

					arch := c.String("arch")
					reg := c.String("registry")
					name := c.String("name")
					user := c.String("username")
					pass := c.String("password")

					dock, err := pvr.NewDocker(reg, arch, user, pass)

					log.Println("user: " + user)
					log.Println("pass: " + pass)

					if err != nil {
						return cli.NewExitError("Error creating docker backend: "+err.Error(), 1)
					}

					if len(c.Args()) != 1 {
						return errors.New("Must have exactly one argument: filename of target squashfs")
					}

					namedRef := c.Args()[0]
					repo, ref := splitNamedRef(namedRef)

					err = dock.ToSquash(repo, ref, name)

					if err != nil {
						return cli.NewExitError("Error creating squash: "+err.Error(), 1)
					}

					err = pvr.AddFile(c.Args())
					if err != nil {
						return cli.NewExitError(err, 3)
					}

					return nil
				},
			},
			{
				Name:  "refresh",
				Usage: "refresh an already existing docker app from given repository and tags",
				Action: func(c *cli.Context) error {
					wd, err := os.Getwd()
					if err != nil {
						return cli.NewExitError(err, 1)
					}

					pvr, err := NewPvr(c.App, wd)
					if err != nil {
						return cli.NewExitError(err, 2)
					}

					err = pvr.AddFile(c.Args())
					if err != nil {
						return cli.NewExitError(err, 3)
					}

					return nil
				},
			},
		},
	}
}
