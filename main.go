package main

import (
	"os"

	"github.com/cozy-creator/cozy-cli/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
