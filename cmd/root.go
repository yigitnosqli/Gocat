package cmd

import (
	"context"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "gocat",
	Short: "A modern netcat-like tool written in Go",
	Long: `Gocat is a modern, fast, and secure netcat-like tool written in Go.
It provides network connectivity with enhanced features including:
- SSL/TLS support
- Proxy support (SOCKS5, HTTP)
- Interactive and non-interactive modes
- IPv4/IPv6 support
- Connection retry with exponential backoff
- Comprehensive logging

Perfect for port scanning, file transfers, reverse shells, port redirection,
and many other networking tasks.`,
	Version: "2.1.0",
	Example: `  # Listen on port 8080
  gocat listen 8080

  # Connect to a host
  gocat connect example.com 8080

  # Interactive mode with signal blocking
  gocat listen -ib 8080

  # Connect with SSL
  gocat connect -S example.com 443

  # Use SOCKS5 proxy
  gocat connect -p socks5://127.0.0.1:1080 example.com 8080`,
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
}
