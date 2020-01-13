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
	"fmt"

	"github.com/urfave/cli"
)

func CommandLogsDeprecated() cli.Command {
	return cli.Command{
		Name:        "logs",
		Aliases:     []string{"log"},
		Usage:       "Get logs for your devices (early preview)",
		Description: "Get streaming logs of devices you own from pantahub",
		Before: func(c *cli.Context) error {
			fmt.Print("\nDEPRECATED: the pvr logs command is deprecated and will go away in some future release. It can now be found as a device subcommand:pvr device logs\n")
			return nil
		},
		Action: CommandLogs().Action,
	}
}
