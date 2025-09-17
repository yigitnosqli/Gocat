package network

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ibrahmsql/gocat/internal/errors"
)

// DualStackConfig holds configuration for dual-stack networking
type DualStackConfig struct {
	PreferIPv6         bool          `yaml:"prefer_ipv6"`
	IPv4Enabled        bool          `yaml:"ipv4_enabled"`
	IPv6Enabled        bool          `yaml:"ipv6_enabled"`
	HappyEyeballsDelay time.Duration `yaml:"happy_eyeballs_delay"`
	ResolutionTimeout  time.Duration `yaml:"resolution_timeout"`
	ConnectionRacing   bool          `yaml:"connection_racing"`
	MaxConcurrentDials int           `yaml:"max_concurrent_dials"`
}

// DefaultDualStackConfig returns default dual-stack configuration
func DefaultDualStackConfig() *DualStackConfig {
	return &DualStackConfig{
		PreferIPv6:         false, // Prefer IPv4 for compatibility
		IPv4Enabled:        true,
		IPv6Enabled:        true,
		HappyEyeballsDelay: 300 * time.Millisecond,
		ResolutionTimeout:  5 * time.Second,
		ConnectionRacing:   true,
		MaxConcurrentDials: 2,
	}
}

// DualStackDialer provides dual-stack dialing capabilities
type DualStackDialer struct {
	config           *DualStackConfig
	metricsCollector MetricsCollector
	mu               sync.RWMutex
}

// NewDualStackDialer creates a new dual-stack dialer
func NewDualStackDialer(config *DualStackConfig) *DualStackDialer {
	if config == nil {
		config = DefaultDualStackConfig()
	}

	return &DualStackDialer{
		config: config,
	}
}

// AddressInfo holds information about a resolved address
type AddressInfo struct {
	IP       net.IP
	Port     int
	Network  string // "tcp4" or "tcp6"
	Priority int    // Lower is higher priority
}

// DialResult holds the result of a dial attempt
type DialResult struct {
	Conn    net.Conn
	Address AddressInfo
	Error   error
	Latency time.Duration
}

// Dial performs dual-stack dialing with Happy Eyeballs algorithm
func (d *DualStackDialer) Dial(ctx context.Context, address string) (net.Conn, error) {
	startTime := time.Now()

	// Parse address
	host, portStr, err := net.SplitHostPort(address)
	if err != nil {
		return nil, errors.ValidationError("VAL021", fmt.Sprintf("Invalid address format: %s", address)).WithSuggestion("Use format 'host:port'")
	}

	port, err := strconv.Atoi(portStr)
	if err != nil {
		return nil, errors.ValidationError("VAL022", fmt.Sprintf("Invalid port: %s", portStr))
	}

	// Validate IPv6 address format if it looks like one
	if strings.Contains(host, ":") && !d.isValidIPv6(host) {
		return nil, errors.ValidationError("VAL023", fmt.Sprintf("Invalid IPv6 address: %s", host))
	}

	// Resolve addresses
	addresses, err := d.resolveAddresses(ctx, host, port)
	if err != nil {
		return nil, err
	}

	if len(addresses) == 0 {
		return nil, errors.NetworkError("NET044", fmt.Sprintf("No addresses found for %s", host)).WithUserFriendly(fmt.Sprintf("Cannot resolve %s", host))
	}

	// Sort addresses by preference
	d.sortAddressesByPreference(addresses)

	// Perform connection racing if enabled
	var conn net.Conn
	if d.config.ConnectionRacing && len(addresses) > 1 {
		conn, err = d.dialWithRacing(ctx, addresses)
	} else {
		conn, err = d.dialSequential(ctx, addresses)
	}

	// Record metrics
	if d.metricsCollector != nil {
		duration := time.Since(startTime)
		tags := map[string]string{
			"host":    host,
			"success": "false",
		}

		if err == nil {
			tags["success"] = "true"
			if tcpAddr, ok := conn.RemoteAddr().(*net.TCPAddr); ok {
				if tcpAddr.IP.To4() != nil {
					tags["protocol"] = "ipv4"
				} else {
					tags["protocol"] = "ipv6"
				}
			}
		}

		d.metricsCollector.RecordTimer("dualstack_dial_duration", duration, tags)
		d.metricsCollector.IncrementCounter("dualstack_dial_attempts", tags)
	}

	return conn, err
}

