package cmd

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/ibrahmsql/gocat/internal/logger"
	"github.com/spf13/cobra"
)

var (
	unixSocketPath    string
	unixSocketType    string // "stream" or "datagram"
	unixRemoveExisting bool
	unixPermissions   uint32
	unixBufferSize    int
)

// unixCmd represents the Unix domain socket command
var unixCmd = &cobra.Command{
	Use:     "unix",
	Aliases: []string{"uds"},
	Short:   "Unix domain socket operations",
	Long: `Unix domain socket server and client for local IPC.

Unix domain sockets provide fast, secure inter-process communication
on the same machine. They're commonly used in containerized environments.

Examples:
  # Listen on Unix socket
  gocat unix listen /tmp/gocat.sock

  # Connect to Unix socket
  gocat unix connect /tmp/gocat.sock

  # Echo server on Unix socket
  gocat unix echo /tmp/echo.sock

  # Datagram socket
  gocat unix listen --type datagram /tmp/gocat.sock`,
}

// unixListenCmd handles Unix socket server
var unixListenCmd = &cobra.Command{
	Use:   "listen [socket-path]",
	Short: "Listen on a Unix domain socket",
	Long: `Start a Unix domain socket server that accepts connections.

The server will create a socket file at the specified path and accept
connections, relaying data between stdin/stdout and the socket.`,
	Args: cobra.ExactArgs(1),
	RunE: runUnixListen,
}

// unixConnectCmd handles Unix socket client
var unixConnectCmd = &cobra.Command{
	Use:     "connect [socket-path]",
	Aliases: []string{"c"},
	Short:   "Connect to a Unix domain socket",
	Long: `Connect to a Unix domain socket and relay data between stdin/stdout.

Examples:
  gocat unix connect /tmp/gocat.sock
  echo "Hello" | gocat unix connect /tmp/gocat.sock`,
	Args: cobra.ExactArgs(1),
	RunE: runUnixConnect,
}

