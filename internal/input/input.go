package input

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// ParseHostPort parses host and port from command line arguments
// If only one argument is provided, it's treated as port with default host
func ParseHostPort(args []string, defaultHost string) (string, string, error) {
	if len(args) == 0 {
		return "", "", fmt.Errorf("missing host and port")
	}

	if len(args) == 1 {
		// Only port provided
		port := args[0]
		if err := validatePort(port); err != nil {
			return "", "", err
		}
		return defaultHost, port, nil
	}

	if len(args) == 2 {
		// Host and port provided
		host, port := args[0], args[1]
		if err := validatePort(port); err != nil {
			return "", "", err
		}
		return host, port, nil
	}

	return "", "", fmt.Errorf("too many arguments")
}

// validatePort validates if the port is a valid number
func validatePort(port string) error {
	portNum, err := strconv.Atoi(port)
	if err != nil {
		return fmt.Errorf("invalid port number: %s", port)
	}

	if portNum < 1 || portNum > 65535 {
		return fmt.Errorf("port number out of range (1-65535): %d", portNum)
	}

	return nil
}

// ParseCommand parses and validates command strings
func ParseCommand(command string) (string, []string) {
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return "", nil
	}

	return parts[0], parts[1:]
}

// ValidateShell validates if the shell path is reasonable
func ValidateShell(shell string) error {
	if shell == "" {
		return fmt.Errorf("shell cannot be empty")
	}

	// Validate shell file exists and is executable
	if _, err := os.Stat(shell); os.IsNotExist(err) {
		return fmt.Errorf("shell does not exist: %s", shell)
	} else if err != nil {
		return fmt.Errorf("cannot access shell: %w", err)
	}

	// Check if shell is executable
	file, err := os.Open(shell)
	if err != nil {
		return fmt.Errorf("cannot open shell for reading: %w", err)
	}
	file.Close()
	if strings.Contains(shell, ";") || strings.Contains(shell, "&") {
		return fmt.Errorf("shell path contains invalid characters")
	}

	return nil
}