// resolveAddresses resolves both IPv4 and IPv6 addresses for the host
func (d *DualStackDialer) resolveAddresses(ctx context.Context, host string, port int) ([]AddressInfo, error) {
	// Create context with timeout for resolution
	resolveCtx, cancel := context.WithTimeout(ctx, d.config.ResolutionTimeout)
	defer cancel()

	// Check if host is already an IP address
	if ip := net.ParseIP(host); ip != nil {
		network := "tcp4"
		if ip.To4() == nil {
			network = "tcp6"
		}
		return []AddressInfo{{
			IP:      ip,
			Port:    port,
			Network: network,
		}}, nil
	}

	var addresses []AddressInfo
	var wg sync.WaitGroup
	var mu sync.Mutex
	var resolveErrors []error

	// Resolve IPv4 addresses
	if d.config.IPv4Enabled {
		wg.Add(1)
		go func() {
			defer wg.Done()

			ips, err := net.DefaultResolver.LookupIPAddr(resolveCtx, host)
			if err != nil {
				mu.Lock()
				resolveErrors = append(resolveErrors, fmt.Errorf("IPv4 resolution failed: %w", err))
				mu.Unlock()
				return
			}

			mu.Lock()
			for _, ip := range ips {
				if ip.IP.To4() != nil {
					addresses = append(addresses, AddressInfo{
						IP:       ip.IP,
						Port:     port,
						Network:  "tcp4",
						Priority: d.getAddressPriority(ip.IP),
					})
				}
			}
			mu.Unlock()
		}()
	}

	// Resolve IPv6 addresses
	if d.config.IPv6Enabled {
		wg.Add(1)
		go func() {
			defer wg.Done()

			ips, err := net.DefaultResolver.LookupIPAddr(resolveCtx, host)
			if err != nil {
				mu.Lock()
				resolveErrors = append(resolveErrors, fmt.Errorf("IPv6 resolution failed: %w", err))
				mu.Unlock()
				return
			}

			mu.Lock()
			for _, ip := range ips {
				if ip.IP.To4() == nil {
					addresses = append(addresses, AddressInfo{
						IP:       ip.IP,
						Port:     port,
						Network:  "tcp6",
						Priority: d.getAddressPriority(ip.IP),
					})
				}
			}
			mu.Unlock()
		}()
	}

	wg.Wait()

	// Check if we got any addresses
	if len(addresses) == 0 {
		if len(resolveErrors) > 0 {
			return nil, errors.NetworkError("NET045", fmt.Sprintf("DNS resolution failed for %s", host)).WithCause(resolveErrors[0])
		}
		return nil, errors.NetworkError("NET046", fmt.Sprintf("No IP addresses found for %s", host))
	}

	return addresses, nil
}

// getAddressPriority returns priority for an IP address (lower is higher priority)
func (d *DualStackDialer) getAddressPriority(ip net.IP) int {
	if ip.To4() != nil {
		// IPv4 address
		if d.config.PreferIPv6 {
			return 2
		}
		return 1
	} else {
		// IPv6 address
		if d.config.PreferIPv6 {
			return 1
		}
		return 2
	}
}

// sortAddressesByPreference sorts addresses by preference and priority
func (d *DualStackDialer) sortAddressesByPreference(addresses []AddressInfo) {
	// Simple bubble sort by priority (good enough for small lists)
	n := len(addresses)
	for i := 0; i < n-1; i++ {
		for j := 0; j < n-i-1; j++ {
			if addresses[j].Priority > addresses[j+1].Priority {
				addresses[j], addresses[j+1] = addresses[j+1], addresses[j]
			}
		}
	}
}