// unixEchoCmd handles Unix socket echo server
var unixEchoCmd = &cobra.Command{
	Use:   "echo [socket-path]",
	Short: "Start a Unix socket echo server",
	Long:  `Start a Unix domain socket echo server that echoes back all received data.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runUnixEcho,
}

func init() {
	rootCmd.AddCommand(unixCmd)
	unixCmd.AddCommand(unixListenCmd)
	unixCmd.AddCommand(unixConnectCmd)
	unixCmd.AddCommand(unixEchoCmd)

	// Listen flags
	unixListenCmd.Flags().StringVar(&unixSocketType, "type", "stream", "Socket type (stream or datagram)")
	unixListenCmd.Flags().BoolVar(&unixRemoveExisting, "remove", true, "Remove existing socket file")
	unixListenCmd.Flags().Uint32Var(&unixPermissions, "permissions", 0660, "Socket file permissions")
	unixListenCmd.Flags().IntVar(&unixBufferSize, "buffer", 8192, "Buffer size for I/O operations")

	// Connect flags
	unixConnectCmd.Flags().StringVar(&unixSocketType, "type", "stream", "Socket type (stream or datagram)")
	unixConnectCmd.Flags().IntVar(&unixBufferSize, "buffer", 8192, "Buffer size for I/O operations")

	// Echo flags
	unixEchoCmd.Flags().BoolVar(&unixRemoveExisting, "remove", true, "Remove existing socket file")
	unixEchoCmd.Flags().Uint32Var(&unixPermissions, "permissions", 0660, "Socket file permissions")
}

func runUnixListen(cmd *cobra.Command, args []string) error {
	socketPath := args[0]
	
	// Validate socket path
	if err := validateSocketPath(socketPath); err != nil {
		return err
	}

	// Remove existing socket if requested
	if unixRemoveExisting {
		if err := removeSocketIfExists(socketPath); err != nil {
			return err
		}
	}

	// Determine network type
	network := "unix"
	if unixSocketType == "datagram" {
		network = "unixgram"
	}

	logger.Info("Starting Unix socket server: %s (type: %s)", socketPath, network)

	// Create listener
	listener, err := net.Listen(network, socketPath)
	if err != nil {
		return fmt.Errorf("failed to create listener: %w", err)
	}
	defer listener.Close()

	// Set permissions
	if err := os.Chmod(socketPath, os.FileMode(unixPermissions)); err != nil {
		logger.Warn("Failed to set socket permissions: %v", err)
	}

	logger.Info("Unix socket listening on %s", socketPath)

	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Accept connections
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				select {
				case <-sigChan:
					return
				default:
					logger.Error("Accept error: %v", err)
					continue
				}
			}

			logger.Info("Connection accepted")
			go handleUnixConnection(conn)
		}
	}()

	// Wait for interrupt
	<-sigChan
	logger.Info("Shutting down Unix socket server...")
	
	// Cleanup socket file
	os.Remove(socketPath)
	
	return nil
}

func runUnixConnect(cmd *cobra.Command, args []string) error {
	socketPath := args[0]

	// Determine network type
	network := "unix"
	if unixSocketType == "datagram" {
		network = "unixgram"
	}

	logger.Info("Connecting to Unix socket: %s (type: %s)", socketPath, network)

	// Connect to socket
	conn, err := net.Dial(network, socketPath)
	if err != nil {
		return fmt.Errorf("connection failed: %w", err)
	}
	defer conn.Close()

	logger.Info("Connected to Unix socket")

	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Read from socket, write to stdout
	go func() {
		buffer := make([]byte, unixBufferSize)
		for {
			n, err := conn.Read(buffer)
			if err != nil {
				if err != io.EOF {
					logger.Error("Socket read error: %v", err)
				}
				cancel()
				return
			}

			if n > 0 {
				if _, err := os.Stdout.Write(buffer[:n]); err != nil {
					logger.Error("Stdout write error: %v", err)
					cancel()
					return
				}
			}
		}
	}()

	// Read from stdin, write to socket
	go func() {
		buffer := make([]byte, unixBufferSize)
		for {
			n, err := os.Stdin.Read(buffer)
			if err != nil {
				if err != io.EOF {
					logger.Error("Stdin read error: %v", err)
				}
				cancel()
				return
			}

			if n > 0 {
				if _, err := conn.Write(buffer[:n]); err != nil {
					logger.Error("Socket write error: %v", err)
					cancel()
					return
				}
			}
		}
	}()

	// Wait for interrupt or context cancellation
	select {
	case <-sigChan:
		logger.Info("Interrupted, closing connection...")
	case <-ctx.Done():
	}

	return nil
}

func runUnixEcho(cmd *cobra.Command, args []string) error {
	socketPath := args[0]

	// Validate and cleanup
	if err := validateSocketPath(socketPath); err != nil {
		return err
	}

	if unixRemoveExisting {
		if err := removeSocketIfExists(socketPath); err != nil {
			return err
		}
	}

	logger.Info("Starting Unix socket echo server: %s", socketPath)

	// Create listener
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		return fmt.Errorf("failed to create listener: %w", err)
	}
	defer listener.Close()

	// Set permissions
	if err := os.Chmod(socketPath, os.FileMode(unixPermissions)); err != nil {
		logger.Warn("Failed to set socket permissions: %v", err)
	}

	logger.Info("Unix echo server listening on %s", socketPath)

	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Accept connections
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				select {
				case <-sigChan:
					return
				default:
					logger.Error("Accept error: %v", err)
					continue
				}
			}

			logger.Info("Echo connection accepted")
			go handleUnixEcho(conn)
		}
	}()

	// Wait for interrupt
	<-sigChan
	logger.Info("Shutting down Unix echo server...")
	
	// Cleanup
	os.Remove(socketPath)
	
	return nil
}

// Helper functions

func handleUnixConnection(conn net.Conn) {
	defer conn.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Read from connection, write to stdout
	go func() {
		buffer := make([]byte, unixBufferSize)
		for {
			n, err := conn.Read(buffer)
			if err != nil {
				if err != io.EOF {
					logger.Error("Connection read error: %v", err)
				}
				cancel()
				return
			}

			if n > 0 {
				if _, err := os.Stdout.Write(buffer[:n]); err != nil {
					logger.Error("Stdout write error: %v", err)
					cancel()
					return
				}
			}
		}
	}()

	// Read from stdin, write to connection
	go func() {
		buffer := make([]byte, unixBufferSize)
		for {
			n, err := os.Stdin.Read(buffer)
			if err != nil {
				if err != io.EOF {
					logger.Error("Stdin read error: %v", err)
				}
				cancel()
				return
			}

			if n > 0 {
				if _, err := conn.Write(buffer[:n]); err != nil {
					logger.Error("Connection write error: %v", err)
					cancel()
					return
				}
			}
		}
	}()

	<-ctx.Done()
	logger.Info("Connection closed")
}

func handleUnixEcho(conn net.Conn) {
	defer conn.Close()

	buffer := make([]byte, unixBufferSize)
	for {
		n, err := conn.Read(buffer)
		if err != nil {
			if err != io.EOF {
				logger.Error("Read error: %v", err)
			}
			break
		}

		if n > 0 {
			logger.Debug("Echoing %d bytes", n)
			if _, err := conn.Write(buffer[:n]); err != nil {
				logger.Error("Write error: %v", err)
				break
			}
		}
	}

	logger.Info("Echo connection closed")
}

func validateSocketPath(path string) error {
	// Check if path is absolute
	if !filepath.IsAbs(path) {
		return fmt.Errorf("socket path must be absolute: %s", path)
	}

	// Check if directory exists
	dir := filepath.Dir(path)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return fmt.Errorf("directory does not exist: %s", dir)
	}

	// Check path length (Unix sockets have a limit)
	if len(path) > 108 { // Standard Unix socket path limit
		return fmt.Errorf("socket path too long (max 108 characters): %s", path)
	}

	return nil
}

func removeSocketIfExists(path string) error {
	// Check if file exists
	if _, err := os.Stat(path); err == nil {
		// Check if it's a socket
		if fi, err := os.Lstat(path); err == nil {
			if fi.Mode()&os.ModeSocket != 0 {
				logger.Debug("Removing existing socket: %s", path)
				if err := os.Remove(path); err != nil {
					return fmt.Errorf("failed to remove existing socket: %w", err)
				}
			} else {
				return fmt.Errorf("path exists but is not a socket: %s", path)
			}
		}
	}
	return nil
}
