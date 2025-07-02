package security

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"net"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// InputValidator provides input validation functions
type InputValidator struct {
	MaxHostnameLength int
	MaxPortNumber     int
	AllowedProtocols  []string
}

// NewInputValidator creates a new input validator with default settings
func NewInputValidator() *InputValidator {
	return &InputValidator{
		MaxHostnameLength: 253, // RFC 1035
		MaxPortNumber:     65535,
		AllowedProtocols:  []string{"tcp", "udp"},
	}
}

// ValidateHostname validates a hostname or IP address
func (v *InputValidator) ValidateHostname(hostname string) error {
	if len(hostname) == 0 {
		return fmt.Errorf("hostname cannot be empty")
	}

	if len(hostname) > v.MaxHostnameLength {
		return fmt.Errorf("hostname too long: %d characters (max %d)", len(hostname), v.MaxHostnameLength)
	}

	// Check if it's a valid IP address
	if ip := net.ParseIP(hostname); ip != nil {
		return nil
	}

	// Validate hostname format
	hostnameRegex := regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?)*$`)
	if !hostnameRegex.MatchString(hostname) {
		return fmt.Errorf("invalid hostname format: %s", hostname)
	}

	return nil
}

// ValidatePort validates a port number
func (v *InputValidator) ValidatePort(portStr string) (int, error) {
	if len(portStr) == 0 {
		return 0, fmt.Errorf("port cannot be empty")
	}

	port, err := strconv.Atoi(portStr)
	if err != nil {
		return 0, fmt.Errorf("invalid port number: %s", portStr)
	}

	if port < 1 || port > v.MaxPortNumber {
		return 0, fmt.Errorf("port number out of range: %d (must be 1-%d)", port, v.MaxPortNumber)
	}

	return port, nil
}

// ValidatePortRange validates a port range (e.g., "80-443" or "80")
func (v *InputValidator) ValidatePortRange(portRange string) ([]int, error) {
	if len(portRange) == 0 {
		return nil, fmt.Errorf("port range cannot be empty")
	}

	// Handle single port
	if !strings.Contains(portRange, "-") {
		port, err := v.ValidatePort(portRange)
		if err != nil {
			return nil, err
		}
		return []int{port}, nil
	}

	// Handle port range
	parts := strings.Split(portRange, "-")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid port range format: %s (expected start-end)", portRange)
	}

	startPort, err := v.ValidatePort(strings.TrimSpace(parts[0]))
	if err != nil {
		return nil, fmt.Errorf("invalid start port: %w", err)
	}

	endPort, err := v.ValidatePort(strings.TrimSpace(parts[1]))
	if err != nil {
		return nil, fmt.Errorf("invalid end port: %w", err)
	}

	if startPort > endPort {
		return nil, fmt.Errorf("start port (%d) cannot be greater than end port (%d)", startPort, endPort)
	}

	if endPort-startPort > 10000 {
		return nil, fmt.Errorf("port range too large: %d ports (max 10000)", endPort-startPort+1)
	}

	var ports []int
	for port := startPort; port <= endPort; port++ {
		ports = append(ports, port)
	}

	return ports, nil
}

// ValidateProtocol validates a network protocol
func (v *InputValidator) ValidateProtocol(protocol string) error {
	protocol = strings.ToLower(protocol)
	for _, allowed := range v.AllowedProtocols {
		if protocol == allowed {
			return nil
		}
	}
	return fmt.Errorf("unsupported protocol: %s (allowed: %v)", protocol, v.AllowedProtocols)
}

// SanitizeCommand sanitizes a command string to prevent injection
func (v *InputValidator) SanitizeCommand(command string) (string, error) {
	if len(command) == 0 {
		return "", fmt.Errorf("command cannot be empty")
	}

	if len(command) > 1000 {
		return "", fmt.Errorf("command too long: %d characters (max 1000)", len(command))
	}

	// Remove dangerous characters and sequences
	dangerous := []string{
		";", "&", "|", "`", "$(", "${", "<(", ">(",
		"rm ", "del ", "format ", "mkfs", "dd if=",
	}

	for _, danger := range dangerous {
		if strings.Contains(strings.ToLower(command), danger) {
			return "", fmt.Errorf("command contains dangerous sequence: %s", danger)
		}
	}

	return command, nil
}

// RateLimiter provides rate limiting functionality
type RateLimiter struct {
	requests    map[string][]time.Time
	maxRequests int
	window      time.Duration
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(maxRequests int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		requests:    make(map[string][]time.Time),
		maxRequests: maxRequests,
		window:      window,
	}
}

// Allow checks if a request from the given identifier is allowed
func (rl *RateLimiter) Allow(identifier string) bool {
	now := time.Now()

	// Clean old requests
	if requests, exists := rl.requests[identifier]; exists {
		var validRequests []time.Time
		for _, req := range requests {
			if now.Sub(req) < rl.window {
				validRequests = append(validRequests, req)
			}
		}
		rl.requests[identifier] = validRequests
	}

	// Check if under limit
	if len(rl.requests[identifier]) >= rl.maxRequests {
		return false
	}

	// Add current request
	rl.requests[identifier] = append(rl.requests[identifier], now)
	return true
}

// GenerateSecureToken generates a cryptographically secure random token
func GenerateSecureToken(length int) (string, error) {
	if length <= 0 {
		return "", fmt.Errorf("token length must be positive")
	}

	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}

	return hex.EncodeToString(bytes), nil
}

// SecureCompare performs a constant-time comparison of two strings
func SecureCompare(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

// IsPrivateIP checks if an IP address is in a private range
func IsPrivateIP(ip net.IP) bool {
	privateRanges := []string{
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
		"127.0.0.0/8",
		"169.254.0.0/16",
		"::1/128",
		"fc00::/7",
		"fe80::/10",
	}

	for _, rangeStr := range privateRanges {
		_, network, err := net.ParseCIDR(rangeStr)
		if err != nil {
			continue
		}
		if network.Contains(ip) {
			return true
		}
	}

	return false
}
