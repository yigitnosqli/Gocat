package cmd

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "gocat",
	Short: "A netcat-like tool written in Go",
	Long: `Gocat is a netcat-like tool written in Go that provides network connectivity.
It can be used for port scanning, file transfers, backdoors, port redirection,
and many other networking tasks.`,
	Version: "2.0.1",
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.CompletionOptions.DisableDefaultCmd = true
}
