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

// CommandApp : pvr app command
func CommandApp() cli.Command {
	cmd := cli.Command{
		Name:    "app",
		Aliases: []string{"ap"},
		Subcommands: []cli.Command{
			CommandAppAdd(),
			CommandAppList(),
			CommandAppInfo(),
			CommandAppRemove(),
			CommandAppInstall(),
			CommandAppUpdate(),
		},
		Usage:       "pvr app ls :list applications in pvr checkout,pvr app info <appname> output info and state of appname ,pvr app rm <appname> : remove app from pvr checkout",
		Description: "\n1.Add new application\n2.List applications in pvr checkout\n3.Output info and state of appname\n4.Remove app from pvr checkout\n5.Install Application\n6.Update Application",
	}
	return cmd
}
