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
			CommandSigUpdate(),
			CommandSigLs(),
		},
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:   "key, k",
				EnvVar: "PVR_SIG_KEY",
				Usage:  "private key in PEM format to use for signing",
			},
			cli.StringFlag{
				Name:   "x5c, x",
				EnvVar: "PVR_X5C_PATH",
				Usage:  "path to cert chain to include in jws x5c header. Note: we will not validate that the actual signature can be validated with this one.",
			},
			cli.StringFlag{
				Name:   "pubkey, p",
				EnvVar: "PVR_SIG_PUBKEY",
				Usage:  "use specific pubkey store to validate signatures.",
			},
			cli.StringFlag{
				Name:   "cacerts, c",
				EnvVar: "PVR_SIG_CACERTS",
				Usage:  "initialize cert pool from file or directory provided in this argument. use __system__ to use system cert store",
				Value:  "_system_",
			},
		},
	}
}
