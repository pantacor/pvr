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
	"github.com/urfave/cli"
)

func CommandSig() cli.Command {
	return cli.Command{
		Name:      "sig",
		Aliases:   []string{"s"},
		ArgsUsage: "",
		Usage:     "manage sig usage",
		Subcommands: []cli.Command{
			CommandSigAdd(),
			CommandSigLs(),
		},
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:   "key, k",
				EnvVar: "PVR_SIG_KEY",
				Usage:  "private key in PEM format to use for signing",
			},
			cli.StringFlag{
				Name:   "pubkey, p",
				EnvVar: "PVR_SIG_PUBKEY",
				Usage:  "pubkey in PEM format to use for signing",
			},
		},
	}
}