// dialWithRacing performs connection racing (Happy Eyeballs)
func (d *DualStackDialer) dialWithRacing(ctx context.Context, addresses []AddressInfo) (net.Conn, error) {
	if len(addresses) == 0 {
		return nil, errors.NetworkError("NET047", "No addresses to dial")
	}

	resultChan := make(chan DialResult, len(addresses))
	dialCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Start with the highest priority address
	go d.dialSingle(dialCtx, addresses[0], resultChan)
	dialsStarted := 1

	// Start additional dials with delay
	go func() {
		delay := d.config.HappyEyeballsDelay
		for i := 1; i < len(addresses) && i < d.config.MaxConcurrentDials; i++ {
			select {
			case <-dialCtx.Done():
				return
			case <-time.After(delay):
				go d.dialSingle(dialCtx, addresses[i], resultChan)
				dialsStarted++
				// Increase delay for subsequent attempts
				delay = delay * 2
			}
		}
	}()

	// Wait for first successful connection
	var lastErr error
	successCount := 0
	errorCount := 0

	for successCount == 0 && errorCount < dialsStarted {
		select {
		case <-ctx.Done():
			return nil, errors.TimeoutError("NET048", "Dial context cancelled").WithCause(ctx.Err())
		case result := <-resultChan:
			if result.Error == nil {
				// Success! Cancel other attempts
				cancel()

				// Record successful connection metrics
				if d.metricsCollector != nil {
					d.metricsCollector.RecordTimer("connection_latency", result.Latency, map[string]string{
						"network": result.Address.Network,
						"ip":      result.Address.IP.String(),
					})
				}

				return result.Conn, nil
			}

			errorCount++
			lastErr = result.Error

			// Record failed connection attempt
			if d.metricsCollector != nil {
				d.metricsCollector.IncrementCounter("connection_failures", map[string]string{
					"network": result.Address.Network,
					"ip":      result.Address.IP.String(),
				})
			}
		}
	}

	return nil, lastErr
}

// dialSequential performs sequential dialing
func (d *DualStackDialer) dialSequential(ctx context.Context, addresses []AddressInfo) (net.Conn, error) {
	var lastErr error

	for _, addr := range addresses {
		select {
		case <-ctx.Done():
			return nil, errors.TimeoutError("NET049", "Dial context cancelled").WithCause(ctx.Err())
		default:
		}

		resultChan := make(chan DialResult, 1)
		go d.dialSingle(ctx, addr, resultChan)

		result := <-resultChan
		if result.Error == nil {
			return result.Conn, nil
		}

		lastErr = result.Error
	}

	return nil, lastErr
}

// dialSingle performs a single dial attempt
func (d *DualStackDialer) dialSingle(ctx context.Context, addr AddressInfo, resultChan chan<- DialResult) {
	startTime := time.Now()

	dialer := &net.Dialer{
		Timeout: 30 * time.Second,
	}

	address := net.JoinHostPort(addr.IP.String(), strconv.Itoa(addr.Port))
	conn, err := dialer.DialContext(ctx, addr.Network, address)

	latency := time.Since(startTime)

	resultChan <- DialResult{
		Conn:    conn,
		Address: addr,
		Error:   err,
		Latency: latency,
	}
}

// isValidIPv6 checks if a string is a valid IPv6 address
func (d *DualStackDialer) isValidIPv6(host string) bool {
	ip := net.ParseIP(host)
	return ip != nil && ip.To4() == nil
}

// SetMetrics sets the metrics collector
func (d *DualStackDialer) SetMetrics(collector MetricsCollector) {
	d.mu.Lock()
	d.metricsCollector = collector
	d.mu.Unlock()
}

// DualStackListener provides dual-stack listening capabilities
type DualStackListener struct {
	ipv4Listener net.Listener
	ipv6Listener net.Listener
	config       *DualStackConfig
	acceptChan   chan net.Conn
	errorChan    chan error
	ctx          context.Context
	cancel       context.CancelFunc
	mu           sync.RWMutex
	closed       bool
}

