//
// Copyright 2017  Pantacor Ltd.
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
	"log"
	"os"
	"strings"

	"github.com/urfave/cli"
)

func CommandDockerLxc() cli.Command {
	return cli.Command{
		Name:    "docker-lxc",
		Aliases: []string{"d"},
		Usage:   "docker-lxc addon for pvr",
		Subcommands: []cli.Command{
			{
				Name:      "add",
				ArgsUsage: "[<docker-registry.tld>/]<docker-repo>[:<tag>]",
				Action: func(c *cli.Context) error {
					wd, err := os.Getwd()
					if err != nil {
						return cli.NewExitError(err, 1)
					}

					pvr, err := NewPvr(c.App, wd)
					if err != nil {
						return cli.NewExitError(err, 2)
					}

					if c.NArg() != 1 {
						return cli.NewExitError("wrong number arguments", 3)
					}

					log.Println("docker-lxc in dir " + pvr.Dir)

					arg := c.Args()[0]
					n := strings.Index(arg, "/")

					repoTags := strings.SplitN(arg, ":", 2)
					urlWords := strings.Split(repoTags[0], "/")

					var registryUrl string
					m := 0

					if strings.Index(urlWords[m], ".") > -1 {
						registryUrl = "https://" + urlWords[m]
						m++
					} else {
						registryUrl = "https://registry-1.docker.io/"
					}

					repoPath := ""
					for _, w := range urlWords[m:] {
						repoPath = repoPath + w + "/"
					}
					repoPath = strings.TrimRight(repoPath, "/")

					repoTag := ""
					if len(repoTags) > 1 {
						repoTag = repoTags[1]
					} else {
						repoTag = "latest"
					}

					log.Printf("found / in %d: registry: %s  repo: %s  tag: %s\n", n, registryUrl, repoPath, repoTag)

					err = pvr.AddDockerLxc(registryUrl, repoPath, repoTag)
					if err != nil {
						return cli.NewExitError(err, 5)
					}
					return nil
				},
			},
		},
	}
}
