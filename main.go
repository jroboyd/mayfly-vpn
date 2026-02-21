package main

import (
	"os"

	"github.com/jamesboyd/mayfly/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