// NewDualStackListener creates a new dual-stack listener
func NewDualStackListener(config *DualStackConfig) *DualStackListener {
	if config == nil {
		config = DefaultDualStackConfig()
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &DualStackListener{
		config:     config,
		acceptChan: make(chan net.Conn, 10),
		errorChan:  make(chan error, 10),
		ctx:        ctx,
		cancel:     cancel,
	}
}

// Listen starts listening on both IPv4 and IPv6
func (l *DualStackListener) Listen(address string) error {
	host, portStr, err := net.SplitHostPort(address)
	if err != nil {
		return errors.ValidationError("VAL024", fmt.Sprintf("Invalid address format: %s", address))
	}

	port, err := strconv.Atoi(portStr)
	if err != nil {
		return errors.ValidationError("VAL025", fmt.Sprintf("Invalid port: %s", portStr))
	}

	// Start IPv4 listener if enabled
	if l.config.IPv4Enabled {
		ipv4Addr := net.JoinHostPort("0.0.0.0", strconv.Itoa(port))
		if host != "" && host != "::" {
			// Use specific IPv4 address if provided
			if ip := net.ParseIP(host); ip != nil && ip.To4() != nil {
				ipv4Addr = net.JoinHostPort(host, strconv.Itoa(port))
			}
		}

		l.ipv4Listener, err = net.Listen("tcp4", ipv4Addr)
		if err != nil {
			return errors.WrapError(err, errors.ErrorTypeNetwork, errors.SeverityHigh, "NET050", "Failed to create IPv4 listener")
		}

		go l.acceptLoop(l.ipv4Listener, "ipv4")
	}

	// Start IPv6 listener if enabled
	if l.config.IPv6Enabled {
		ipv6Addr := net.JoinHostPort("::", strconv.Itoa(port))
		if host != "" && host != "0.0.0.0" {
			// Use specific IPv6 address if provided
			if ip := net.ParseIP(host); ip != nil && ip.To4() == nil {
				ipv6Addr = net.JoinHostPort(host, strconv.Itoa(port))
			}
		}

		l.ipv6Listener, err = net.Listen("tcp6", ipv6Addr)
		if err != nil {
			// If IPv4 listener was created, close it
			if l.ipv4Listener != nil {
				l.ipv4Listener.Close()
			}
			return errors.WrapError(err, errors.ErrorTypeNetwork, errors.SeverityHigh, "NET051", "Failed to create IPv6 listener")
		}

		go l.acceptLoop(l.ipv6Listener, "ipv6")
	}

	return nil
}

// acceptLoop runs the accept loop for a specific listener
func (l *DualStackListener) acceptLoop(listener net.Listener, protocol string) {
	for {
		select {
		case <-l.ctx.Done():
			return
		default:
		}

		conn, err := listener.Accept()
		if err != nil {
			select {
			case <-l.ctx.Done():
				return
			case l.errorChan <- errors.WrapError(err, errors.ErrorTypeNetwork, errors.SeverityMedium, "NET052", fmt.Sprintf("Accept failed on %s listener", protocol)):
			}
			continue
		}

		select {
		case <-l.ctx.Done():
			conn.Close()
			return
		case l.acceptChan <- conn:
		}
	}
}

// Accept accepts a connection from either IPv4 or IPv6 listener
func (l *DualStackListener) Accept() (net.Conn, error) {
	l.mu.RLock()
	closed := l.closed
	l.mu.RUnlock()

	if closed {
		return nil, errors.NetworkError("NET053", "Dual-stack listener is closed")
	}

	select {
	case <-l.ctx.Done():
		return nil, errors.NetworkError("NET054", "Dual-stack listener context cancelled")
	case conn := <-l.acceptChan:
		return conn, nil
	case err := <-l.errorChan:
		return nil, err
	}
}

// Close closes both listeners
func (l *DualStackListener) Close() error {
	l.mu.Lock()
	if l.closed {
		l.mu.Unlock()
		return nil
	}
	l.closed = true
	l.mu.Unlock()

	l.cancel()

	var lastErr error

	if l.ipv4Listener != nil {
		if err := l.ipv4Listener.Close(); err != nil {
			lastErr = err
		}
	}

	if l.ipv6Listener != nil {
		if err := l.ipv6Listener.Close(); err != nil {
			lastErr = err
		}
	}

	return lastErr
}

// Addr returns the address of the first available listener
func (l *DualStackListener) Addr() net.Addr {
	if l.ipv4Listener != nil {
		return l.ipv4Listener.Addr()
	}
	if l.ipv6Listener != nil {
		return l.ipv6Listener.Addr()
	}
	return nil
}

// GetAddresses returns addresses of both listeners
func (l *DualStackListener) GetAddresses() map[string]net.Addr {
	addresses := make(map[string]net.Addr)

	if l.ipv4Listener != nil {
		addresses["ipv4"] = l.ipv4Listener.Addr()
	}

	if l.ipv6Listener != nil {
		addresses["ipv6"] = l.ipv6Listener.Addr()
	}

	return addresses
}
