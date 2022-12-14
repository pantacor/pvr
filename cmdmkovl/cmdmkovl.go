//
// Copyright 2022  Pantacor Ltd.
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
	"os"
	"os/exec"

	"gitlab.com/pantacor/pvr/libpvr"
)

func main() {

	if len(os.Args) < 3 {
		fmt.Println("ERROR: must have A and B dir as cli argument")
		os.Exit(2)
	}

	diff := libpvr.MkTreeDiff(os.Args[1], os.Args[2])
	diff.MkOvl(os.Args[3])
	cmd := exec.Command("find", os.Args[3])
	cmd.Run()
}
