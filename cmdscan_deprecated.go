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
	"fmt"

	"github.com/urfave/cli"
)

func CommandScanDeprecated() cli.Command {
	return cli.Command{
		Name:      "scan",
		ArgsUsage: "",
		Usage:     "Scan for pantavisor devices announcing themselves through MDNS on local network.",
		Before: func(c *cli.Context) error {
			fmt.Print("\nDEPRECATED: the pvr scan command is deprecated and will go away in some future release. It can now be found as a device subcommand:pvr device scan\n")
			return nil
		},
		Action: CommandScan().Action,
	}
}
