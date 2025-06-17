package cmd

import (
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/creack/pty"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/ibrahmsql/gocat/internal/logger"
	"github.com/ibrahmsql/gocat/internal/readline"
	"github.com/ibrahmsql/gocat/internal/signals"
	"github.com/ibrahmsql/gocat/internal/terminal"
)

var (
	interactive      bool
	blockSignals     bool
	localInteractive bool
	execCmd          string
)

var listenCmd = &cobra.Command{
	Use:     "listen [host] <port>",
	Aliases: []string{"l"},
	Short:   "Start a listener for incoming connections",
	Long:    `Start a TCP listener on the specified port and optionally host.`,
	Args:    cobra.RangeArgs(1, 2),
	Run:     runListen,
}

func init() {
	rootCmd.AddCommand(listenCmd)

	listenCmd.Flags().BoolVarP(&interactive, "interactive", "i", false, "Interactive mode")
	listenCmd.Flags().BoolVarP(&blockSignals, "block-signals", "b", false, "Block exit signals like CTRL-C")
	listenCmd.Flags().BoolVarP(&localInteractive, "local-interactive", "l", false, "Local interactive mode")
	listenCmd.Flags().StringVarP(&execCmd, "exec", "e", "", "Execute command when connection received")

	// Mark conflicting flags
	listenCmd.MarkFlagsMutuallyExclusive("interactive", "local-interactive")
}

func runListen(cmd *cobra.Command, args []string) {
	var host, port string

	if len(args) == 1 {
		host = "0.0.0.0"
		port = args[0]
	} else {
		host = args[0]
		port = args[1]
	}

	if err := listen(host, port); err != nil {
		logger.Fatal("Error: %v", err)
	}
}

func listen(host, port string) error {
	address := net.JoinHostPort(host, port)
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return fmt.Errorf("failed to bind to %s: %v", address, err)
	}
	defer listener.Close()

	color.Green("Listening on %s", address)

	conn, err := listener.Accept()
	if err != nil {
		return fmt.Errorf("failed to accept connection: %v", err)
	}
	defer conn.Close()

	color.Cyan("Connection received")

	if interactive {
		return handleInteractive(conn)
	} else if localInteractive {
		return handleLocalInteractive(conn)
	} else {
		return handleNormal(conn)
	}
}

func handleNormal(conn net.Conn) error {
	if blockSignals {
		signals.BlockExitSignals()
	}

	if execCmd != "" {
		if _, err := conn.Write([]byte(execCmd + "\n")); err != nil {
			return fmt.Errorf("failed to send exec command: %v", err)
		}
	}

	// Start goroutines for bidirectional communication
	go func() {
		if _, err := io.Copy(conn, os.Stdin); err != nil {
			logger.Error("stdin copy error: %v", err)
		}
		os.Exit(0)
	}()

	if _, err := io.Copy(os.Stdout, conn); err != nil {
		return fmt.Errorf("stdout copy error: %v", err)
	}

	return nil
}

func handleInteractive(conn net.Conn) error {
	if blockSignals {
		signals.BlockExitSignals()
	}

	// Setup terminal for interactive mode on Unix systems
	if runtime.GOOS != "windows" {
		if termState, err := terminal.SetupTerminal(); err == nil {
			defer termState.Restore()
		}
	}

	// Create a PTY for better shell interaction
	shell := "/bin/sh"
	if runtime.GOOS == "windows" {
		shell = "cmd.exe"
	} else if os.Getenv("SHELL") != "" {
		shell = os.Getenv("SHELL")
	}

	cmd := exec.Command(shell, "-i")
	ptmx, err := pty.Start(cmd)
	if err != nil {
		return fmt.Errorf("failed to start pty: %v", err)
	}
	defer ptmx.Close()

	// Handle PTY size changes
	go func() {
		for {
			if err := pty.InheritSize(os.Stdin, ptmx); err != nil {
				logger.Error("error resizing pty: %v", err)
			}
		}
	}()

	// Copy data between connection and PTY
	go func() {
		if _, err := io.Copy(ptmx, conn); err != nil {
			logger.Error("conn to pty copy error: %v", err)
		}
	}()

	if _, err := io.Copy(conn, ptmx); err != nil {
		return fmt.Errorf("pty to conn copy error: %v", err)
	}

	return nil
}

func handleLocalInteractive(conn net.Conn) error {
	// Start goroutine to read from connection and write to stdout
	go func() {
		if _, err := io.Copy(os.Stdout, conn); err != nil {
			logger.Error("connection read error: %v", err)
		}
	}()

	color.Cyan("Connection received")

	// Create readline editor
	editor := readline.NewEditor()
	editor.SetPrompt(">> ")

	// Readline loop
	for {
		command, err := editor.Readline()
		if err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("failed to read command: %v", err)
		}

		if strings.TrimSpace(command) == "exit" {
			break
		}

		if _, err := conn.Write([]byte(command + "\n")); err != nil {
			return fmt.Errorf("failed to send command: %v", err)
		}
	}

	return nil
}
