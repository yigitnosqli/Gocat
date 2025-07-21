//go:build windows

package cmd

import (
	"fmt"
	"net"
	"os/exec"

	"github.com/ibrahmsql/gocat/internal/logger"
)

func connectUnix(conn net.Conn, shell string) error {
	// On Windows, we use the pipe-based approach
	return connectUnixPipes(conn, shell)
}

// connectUnixPipes is the fallback method using pipes for Windows
func connectUnixPipes(conn net.Conn, shell string) error {
	// Handle data flow control modes
	if sendOnly || recvOnly {
		return handleDataFlowControl(conn)
	}

	// Create shell command
	cmd := exec.Command(shell)

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
		logger.Info("Shell exited normally")
	}

	return nil
}