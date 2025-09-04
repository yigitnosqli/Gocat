package network

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/ibrahmsql/gocat/internal/logger"
)

// UnixSocketConfig holds configuration for Unix domain sockets
type UnixSocketConfig struct {
	Path        string
	Permissions os.FileMode
	Cleanup     bool // Remove socket file on close
	Timeout     time.Duration
}

// DefaultUnixSocketConfig returns default configuration for Unix sockets
func DefaultUnixSocketConfig() *UnixSocketConfig {
	return &UnixSocketConfig{
		Permissions: 0666,
		Cleanup:     true,
		Timeout:     30 * time.Second,
	}
}

// UnixDialer provides Unix domain socket dialing functionality
type UnixDialer struct {
	config *UnixSocketConfig
}

// NewUnixDialer creates a new Unix socket dialer
func NewUnixDialer(config *UnixSocketConfig) *UnixDialer {
	if config == nil {
		config = DefaultUnixSocketConfig()
	}
	return &UnixDialer{config: config}
}

// Dial connects to a Unix domain socket
func (d *UnixDialer) Dial(socketPath string) (net.Conn, error) {
	if !isValidUnixSocketPath(socketPath) {
		return nil, fmt.Errorf("invalid Unix socket path: %s", socketPath)
	}

	// Check if socket file exists
	if _, err := os.Stat(socketPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("Unix socket does not exist: %s", socketPath)
	}

	logger.Debug("Connecting to Unix socket: %s", socketPath)

	// Create Unix address
	addr, err := net.ResolveUnixAddr("unix", socketPath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve Unix address: %w", err)
	}

	// Dial with timeout
	conn, err := net.DialTimeout("unix", addr.String(), d.config.Timeout)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Unix socket: %w", err)
	}

	logger.Info("Connected to Unix socket: %s", socketPath)
	return conn, nil
}

// UnixListener provides Unix domain socket listening functionality
type UnixListener struct {
	config   *UnixSocketConfig
	listener net.Listener
	path     string
}

// NewUnixListener creates a new Unix socket listener
func NewUnixListener(config *UnixSocketConfig) *UnixListener {
	if config == nil {
		config = DefaultUnixSocketConfig()
	}
	return &UnixListener{config: config}
}

