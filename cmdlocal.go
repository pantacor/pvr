//
// Copyright 2019  Pantacor Ltd.
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

// CommandLocal : pvr local command
func CommandLocal() cli.Command {
	cmd := cli.Command{
		Name:    "local",
		Aliases: []string{"lo"},
		Subcommands: []cli.Command{
			CommandLocalClone(),
			CommandLocalGet(),
			CommandLocalPost(),
			CommandLocalLogs(),
		},
		Usage:       "pvr local <subcommand> [http://]{deviceip|hostname}[:PORT][/revision]",
		Description: "pvr local experience commands to interact directly with devices without using pantahub\n1.Clone a local device\n2.Get updates from a local device\n3.Post to a local device\n4.Get logs of a local device",
	}
	return cmd
}
