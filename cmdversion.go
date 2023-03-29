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
	"runtime/debug"
	"strings"

	"github.com/urfave/cli"
)

func CommandVersion() cli.Command {
	return cli.Command{
		Name:        "version",
		Usage:       "pvr version",
		Description: "Get pvr version",
		Action: func(c *cli.Context) error {
			fmt.Println(c.App.Version)
			bi, ok := debug.ReadBuildInfo()
			if !ok {
				return nil
			}

			fmt.Printf("  go version: %s\n", strings.ReplaceAll(bi.GoVersion, "go", ""))

			return nil
		},
	}
}
