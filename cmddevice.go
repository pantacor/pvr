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

// CommandDevice : pvr device command
func CommandDevice() cli.Command {
	cmd := cli.Command{
		Name:    "device",
		Aliases: []string{"dev"},
		Subcommands: []cli.Command{
			CommandPs(),
			CommandLogs(),
			CommandScan(),
			CommandDeviceCreate(),
			CommandDeviceGet(),
			CommandDeviceSet(),
		},
		Usage:       "pvr device <ps|logs|scan>",
		Description: "\n1.Show Owned Devices\n 2.Get logs for your devices (early preview)\n 3.Scan for pantavisor devices announcing themselves through MDNS on local network.\n4.Create new device\n5.Set device user-meta|device-meta fields (Note:If you are logged in as USER then you can update user-meta field but if you are logged in as DEVICE then you can update device-meta field)",
	}
	return cmd
}
