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
	"github.com/urfave/cli"
)

const (
	defaultCertsDownloadUrl = "https://gitlab.com/pantacor/pv-developer-ca/-/raw/master/pvs/pvs.defaultkeys.tar.gz?inline=false"
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
				Usage:  "path to cert chain to include in jws x5c header. Use 'no' to not include any.",
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
			},
			cli.StringFlag{
				Name:   "certs-url",
				Usage:  "Use `PVS_CERTS_URL` for downloading the pvs certificates as a tarball.",
				EnvVar: "PVS_CERTS_URL",
				Value:  defaultCertsDownloadUrl,
			},
			cli.BoolFlag{
				Name:   "with-payload",
				Usage:  "Use `PVS_SIG_WITH_PAYLOAD` for including full payload in jose serialization.",
				EnvVar: "PVS_SIG_WITH_PAYLOAD",
			},
			cli.StringFlag{
				Name:   "output",
				Usage:  "Use `PVS_SIG_OUTPUT` output of the signature file to the file given. '-' will print to stdout.",
				EnvVar: "PVS_SIG_OUTPUT",
			},
		},
		Before: func(c *cli.Context) error {
			if c.GlobalString("certs-url") != "" {
				c.App.Metadata["PVS_CERTS_URL"] = c.GlobalString("certs-url")
			} else {
				c.App.Metadata["PVS_CERTS_URL"] = defaultCertsDownloadUrl
			}
			return nil
		},
	}
}
