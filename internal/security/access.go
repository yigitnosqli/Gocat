package security

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"

	"github.com/ibrahmsql/gocat/internal/logger"
)

// AccessControl manages IP-based access control
type AccessControl struct {
	allowList    map[string]bool
	denyList     map[string]bool
	allowNets    []*net.IPNet
	denyNets     []*net.IPNet
	defaultAllow bool
	mutex        sync.RWMutex
}

// NewAccessControl creates and returns a new AccessControl with empty allow/deny
// maps and network slices. The instance is initialized with a default "allow"
// policy (DefaultAllow = true).
func NewAccessControl() *AccessControl {
	return &AccessControl{
		allowList:    make(map[string]bool),
		denyList:     make(map[string]bool),
		allowNets:    make([]*net.IPNet, 0),
		denyNets:     make([]*net.IPNet, 0),
		defaultAllow: true, // Default behavior: allow all
	}
}

// SetDefaultPolicy sets the default access policy
func (ac *AccessControl) SetDefaultPolicy(allow bool) {
	ac.mutex.Lock()
	defer ac.mutex.Unlock()
	ac.defaultAllow = allow
}

// AddAllowedHost adds a host to the allow list
func (ac *AccessControl) AddAllowedHost(host string) error {
	ac.mutex.Lock()
	defer ac.mutex.Unlock()

	// Try to parse as CIDR first
	if strings.Contains(host, "/") {
		_, ipNet, err := net.ParseCIDR(host)
		if err != nil {
			return fmt.Errorf("invalid CIDR notation: %s", host)
		}
		ac.allowNets = append(ac.allowNets, ipNet)
		logger.Debug("Added allowed network: %s", host)
		return nil
	}

	// Try to parse as IP address
	if ip := net.ParseIP(host); ip != nil {
		ac.allowList[ip.String()] = true
		logger.Debug("Added allowed IP: %s", ip.String())
		return nil
	}

	// Try to resolve hostname
	ips, err := net.LookupIP(host)
	if err != nil {
		return fmt.Errorf("failed to resolve hostname %s: %w", host, err)
	}

	for _, ip := range ips {
		ac.allowList[ip.String()] = true
		logger.Debug("Added allowed IP from hostname %s: %s", host, ip.String())
	}

	return nil
}

// AddDeniedHost adds a host to the deny list
func (ac *AccessControl) AddDeniedHost(host string) error {
	ac.mutex.Lock()
	defer ac.mutex.Unlock()

	// Try to parse as CIDR first
	if strings.Contains(host, "/") {
		_, ipNet, err := net.ParseCIDR(host)
		if err != nil {
			return fmt.Errorf("invalid CIDR notation: %s", host)
		}
		ac.denyNets = append(ac.denyNets, ipNet)
		logger.Debug("Added denied network: %s", host)
		return nil
	}

	// Try to parse as IP address
	if ip := net.ParseIP(host); ip != nil {
		ac.denyList[ip.String()] = true
		logger.Debug("Added denied IP: %s", ip.String())
		return nil
	}

	// Try to resolve hostname
	ips, err := net.LookupIP(host)
	if err != nil {
		return fmt.Errorf("failed to resolve hostname %s: %w", host, err)
	}

	for _, ip := range ips {
		ac.denyList[ip.String()] = true
		logger.Debug("Added denied IP from hostname %s: %s", host, ip.String())
	}

	return nil
}

// LoadAllowFile loads allowed hosts from a file
func (ac *AccessControl) LoadAllowFile(filename string) error {
	hosts, err := ac.loadHostsFromFile(filename)
	if err != nil {
		return fmt.Errorf("failed to load allow file: %w", err)
	}

	for _, host := range hosts {
		if err := ac.AddAllowedHost(host); err != nil {
			logger.Warn("Failed to add allowed host %s: %v", host, err)
		}
	}

	logger.Info("Loaded %d allowed hosts from %s", len(hosts), filename)
	return nil
}

// LoadDenyFile loads denied hosts from a file
func (ac *AccessControl) LoadDenyFile(filename string) error {
	hosts, err := ac.loadHostsFromFile(filename)
	if err != nil {
		return fmt.Errorf("failed to load deny file: %w", err)
	}

	for _, host := range hosts {
		if err := ac.AddDeniedHost(host); err != nil {
			logger.Warn("Failed to add denied host %s: %v", host, err)
		}
	}

	logger.Info("Loaded %d denied hosts from %s", len(hosts), filename)
	return nil
}

