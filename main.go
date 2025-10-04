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

func main() {
	// Set build information for cmd package
	cmd.SetBuildInfo(version, buildTime, gitCommit, gitBranch, builtBy)

	// Execute the root command
	if err := cmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