// Listen starts listening on a Unix domain socket
func (l *UnixListener) Listen(socketPath string) error {
	if !isValidUnixSocketPath(socketPath) {
		return fmt.Errorf("invalid Unix socket path: %s", socketPath)
	}

	l.path = socketPath

	// Remove existing socket file if it exists
	if err := l.removeExistingSocket(socketPath); err != nil {
		return fmt.Errorf("failed to remove existing socket: %w", err)
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(socketPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create socket directory: %w", err)
	}

	logger.Debug("Creating Unix socket listener: %s", socketPath)

	// Create Unix address
	addr, err := net.ResolveUnixAddr("unix", socketPath)
	if err != nil {
		return fmt.Errorf("failed to resolve Unix address: %w", err)
	}

	// Start listening
	listener, err := net.ListenUnix("unix", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on Unix socket: %w", err)
	}

	l.listener = listener

	// Set socket permissions
	if err := os.Chmod(socketPath, l.config.Permissions); err != nil {
		logger.Warn("Failed to set socket permissions: %v", err)
	}

	logger.Info("Listening on Unix socket: %s", socketPath)
	return nil
}

// Accept accepts a connection on the Unix socket
func (l *UnixListener) Accept() (net.Conn, error) {
	if l.listener == nil {
		return nil, fmt.Errorf("listener not initialized")
	}
	return l.listener.Accept()
}

// Close closes the Unix socket listener
func (l *UnixListener) Close() error {
	var err error
	if l.listener != nil {
		err = l.listener.Close()
	}

	// Clean up socket file if configured
	if l.config.Cleanup && l.path != "" {
		if removeErr := os.Remove(l.path); removeErr != nil && !os.IsNotExist(removeErr) {
			logger.Warn("Failed to remove socket file: %v", removeErr)
		}
	}

	return err
}

// Addr returns the listener's network address
func (l *UnixListener) Addr() net.Addr {
	if l.listener == nil {
		return nil
	}
	return l.listener.Addr()
}

// removeExistingSocket removes an existing socket file if it exists
func (l *UnixListener) removeExistingSocket(socketPath string) error {
	stat, err := os.Stat(socketPath)
	if os.IsNotExist(err) {
		return nil // Socket doesn't exist, nothing to remove
	}
	if err != nil {
		return fmt.Errorf("failed to stat socket file: %w", err)
	}

	// Check if it's a socket
	if stat.Mode()&os.ModeSocket == 0 {
		return fmt.Errorf("file exists but is not a socket: %s", socketPath)
	}

	// Try to connect to see if socket is in use
	if l.isSocketInUse(socketPath) {
		return fmt.Errorf("socket is already in use: %s", socketPath)
	}

	// Remove the socket file
	if err := os.Remove(socketPath); err != nil {
		return fmt.Errorf("failed to remove socket file: %w", err)
	}

	logger.Debug("Removed existing socket file: %s", socketPath)
	return nil
}

// isSocketInUse checks if a Unix socket is currently in use
func (l *UnixListener) isSocketInUse(socketPath string) bool {
	conn, err := net.DialTimeout("unix", socketPath, 100*time.Millisecond)
	if err != nil {
		return false // Socket is not in use
	}
	conn.Close()
	return true // Socket is in use
}

// isValidUnixSocketPath validates a Unix socket path
func isValidUnixSocketPath(path string) bool {
	if path == "" {
		return false
	}

	// Check path length (Unix socket paths have a limit)
	if len(path) > 104 { // Typical limit on most systems
		return false
	}

	// Path should be absolute or relative
	if !filepath.IsAbs(path) && !strings.HasPrefix(path, "./") && !strings.HasPrefix(path, "../") {
		// Allow simple relative paths
		if !strings.Contains(path, "/") {
			return true
		}
		return false
	}

	return true
}

// GetUnixSocketInfo returns information about a Unix socket
func GetUnixSocketInfo(socketPath string) (*UnixSocketInfo, error) {
	stat, err := os.Stat(socketPath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat socket: %w", err)
	}

	info := &UnixSocketInfo{
		Path:        socketPath,
		Permissions: stat.Mode(),
		Size:        stat.Size(),
		ModTime:     stat.ModTime(),
		IsSocket:    stat.Mode()&os.ModeSocket != 0,
	}

	// Get additional system info if available
	if sysInfo, ok := stat.Sys().(*syscall.Stat_t); ok {
		info.UID = sysInfo.Uid
		info.GID = sysInfo.Gid
		info.Inode = sysInfo.Ino
	}

	// Check if socket is in use
	conn, err := net.DialTimeout("unix", socketPath, 100*time.Millisecond)
	if err == nil {
		conn.Close()
		info.InUse = true
	}

	return info, nil
}

// UnixSocketInfo contains information about a Unix socket
type UnixSocketInfo struct {
	Path        string
	Permissions os.FileMode
	Size        int64
	ModTime     time.Time
	IsSocket    bool
	InUse       bool
	UID         uint32
	GID         uint32
	Inode       uint64
}

// String returns a string representation of the socket info
func (info *UnixSocketInfo) String() string {
	status := "available"
	if info.InUse {
		status = "in use"
	}

	return fmt.Sprintf("Unix Socket: %s\n"+
		"  Permissions: %s\n"+
		"  Size: %d bytes\n"+
		"  Modified: %s\n"+
		"  Status: %s\n"+
		"  UID/GID: %d/%d\n"+
		"  Inode: %d",
		info.Path,
		info.Permissions,
		info.Size,
		info.ModTime.Format(time.RFC3339),
		status,
		info.UID,
		info.GID,
		info.Inode)
}