// loadHostsFromFile loads hosts from a text file (one per line)
func (ac *AccessControl) loadHostsFromFile(filename string) ([]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var hosts []string
	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		
		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Handle inline comments
		if idx := strings.Index(line, "#"); idx != -1 {
			line = strings.TrimSpace(line[:idx])
		}

		if line != "" {
			hosts = append(hosts, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading file at line %d: %w", lineNum, err)
	}

	return hosts, nil
}

// IsAllowed checks if a connection from the given address is allowed
func (ac *AccessControl) IsAllowed(addr net.Addr) bool {
	ac.mutex.RLock()
	defer ac.mutex.RUnlock()

	// Extract IP from address
	var ip net.IP
	switch a := addr.(type) {
	case *net.TCPAddr:
		ip = a.IP
	case *net.UDPAddr:
		ip = a.IP
	case *net.IPAddr:
		ip = a.IP
	default:
		// For unknown address types, try to parse the string
		host, _, err := net.SplitHostPort(addr.String())
		if err != nil {
			logger.Warn("Failed to parse address: %s", addr.String())
			return ac.defaultAllow
		}
		ip = net.ParseIP(host)
		if ip == nil {
			logger.Warn("Failed to parse IP from address: %s", host)
			return ac.defaultAllow
		}
	}

	ipStr := ip.String()

	// Check deny list first (deny takes precedence)
	if ac.denyList[ipStr] {
		logger.Debug("Connection denied by IP deny list: %s", ipStr)
		return false
	}

	// Check deny networks
	for _, denyNet := range ac.denyNets {
		if denyNet.Contains(ip) {
			logger.Debug("Connection denied by network deny list: %s in %s", ipStr, denyNet.String())
			return false
		}
	}

	// If we have allow rules, check them
	if len(ac.allowList) > 0 || len(ac.allowNets) > 0 {
		// Check allow list
		if ac.allowList[ipStr] {
			logger.Debug("Connection allowed by IP allow list: %s", ipStr)
			return true
		}

		// Check allow networks
		for _, allowNet := range ac.allowNets {
			if allowNet.Contains(ip) {
				logger.Debug("Connection allowed by network allow list: %s in %s", ipStr, allowNet.String())
				return true
			}
		}

		// If we have allow rules but IP is not in them, deny
		logger.Debug("Connection denied: %s not in allow list", ipStr)
		return false
	}

	// No specific rules, use default policy
	logger.Debug("Connection using default policy for %s: %v", ipStr, ac.defaultAllow)
	return ac.defaultAllow
}

// GetStats returns access control statistics
func (ac *AccessControl) GetStats() *AccessStats {
	ac.mutex.RLock()
	defer ac.mutex.RUnlock()

	return &AccessStats{
		AllowedIPs:      len(ac.allowList),
		DeniedIPs:       len(ac.denyList),
		AllowedNetworks: len(ac.allowNets),
		DeniedNetworks:  len(ac.denyNets),
		DefaultAllow:    ac.defaultAllow,
	}
}

// AccessStats contains access control statistics
type AccessStats struct {
	AllowedIPs      int
	DeniedIPs       int
	AllowedNetworks int
	DeniedNetworks  int
	DefaultAllow    bool
}

// String returns a string representation of the access control stats
func (stats *AccessStats) String() string {
	defaultPolicy := "deny"
	if stats.DefaultAllow {
		defaultPolicy = "allow"
	}

	return fmt.Sprintf("Access Control Stats:\n"+
		"  Allowed IPs: %d\n"+
		"  Denied IPs: %d\n"+
		"  Allowed Networks: %d\n"+
		"  Denied Networks: %d\n"+
		"  Default Policy: %s",
		stats.AllowedIPs,
		stats.DeniedIPs,
		stats.AllowedNetworks,
		stats.DeniedNetworks,
		defaultPolicy)
}

// Clear removes all access control rules
func (ac *AccessControl) Clear() {
	ac.mutex.Lock()
	defer ac.mutex.Unlock()

	ac.allowList = make(map[string]bool)
	ac.denyList = make(map[string]bool)
	ac.allowNets = make([]*net.IPNet, 0)
	ac.denyNets = make([]*net.IPNet, 0)
	ac.defaultAllow = true

	logger.Debug("Access control rules cleared")
}

// RemoveAllowedHost removes a host from the allow list
func (ac *AccessControl) RemoveAllowedHost(host string) {
	ac.mutex.Lock()
	defer ac.mutex.Unlock()

	if ip := net.ParseIP(host); ip != nil {
		delete(ac.allowList, ip.String())
		logger.Debug("Removed allowed IP: %s", ip.String())
	}
}

// RemoveDeniedHost removes a host from the deny list
func (ac *AccessControl) RemoveDeniedHost(host string) {
	ac.mutex.Lock()
	defer ac.mutex.Unlock()

	if ip := net.ParseIP(host); ip != nil {
		delete(ac.denyList, ip.String())
		logger.Debug("Removed denied IP: %s", ip.String())
	}
}