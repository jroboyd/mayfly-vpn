package main

import (
	"os"

	"github.com/jamesboyd/mayfly/cmd"
	"github.com/joho/godotenv"
)

func main() {
	// Load .env if present (silently ignored if missing).
	godotenv.Load()

	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
