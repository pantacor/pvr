//
// Copyright 2017-2021  Pantacor Ltd.
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
	"os"
	"strings"

	"github.com/bmatcuk/doublestar"
	"github.com/urfave/cli"
	"gitlab.com/pantacor/pvr/libpvr"
)

func CommandSigUpdate() cli.Command {
	return cli.Command{
		Name:    "update",
		Aliases: []string{"up"},
		ArgsUsage: "	",
		Usage: "update a pvs@2 signature by using the encoded matchrule",
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

			patterns := []string{}

			for _, v := range c.Args() {
				// either a full pattern or a name in _sigs can be matched
				if strings.HasSuffix(v, ".json") {
					patterns = append(patterns, v)
				} else {
					patterns = append(patterns, "_sigs/"+v+".json")
				}
			}

			// by default we match all in _sigs
			if len(patterns) == 0 {
				patterns = append(patterns, "_sigs/*.json")
			}

			ops := libpvr.PvsOptions{}
			ops.IncludePayLoad = c.Parent().Bool("with-payload")

			keyPath := c.Parent().String("key")
			if keyPath == "" {
				keyPath, err = libpvr.GetFromConfigPvs(
					c.App.Metadata["PVS_CERTS_URL"].(string),
					pvr.Session.GetConfigDir(),
					libpvr.SigKeyFilename,
				)
				if err != nil {
					return cli.NewExitError(err, 127)
				}
			}

			ops.X5cPath = c.Parent().String("x5c")
			if ops.X5cPath == "" {
				ops.X5cPath, err = libpvr.GetFromConfigPvs(
					c.App.Metadata["PVS_CERTS_URL"].(string),
					pvr.Session.GetConfigDir(),
					libpvr.SigX5cFilename,
				)
				if err != nil {
					return cli.NewExitError(err, 127)
				}
			}

			for k, v := range pvr.PristineJsonMap {
				vmap, ok := v.(map[string]interface{})
				if !ok {
					continue
				}
				specV, ok := vmap["#spec"]
				if !ok {
					continue
				}
				specVString, ok := specV.(string)
				if specVString != "pvs@2" {
					continue
				}

				// here we have a pvs@2 spec element at hands.
				// now proceed if we have a match pattern.
				for _, p := range patterns {
					m, err := doublestar.Match(p, k)
					if err != nil {
						return cli.NewExitError(err, 123)
					}
					if m {
						fmt.Print("Updating pvs signature @ " + k)
						err = pvr.JwsSignPvs(keyPath, k, &ops)
						if err != nil {
							fmt.Print(" [ERROR]\n")
							return cli.NewExitError(err, 126)
						}
						fmt.Print(" [DONE]\n")

						break
					}

				}

			}

			return nil
		},
	}
}
