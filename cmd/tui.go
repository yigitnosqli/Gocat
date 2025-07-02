package cmd

import (
	"fmt"
	"os"

	"github.com/ibrahmsql/gocat/internal/ui"
	"github.com/spf13/cobra"
)

// tuiCmd represents the tui command
var tuiCmd = &cobra.Command{
	Use:   "tui [mode]",
	Short: "Start the interactive terminal user interface",
	Long: `Start GoCat's beautiful terminal user interface (TUI).

The TUI provides an interactive way to use all GoCat features:
- Connect to remote hosts
- Listen for incoming connections
- Real-time chat communication
- Network traffic brokering
- Comprehensive port scanning

Optional mode argument can be one of:
- connect: Start directly in connect mode
- listen:  Start directly in listen mode
- chat:    Start directly in chat mode
- broker:  Start directly in broker mode
- scan:    Start directly in scan mode
- help:    Start directly in help mode

If no mode is specified, the main menu will be displayed.

Examples:
  gocat tui              # Start with main menu
  gocat tui connect      # Start directly in connect mode
  gocat tui scan         # Start directly in scan mode`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		// Check terminal support
		if err := ui.CheckTerminalSupport(); err != nil {
			fmt.Fprintf(os.Stderr, "Terminal compatibility error: %v\n", err)
			fmt.Fprintf(os.Stderr, "Please ensure you're running in a compatible terminal with minimum 80x24 size.\n")
			os.Exit(1)
		}

		// Start TUI with optional mode argument
		if err := ui.RunTUIWithArgs(args); err != nil {
			fmt.Fprintf(os.Stderr, "Error starting TUI: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	// Add tui command to root
	rootCmd.AddCommand(tuiCmd)
}