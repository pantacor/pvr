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
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/urfave/cli"
	"gitlab.com/pantacor/pvr/libpvr"
)

func uniqueSlice(stringSlice []string) []string {
	var unique []string
	var stack *string
	for _, v := range stringSlice {
		// if we have nothing on stack, we will add this one
		if stack == nil {
			goto append
		}
		// if the stack element is the same; we skip
		if v == *stack {
			continue
		}

	append:
		// here we move forward
		v1 := v
		stack = &v1
		unique = append(unique, v)
	}

	return unique
}

func mergeSummary(dest *libpvr.JwsVerifySummary, merge ...libpvr.JwsVerifySummary) {
	for _, m := range merge {
		dest.Excluded = append(dest.Excluded, m.Excluded...)
		dest.Protected = append(dest.Protected, m.Protected...)
		dest.NotSeen = append(dest.NotSeen, m.NotSeen...)
		dest.FullJSONWebSigs = append(dest.FullJSONWebSigs, m.FullJSONWebSigs...)
	}
	sort.Strings(dest.Excluded)
	sort.Strings(dest.NotSeen)
	sort.Strings(dest.Protected)
	dest.Protected = uniqueSlice(dest.Protected)
	excluded := uniqueSlice(dest.Excluded)
	dest.Excluded = nil
	for _, v := range excluded {
		i := sort.SearchStrings(dest.Protected, v)
		if i < len(dest.Protected) && dest.Protected[i] == v {
			continue
		}
		dest.Excluded = append(dest.Excluded, v)
	}
	notseen := uniqueSlice(dest.NotSeen)
	dest.NotSeen = nil
	for _, v := range notseen {
		i := sort.SearchStrings(dest.Protected, v)
		if i < len(dest.Protected) && dest.Protected[i] == v {
			continue
		}
		i = sort.SearchStrings(dest.Excluded, v)
		if i < len(dest.Excluded) && dest.Excluded[i] == v {
			continue
		}
		dest.NotSeen = append(dest.NotSeen, v)
	}
}

func CommandSigLs() cli.Command {
	return cli.Command{
		Name:      "ls",
		ArgsUsage: "[<NAME>] ...",
		Usage:     "verify and list content protected by _sigs/<NAME>.json for all arguments; by default it matches *",
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

			pubkey := c.Parent().String("pubkey")
			if pubkey != "" {
				keyFs, err := os.Stat(pubkey)

				if err != nil {
					return cli.NewExitError(("ERROR: errors accessing signing key; see --help: " + err.Error()), 11)
				}

				if keyFs.IsDir() {
					return cli.NewExitError(("ERROR: signing key is not a file; see --help: " + err.Error()), 12)
				}
			}

			_args := c.Args()
			var args []string
			if len(_args) == 0 {
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

					args = append(args, k)
				}
			} else {
				for _, v := range _args {
					// if we have a full path we use it as absolute pvs file
					if strings.HasSuffix(v, ".json") {
						args = append(args, v)
					} else {
						args = append(args, "_sigs/"+v+".json")
					}
				}
			}

			var resultSummary libpvr.JwsVerifySummary
			var verifySummary []libpvr.JwsVerifySummary

			cacerts := c.Parent().String("cacerts")

			if pubkey == "" && cacerts == "" {
				cacerts, err = libpvr.GetFromConfigPvs(
					c.App.Metadata["PVS_CERTS_URL"].(string),
					pvr.Session.GetConfigDir(),
					libpvr.SigCacertFilename,
				)
				if err != nil {
					return cli.NewExitError(err, 127)
				}
			}

			for _, v := range args {
				w, err := pvr.JwsVerifyPvs(pubkey, cacerts, v, c.Parent().Bool("with-payload"))
				if errors.Is(err, os.ErrNotExist) {
					return cli.NewExitError("ERROR: signature file does not exist with name "+v, 125)
				}
				if err != nil {
					return cli.NewExitError(err, 13)
				}
				verifySummary = append(verifySummary, *w)
			}

			mergeSummary(&resultSummary, verifySummary...)

			if !c.Bool("with-sigs") {
				resultSummary.FullJSONWebSigs = nil
			}
			jsonBuf, err := json.MarshalIndent(resultSummary, "", "    ")

			if err != nil {
				return cli.NewExitError(err, 13)
			}

			fmt.Println(string(jsonBuf))

			return nil
		},
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:   "part, p",
				Usage:  "select elements of part",
				EnvVar: "PVR_SIG_ADD_PART",
			},
			cli.BoolFlag{
				Name:   "with-sigs, s",
				Usage:  "Show full json web signatures in summary display",
				EnvVar: "PVR_SIG_LS_WITH_SIGS",
			},
		},
	}
}
