package main

import (
	"fmt"
	"os"
	"runtime"

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

// showVersion displays version information
func showVersion() {
	fmt.Printf("GoCat %s\n", version)
	fmt.Printf("\nBuild Information:\n")
	fmt.Printf("  Version:     %s\n", version)
	fmt.Printf("  Git Commit:  %s\n", gitCommit)
	fmt.Printf("  Git Branch:  %s\n", gitBranch)
	fmt.Printf("  Build Time:  %s\n", buildTime)
	fmt.Printf("  Built By:    %s\n", builtBy)
	fmt.Printf("\nRuntime Information:\n")
	fmt.Printf("  Go Version:  %s\n", runtime.Version())
	fmt.Printf("  OS/Arch:     %s/%s\n", runtime.GOOS, runtime.GOARCH)
	fmt.Printf("  CPUs:        %d\n", runtime.NumCPU())
}

}

