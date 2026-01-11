// Package main is the entry point for the lazycfg CLI.
package main

import (
	"fmt"
	"os"

	"github.com/jmreicha/lazycfg/internal/cli"
)

// Build information set via ldflags
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	versionStr := fmt.Sprintf("%s (commit: %s, built: %s)", version, commit, date)
	rootCmd := cli.NewRootCmd(versionStr)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
