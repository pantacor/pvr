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
	"os"
	"path/filepath"

	"github.com/urfave/cli"
)

func CommandCompletion() cli.Command {
	return cli.Command{
		Name:    "completion",
		Aliases: []string{"com"},
		Usage:   "pvr completion",
		Description: `Get the bash completion file contents by running "pvr completion"

Set up instructions for Ubuntu/Linux:
-------------------------------------

Step 1: Install bash-completion

	  sudo apt-get install bash-completion

Step 2: Add to ~/.bashrc the following line:
		
	  source <(pvr completion)
		`,
		Action: func(c *cli.Context) error {
			cmd := filepath.Base(os.Args[0])

			contents := `#!/bin/bash

_cli_bash_autocomplete() {
if [[ "${COMP_WORDS[0]}" != "source" ]]; then

	local cur opts base
	COMPREPLY=()

	if ((${#COMP_WORDS[@]}>=3)); then
		TEMP_COMP_WORDS=${COMP_WORDS[@]}
		COMP_WORDS=()
		for word in ${TEMP_COMP_WORDS[@]}; do
			if [[ $word =~ ^--.* || $word == "http" || $word == "https" || $word =~ ^=.* || $word =~ ^:.* ||  $word =~ ^//.* || $word =~ ^[0-9]+$   ]];
				then
					continue
				else
					COMP_WORDS+=( ${word} )
			fi
		done

		COMP_CWORD=${#COMP_WORDS[@]}
	fi

	cur="${COMP_WORDS[COMP_CWORD]}"

	if [[ "$cur" == "-"* ]]; 
		then
			opts=$( ${COMP_WORDS[@]:0:$COMP_CWORD} ${cur} --generate-bash-completion )
	    else
			opts=$( ${COMP_WORDS[@]:0:$COMP_CWORD} --generate-bash-completion )
	fi

	COMPREPLY=( $(compgen -W "${opts} " -- ${cur}) )

	if [ "${COMPREPLY}" == "${COMP_WORDS[-1]}" ];then
		COMPREPLY=""
	fi

	if [ "${COMPREPLY: -1}" != "/" ] && [ "${COMPREPLY}" != "" ] && [ "${COMPREPLY: -1}" != " " ];then
		COMPREPLY="${COMPREPLY} "
	fi
	return 0
fi
}

complete -o bashdefault -o default -o nospace -F _cli_bash_autocomplete ` + cmd
			fmt.Print(string(contents))
			fmt.Print("\n\n")
			return nil
		},
	}
}