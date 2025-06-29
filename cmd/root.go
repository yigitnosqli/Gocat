package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/ibrahmsql/gocat/internal/logger"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "gocat",
	Short: "A modern netcat-like tool written in Go",
	Long: `Gocat is a netcat-like tool written in Go that provides network connectivity.
It can be used for port scanning, file transfers, backdoors, port redirection,
and many other networking tasks.
`,
}

func Execute() error {
	return rootCmd.Execute()
}

func ExecuteContext(ctx context.Context) error {
	return rootCmd.ExecuteContext(ctx)
}

func init() {
	rootCmd.CompletionOptions.DisableDefaultCmd = true

	// Add global flags
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "Enable verbose output")
	rootCmd.PersistentFlags().BoolP("quiet", "q", false, "Suppress output")
	rootCmd.PersistentFlags().String("theme", "", "Path to color theme file (default: ~/.gocat-theme.yml)")

	// Initialize theme on startup
	cobra.OnInitialize(initTheme)
}

// initTheme loads the color theme
func initTheme() {
	themePath, _ := rootCmd.PersistentFlags().GetString("theme")
	if err := logger.LoadTheme(themePath); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to load theme: %v\n", err)
	}
}
