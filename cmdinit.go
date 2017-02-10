/*
 * Copyright 2017  Alexander Sack <asac129@gmail.com>
 */
package main

import (
	"fmt"
	"os"

	"github.com/urfave/cli"
)

var SYSTEMC_TEMPLATE string = `
{
	"#spec": "pantavisor-multi-platform@1",
	"systemc.json": {
		"linux": "",
		"initrd": [
			""	
		],
		"platforms:": [],
		"volumes": {}
	}
}`

func CommandInit() cli.Command {
	return cli.Command{
		Name:        "init",
		Aliases:     []string{"i"},
		ArgsUsage:   "",
		Usage:       "pvr'ize the working directory",
		Description: "Creates the .pvr according to default spec. Creates systemc if not exists.",
		Action: func(c *cli.Context) error {
			wd, err := os.Getwd()
			if err != nil {
				return err
			}
			err = os.Mkdir(wd+"/.pvr", 0755)
			if err != nil {
				return err
			}
			err = os.Mkdir(wd+"/.pvr/objects", 0755)

			jsonFile, err := os.OpenFile(wd+"/.pvr/json", os.O_CREATE|os.O_WRONLY, 0644)

			if err != nil {
				return err
			}

			jsonFile.Write([]byte(SYSTEMC_TEMPLATE))

			fmt.Println("pvc directory ready for use.")
			return err
		},
	}
}
