package main

import (
	"os"

	"github.com/cozy-creator/cozyctl/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
