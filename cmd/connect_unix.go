//go:build !windows

package cmd

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"

	"github.com/ibrahmsql/gocat/internal/logger"
	"golang.org/x/sys/unix"
)

func connectUnix(conn net.Conn, shell string) error {
	// Handle data flow control modes
	if sendOnly || recvOnly {
		return handleDataFlowControl(conn)
	}

	// Get the file descriptor from the connection
	var fd int
	switch c := conn.(type) {
	case *net.TCPConn:
		file, err := c.File()
		if err != nil {
			return fmt.Errorf("failed to get file descriptor: %v", err)
		}
		defer func() {
			if err := file.Close(); err != nil {
				log.Printf("Error closing file: %v", err)
			}
		}()
		fd = int(file.Fd())
	case *net.UnixConn:
		file, err := c.File()
		if err != nil {
			return fmt.Errorf("failed to get file descriptor: %v", err)
		}
		defer func() {
			if err := file.Close(); err != nil {
				log.Printf("Error closing file: %v", err)
			}
		}()
		fd = int(file.Fd())
	default:
		// Fallback to pipe-based approach for other connection types
		return connectUnixPipes(conn, shell)
	}

	// Duplicate file descriptor for stdin, stdout, stderr
	if err := unix.Dup2(fd, int(os.Stdin.Fd())); err != nil {
		return fmt.Errorf("failed to dup stdin: %v", err)
	}
	if err := unix.Dup2(fd, int(os.Stdout.Fd())); err != nil {
		return fmt.Errorf("failed to dup stdout: %v", err)
	}
	if err := unix.Dup2(fd, int(os.Stderr.Fd())); err != nil {
		return fmt.Errorf("failed to dup stderr: %v", err)
	}

	// Create shell command
	cmd := exec.Command(shell, "-i")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

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

// Fallback method using pipes for connection types that don't support File()
func connectUnixPipes(conn net.Conn, shell string) error {
	// Handle data flow control modes
	if sendOnly || recvOnly {
		return handleDataFlowControl(conn)
	}

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
		logger.Info("Shell exited normally")
	}

	return nil
}
