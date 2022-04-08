package main

import (
	"fmt"
	"os"

	"gitlab.com/pantacor/pvr/libpvr"
)

func main() {

	if len(os.Args) < 2 {
		fmt.Println("ERROR: must have A and B dir as cli argument")
		os.Exit(2)
	}

	diff := libpvr.MkTreeDiff(os.Args[1], os.Args[2])

	fmt.Println(diff)
}
