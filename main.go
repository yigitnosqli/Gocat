package main

import (
	"fmt"
	"os"

	"github.com/ibrahmsql/gocat/cmd"
)

// Build information (set by ldflags)
var (
	version   = "dev"
	buildTime = "unknown"
	gitCommit = "unknown"
	gitBranch = "unknown"
	builtBy   = "unknown"
)

// main sets build metadata for the command package, runs the root command, and exits with status 1 if command execution returns an error.
// It propagates ldflags-provided build information (version, buildTime, gitCommit, gitBranch, builtBy) to cmd before invoking cmd.Execute().
func main() {
	// Set build information for cmd package
	cmd.SetBuildInfo(version, buildTime, gitCommit, gitBranch, builtBy)

	// Execute the root command
	if err := cmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}