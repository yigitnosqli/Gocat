package network

import (
	"context"
	"fmt"
	"net"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"github.com/ibrahmsql/gocat/internal/logger"
)

// SCTP protocol constants
const (
	IPPROTO_SCTP = 132
	SOL_SCTP     = 132

	// SCTP socket options
	SCTP_RTOINFO               = 0
	SCTP_ASSOCINFO             = 1
	SCTP_INITMSG               = 2
	SCTP_NODELAY               = 3
	SCTP_AUTOCLOSE             = 4
	SCTP_SET_PEER_ADDR         = 5
	SCTP_PRIMARY_ADDR          = 6
	SCTP_ADAPTATION_LAYER      = 7
	SCTP_DISABLE_FRAGMENTS     = 8
	SCTP_PEER_ADDR_PARAMS      = 9
	SCTP_DEFAULT_SEND_PARAM    = 10
	SCTP_EVENTS                = 11
	SCTP_I_WANT_MAPPED_V4_ADDR = 12
	SCTP_MAXSEG                = 13
	SCTP_STATUS                = 14
	SCTP_GET_PEER_ADDR_INFO    = 15
)

// SCTPConfig holds SCTP-specific configuration
type SCTPConfig struct {
	Streams        int           // Number of streams
	MaxAttempts    int           // Maximum init attempts
	MaxInitTimeout time.Duration // Maximum init timeout
	Heartbeat      bool          // Enable heartbeat
	Nodelay        bool          // Disable Nagle algorithm
	AutoClose      time.Duration // Auto close timeout
}

// DefaultSCTPConfig returns a pointer to an SCTPConfig populated with sane defaults:
// Streams = 10, MaxAttempts = 4, MaxInitTimeout = 60s, Heartbeat = true, Nodelay = false,
// and AutoClose = 0 (disabled).
func DefaultSCTPConfig() *SCTPConfig {
	return &SCTPConfig{
		Streams:        10,
		MaxAttempts:    4,
		MaxInitTimeout: 60 * time.Second,
		Heartbeat:      true,
		Nodelay:        false,
		AutoClose:      0, // Disabled
	}
}

// SCTPConn represents an SCTP connection
type SCTPConn struct {
	fd            int
	laddr         *SCTPAddr
	raddr         *SCTPAddr
	config        *SCTPConfig
	readDeadline  time.Time
	writeDeadline time.Time
	mu            sync.RWMutex
}

// SCTPAddr represents an SCTP address
type SCTPAddr struct {
	IPs  []net.IP
	Port int
}

// Network returns the network type
func (a *SCTPAddr) Network() string {
	return "sctp"
}

// String returns string representation of the address
func (a *SCTPAddr) String() string {
	if len(a.IPs) == 0 {
		return fmt.Sprintf(":%d", a.Port)
	}
	if len(a.IPs) == 1 {
		return fmt.Sprintf("%s:%d", a.IPs[0].String(), a.Port)
	}
	// Multiple IPs (multihoming)
	var ips []string
	for _, ip := range a.IPs {
		ips = append(ips, ip.String())
	}
	return fmt.Sprintf("[%s]:%d", fmt.Sprintf("%v", ips), a.Port)
}

// ResolveSCTPAddr resolves the given SCTP network/address string into an SCTPAddr.
//
// ResolveSCTPAddr accepts network "sctp", "sctp4", or "sctp6" and an address of the
// form "host:port". If host is empty the returned address will contain the
// wildcard IP appropriate for the network family (IPv4 or IPv6). Otherwise the
// host is resolved to one or more IPs and included in the returned SCTPAddr.
// The returned SCTPAddr.IPs slice may contain multiple entries for multihomed
// hosts. Errors are returned for unsupported network values, invalid address
// syntax, unknown ports, or host resolution failures.
func ResolveSCTPAddr(network, address string) (*SCTPAddr, error) {
	if network != "sctp" && network != "sctp4" && network != "sctp6" {
		return nil, fmt.Errorf("unsupported network type: %s", network)
	}

	host, portStr, err := net.SplitHostPort(address)
	if err != nil {
		return nil, fmt.Errorf("invalid address format: %w", err)
	}

	port, err := net.LookupPort("sctp", portStr)
	if err != nil {
		return nil, fmt.Errorf("invalid port: %w", err)
	}

	var ips []net.IP
	if host == "" {
		// Listen on all interfaces
		if network == "sctp6" {
			ips = []net.IP{net.IPv6zero}
		} else {
			ips = []net.IP{net.IPv4zero}
		}
	} else {
		// Resolve hostname
		resolvedIPs, err := net.LookupIP(host)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve host: %w", err)
		}
		ips = resolvedIPs
	}

	return &SCTPAddr{
		IPs:  ips,
		Port: port,
	}, nil
}

// Read reads data from SCTP connection
func (c *SCTPConn) Read(b []byte) (int, error) {
	n, err := syscall.Read(c.fd, b)
	if err != nil {
		return 0, err
	}
	return n, nil
}

// Write writes data to SCTP connection
func (c *SCTPConn) Write(b []byte) (int, error) {
	n, err := syscall.Write(c.fd, b)
	if err != nil {
		return 0, err
	}
	return n, nil
}

// Close closes the SCTP connection
func (c *SCTPConn) Close() error {
	if c.fd >= 0 {
		err := syscall.Close(c.fd)
		c.fd = -1
		return err
	}
	return nil
}

// LocalAddr returns local address
func (c *SCTPConn) LocalAddr() net.Addr {
	return c.laddr
}

// RemoteAddr returns remote address
func (c *SCTPConn) RemoteAddr() net.Addr {
	return c.raddr
}

// SetDeadline sets read and write deadlines
func (c *SCTPConn) SetDeadline(t time.Time) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	c.readDeadline = t
	c.writeDeadline = t
	
	// Set socket timeout if deadline is in the future
	if !t.IsZero() {
		duration := time.Until(t)
		if duration > 0 {
			if err := c.setSCTPTimeout(duration); err != nil {
				logger.Debug("Failed to set SCTP timeout: %v", err)
			}
		}
	}
	
	return nil
}

// SetReadDeadline sets read deadline
func (c *SCTPConn) SetReadDeadline(t time.Time) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	c.readDeadline = t
	
	// Set receive timeout on socket
	if !t.IsZero() {
		duration := time.Until(t)
		if duration > 0 {
			if err := c.setReceiveTimeout(duration); err != nil {
				logger.Debug("Failed to set SCTP receive timeout: %v", err)
			}
		}
	}
	
	return nil
}

// SetWriteDeadline sets write deadline
func (c *SCTPConn) SetWriteDeadline(t time.Time) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	c.writeDeadline = t
	
	// Set send timeout on socket
	if !t.IsZero() {
		duration := time.Until(t)
		if duration > 0 {
			if err := c.setSendTimeout(duration); err != nil {
				logger.Debug("Failed to set SCTP send timeout: %v", err)
			}
		}
	}
	
	return nil
}

// SCTPListener represents an SCTP listener
type SCTPListener struct {
	fd     int
	laddr  *SCTPAddr
	config *SCTPConfig
}

// Accept accepts incoming SCTP connections
func (l *SCTPListener) Accept() (net.Conn, error) {
	fd, sa, err := syscall.Accept(l.fd)
	if err != nil {
		return nil, err
	}

	// Parse remote address from sockaddr
	raddr, err := parseSockaddr(sa)
	if err != nil {
		syscall.Close(fd)
		return nil, fmt.Errorf("failed to parse remote address: %w", err)
	}

	conn := &SCTPConn{
		fd:     fd,
		laddr:  l.laddr,
		raddr:  raddr,
		config: l.config,
	}

	// Apply SCTP-specific socket options
	if err := l.applySCTPOptions(fd); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to apply SCTP options: %w", err)
	}

	return conn, nil
}

// Close closes the SCTP listener
func (l *SCTPListener) Close() error {
	if l.fd >= 0 {
		err := syscall.Close(l.fd)
		l.fd = -1
		return err
	}
	return nil
}

// Addr returns listener address
func (l *SCTPListener) Addr() net.Addr {
	return l.laddr
}

// applySCTPOptions applies SCTP-specific socket options
func (l *SCTPListener) applySCTPOptions(fd int) error {
	if l.config.Nodelay {
		if err := syscall.SetsockoptInt(fd, SOL_SCTP, SCTP_NODELAY, 1); err != nil {
			logger.Warn("Failed to set SCTP_NODELAY: %v", err)
		}
	}

	if l.config.AutoClose > 0 {
		autoCloseSeconds := int(l.config.AutoClose.Seconds())
		if err := syscall.SetsockoptInt(fd, SOL_SCTP, SCTP_AUTOCLOSE, autoCloseSeconds); err != nil {
			logger.Warn("Failed to set SCTP_AUTOCLOSE: %v", err)
		}
	}

	return nil
}

// ListenSCTP creates and returns an SCTP listener bound to the given local
// address and configured according to config.
//
// The network must be "sctp", "sctp4", or "sctp6". "sctp" auto-detects the
// address family from laddr (defaults to IPv4 if indeterminate). If laddr is
// nil or has no IPs, the listener is bound to all interfaces for the chosen
// family; laddr.Port is used as the bind port. If config is nil, DefaultSCTPConfig()
// is used.
//
// The function creates a native SCTP socket, enables SO_REUSEADDR, binds it and
// starts listening. It returns an SCTPListener that must be closed when no
// longer needed. Errors are returned for unsupported networks or on socket,
// bind, or listen failures (wrapping the underlying syscall errors).
func ListenSCTP(network string, laddr *SCTPAddr, config *SCTPConfig) (*SCTPListener, error) {
	if config == nil {
		config = DefaultSCTPConfig()
	}

	// Determine address family
	var family int
	switch network {
	case "sctp4":
		family = syscall.AF_INET
	case "sctp6":
		family = syscall.AF_INET6
	case "sctp":
		// Auto-detect based on address
		if laddr != nil && len(laddr.IPs) > 0 {
			if laddr.IPs[0].To4() != nil {
				family = syscall.AF_INET
			} else {
				family = syscall.AF_INET6
			}
		} else {
			family = syscall.AF_INET // Default to IPv4
		}
	default:
		return nil, fmt.Errorf("unsupported network: %s", network)
	}

	// Create SCTP socket
	fd, err := syscall.Socket(family, syscall.SOCK_STREAM, IPPROTO_SCTP)
	if err != nil {
		return nil, fmt.Errorf("failed to create SCTP socket: %w", err)
	}

	// Set socket options
	if err := syscall.SetsockoptInt(fd, syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1); err != nil {
		syscall.Close(fd)
		return nil, fmt.Errorf("failed to set SO_REUSEADDR: %w", err)
	}

	// Bind to address
	var sa syscall.Sockaddr
	if laddr != nil && len(laddr.IPs) > 0 {
		sa, err = createSockaddr(laddr.IPs[0], laddr.Port, family)
		if err != nil {
			syscall.Close(fd)
			return nil, fmt.Errorf("failed to create sockaddr: %w", err)
		}
	} else {
		// Bind to all interfaces
		if family == syscall.AF_INET6 {
			sa = &syscall.SockaddrInet6{Port: laddr.Port}
		} else {
			sa = &syscall.SockaddrInet4{Port: laddr.Port}
		}
	}

	if err := syscall.Bind(fd, sa); err != nil {
		syscall.Close(fd)
		return nil, fmt.Errorf("failed to bind SCTP socket: %w", err)
	}

	// Listen for connections
	if err := syscall.Listen(fd, syscall.SOMAXCONN); err != nil {
		syscall.Close(fd)
		return nil, fmt.Errorf("failed to listen on SCTP socket: %w", err)
	}

	listener := &SCTPListener{
		fd:     fd,
		laddr:  laddr,
		config: config,
	}

	return listener, nil
}

// DialSCTP dials an SCTP association to raddr from laddr using the specified network.
// It is a convenience wrapper around DialSCTPTimeout with a zero timeout (blocking connect).
// If config is nil, DefaultSCTPConfig is used.
func DialSCTP(network string, laddr, raddr *SCTPAddr, config *SCTPConfig) (*SCTPConn, error) {
	return DialSCTPTimeout(network, laddr, raddr, 0, config)
}

// DialSCTPTimeout establishes an SCTP connection to raddr within the given timeout.
//
// DialSCTPTimeout chooses the address family from network ("sctp", "sctp4", "sctp6")
// (for "sctp" it autodetects from raddr.IPs), optionally binds to laddr if provided,
// and uses a non-blocking connect when timeout > 0 to implement the timeout behavior.
// If config is nil DefaultSCTPConfig() is used. On success it returns an *SCTPConn
// with the underlying socket ready for use. Returns an error for unsupported network,
// address construction failures, bind/connect errors, or when the connection times out.
func DialSCTPTimeout(network string, laddr, raddr *SCTPAddr, timeout time.Duration, config *SCTPConfig) (*SCTPConn, error) {
	if config == nil {
		config = DefaultSCTPConfig()
	}

	if raddr == nil {
		return nil, fmt.Errorf("remote address cannot be nil")
	}

	// Determine address family
	var family int
	switch network {
	case "sctp4":
		family = syscall.AF_INET
	case "sctp6":
		family = syscall.AF_INET6
	case "sctp":
		// Auto-detect based on remote address
		if len(raddr.IPs) > 0 && raddr.IPs[0].To4() != nil {
			family = syscall.AF_INET
		} else {
			family = syscall.AF_INET6
		}
	default:
		return nil, fmt.Errorf("unsupported network: %s", network)
	}

	// Create SCTP socket
	fd, err := syscall.Socket(family, syscall.SOCK_STREAM, IPPROTO_SCTP)
	if err != nil {
		return nil, fmt.Errorf("failed to create SCTP socket: %w", err)
	}

	// Bind to local address if specified
	if laddr != nil && len(laddr.IPs) > 0 {
		lsa, err := createSockaddr(laddr.IPs[0], laddr.Port, family)
		if err != nil {
			syscall.Close(fd)
			return nil, fmt.Errorf("failed to create local sockaddr: %w", err)
		}
		if err := syscall.Bind(fd, lsa); err != nil {
			syscall.Close(fd)
			return nil, fmt.Errorf("failed to bind to local address: %w", err)
		}
	}

	// Create remote sockaddr
	rsa, err := createSockaddr(raddr.IPs[0], raddr.Port, family)
	if err != nil {
		syscall.Close(fd)
		return nil, fmt.Errorf("failed to create remote sockaddr: %w", err)
	}

	// Set non-blocking for timeout support
	if timeout > 0 {
		if err := syscall.SetNonblock(fd, true); err != nil {
			syscall.Close(fd)
			return nil, fmt.Errorf("failed to set non-blocking: %w", err)
		}
	}

	// Connect
	err = syscall.Connect(fd, rsa)
	if err != nil && err != syscall.EINPROGRESS {
		syscall.Close(fd)
		return nil, fmt.Errorf("failed to connect: %w", err)
	}

	// Handle timeout
	if timeout > 0 && err == syscall.EINPROGRESS {
		// Wait for connection to complete
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

	connectionLoop:
		for {
			select {
			case <-ctx.Done():
				syscall.Close(fd)
				return nil, fmt.Errorf("connection timeout")
			default:
				// Check if connection is ready
				soErr, err := syscall.GetsockoptInt(fd, syscall.SOL_SOCKET, syscall.SO_ERROR)
				if err == nil && soErr == 0 {
					// Connection successful
					break connectionLoop
				} else if err != nil || soErr != int(syscall.EINPROGRESS) {
					syscall.Close(fd)
					return nil, fmt.Errorf("connection failed: %v", err)
				}
				time.Sleep(10 * time.Millisecond)
			}
		}

		// Set back to blocking
		if err := syscall.SetNonblock(fd, false); err != nil {
			syscall.Close(fd)
			return nil, fmt.Errorf("failed to set blocking: %w", err)
		}
	}

	conn := &SCTPConn{
		fd:     fd,
		laddr:  laddr,
		raddr:  raddr,
		config: config,
	}

	return conn, nil
}

// createSockaddr constructs a syscall.Sockaddr for the given IP, port, and address family.
//
// It returns a *syscall.SockaddrInet4 when family is syscall.AF_INET and the provided IP is IPv4,
// or a *syscall.SockaddrInet6 when family is syscall.AF_INET6 and the provided IP is IPv6.
// Returns an error if the IP does not match the requested family or if the family is unsupported.
func createSockaddr(ip net.IP, port int, family int) (syscall.Sockaddr, error) {
	switch family {
	case syscall.AF_INET:
		if ip4 := ip.To4(); ip4 != nil {
			sa := &syscall.SockaddrInet4{Port: port}
			copy(sa.Addr[:], ip4)
			return sa, nil
		}
		return nil, fmt.Errorf("invalid IPv4 address")
	case syscall.AF_INET6:
		if ip16 := ip.To16(); ip16 != nil {
			sa := &syscall.SockaddrInet6{Port: port}
			copy(sa.Addr[:], ip16)
			return sa, nil
		}
		return nil, fmt.Errorf("invalid IPv6 address")
	default:
		return nil, fmt.Errorf("unsupported address family: %d", family)
	}
}

// parseSockaddr converts a syscall.Sockaddr (either *syscall.SockaddrInet4 or
// *syscall.SockaddrInet6) into an *SCTPAddr containing a single IP and port.
// Returns an error for unsupported sockaddr types.
func parseSockaddr(sa syscall.Sockaddr) (*SCTPAddr, error) {
	switch s := sa.(type) {
	case *syscall.SockaddrInet4:
		ip := net.IPv4(s.Addr[0], s.Addr[1], s.Addr[2], s.Addr[3])
		return &SCTPAddr{IPs: []net.IP{ip}, Port: s.Port}, nil
	case *syscall.SockaddrInet6:
		ip := make(net.IP, 16)
		copy(ip, s.Addr[:])
		return &SCTPAddr{IPs: []net.IP{ip}, Port: s.Port}, nil
	default:
		return nil, fmt.Errorf("unsupported sockaddr type: %T", sa)
	}
}

// IsSCTPSupported reports whether the platform appears to support SCTP by
// attempting to create a basic SCTP socket. It returns true if socket
// creation succeeds and false otherwise; it does not guarantee full SCTP
// feature availability beyond the ability to open a socket.
func IsSCTPSupported() bool {
	// Try to create an SCTP socket
	fd, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_STREAM, IPPROTO_SCTP)
	if err != nil {
		return false
	}
	syscall.Close(fd)
	return true
}

// GetSCTPInfo returns SCTP connection information
func (c *SCTPConn) GetSCTPInfo() (*SCTPInfo, error) {
	if c.fd < 0 {
		return nil, fmt.Errorf("invalid SCTP file descriptor")
	}

	// Try to get SCTP status using platform-specific syscall
	info := &SCTPInfo{
		Streams:    c.config.Streams,
		LocalAddr:  c.laddr,
		RemoteAddr: c.raddr,
	}

	// Attempt to get SCTP_STATUS information
	state, err := c.getSCTPState()
	if err != nil {
		// If we can't get state, check if connection is still valid
		if c.isConnected() {
			info.State = "ESTABLISHED"
		} else {
			info.State = "UNKNOWN"
		}
		logger.Debug("Could not retrieve SCTP state: %v", err)
	} else {
		info.State = state
	}

	// Get additional statistics if available
	if stats, err := c.getSCTPStats(); err == nil {
		info.RTO = stats.RTO
		info.MTU = stats.MTU
		info.UnackedData = stats.UnackedData
		info.InboundStreams = stats.InboundStreams
		info.OutboundStreams = stats.OutboundStreams
	}

	return info, nil
}

// SCTPInfo contains SCTP connection information
type SCTPInfo struct {
	State            string
	Streams          int
	LocalAddr        *SCTPAddr
	RemoteAddr       *SCTPAddr
	RTO              time.Duration // Retransmission Timeout
	MTU              int           // Maximum Transmission Unit
	UnackedData      int           // Unacknowledged data
	InboundStreams   int           // Number of inbound streams
	OutboundStreams  int           // Number of outbound streams
}

// String returns string representation of SCTP info
func (info *SCTPInfo) String() string {
	return fmt.Sprintf("SCTP Connection: %s -> %s (State: %s, Streams: %d)",
		info.LocalAddr.String(),
		info.RemoteAddr.String(),
		info.State,
		info.Streams)
}

// sctpStats holds internal SCTP statistics
type sctpStats struct {
	RTO              time.Duration
	MTU              int
	UnackedData      int
	InboundStreams   int
	OutboundStreams  int
}

// getSCTPState retrieves the current SCTP connection state using platform-specific syscalls
func (c *SCTPConn) getSCTPState() (string, error) {
	// SCTP state constants
	const (
		SCTP_EMPTY             = 0
		SCTP_CLOSED            = 1
		SCTP_COOKIE_WAIT       = 2
		SCTP_COOKIE_ECHOED     = 3
		SCTP_ESTABLISHED       = 4
		SCTP_SHUTDOWN_PENDING  = 5
		SCTP_SHUTDOWN_SENT     = 6
		SCTP_SHUTDOWN_RECEIVED = 7
		SCTP_SHUTDOWN_ACK_SENT = 8
	)

	// Try to get SCTP status via getsockopt
	// This is platform-specific and may not work on all systems
	var status [128]byte
	n := uint32(len(status))
	
	_, _, errno := syscall.Syscall6(
		syscall.SYS_GETSOCKOPT,
		uintptr(c.fd),
		uintptr(SOL_SCTP),
		uintptr(SCTP_STATUS),
		uintptr(unsafe.Pointer(&status[0])),
		uintptr(unsafe.Pointer(&n)),
		0,
	)
	
	if errno != 0 {
		return "", fmt.Errorf("getsockopt SCTP_STATUS failed: %v", errno)
	}

	// Parse state from status (platform-specific)
	// The actual structure depends on the OS
	stateValue := int(status[0])
	
	stateNames := map[int]string{
		SCTP_EMPTY:             "EMPTY",
		SCTP_CLOSED:            "CLOSED",
		SCTP_COOKIE_WAIT:       "COOKIE_WAIT",
		SCTP_COOKIE_ECHOED:     "COOKIE_ECHOED",
		SCTP_ESTABLISHED:       "ESTABLISHED",
		SCTP_SHUTDOWN_PENDING:  "SHUTDOWN_PENDING",
		SCTP_SHUTDOWN_SENT:     "SHUTDOWN_SENT",
		SCTP_SHUTDOWN_RECEIVED: "SHUTDOWN_RECEIVED",
		SCTP_SHUTDOWN_ACK_SENT: "SHUTDOWN_ACK_SENT",
	}
	
	if name, ok := stateNames[stateValue]; ok {
		return name, nil
	}
	
	return "UNKNOWN", nil
}

// isConnected checks if the SCTP connection is still valid
func (c *SCTPConn) isConnected() bool {
	if c.fd < 0 {
		return false
	}
	
	// Try to read socket error to check connection validity
	var err int
	errLen := uint32(4)
	
	_, _, errno := syscall.Syscall6(
		syscall.SYS_GETSOCKOPT,
		uintptr(c.fd),
		uintptr(syscall.SOL_SOCKET),
		uintptr(syscall.SO_ERROR),
		uintptr(unsafe.Pointer(&err)),
		uintptr(unsafe.Pointer(&errLen)),
		0,
	)
	
	return errno == 0 && err == 0
}

// getSCTPStats retrieves detailed SCTP statistics
func (c *SCTPConn) getSCTPStats() (*sctpStats, error) {
	stats := &sctpStats{
		InboundStreams:  c.config.Streams,
		OutboundStreams: c.config.Streams,
	}
	
	// Try to get RTO info
	var rtoInfo [12]byte // Platform-specific structure size
	n := uint32(len(rtoInfo))
	
	_, _, errno := syscall.Syscall6(
		syscall.SYS_GETSOCKOPT,
		uintptr(c.fd),
		uintptr(SOL_SCTP),
		uintptr(SCTP_RTOINFO),
		uintptr(unsafe.Pointer(&rtoInfo[0])),
		uintptr(unsafe.Pointer(&n)),
		0,
	)
	
	if errno == 0 {
		// Parse RTO (platform-specific, this is simplified)
		rtoMs := uint32(rtoInfo[0]) | uint32(rtoInfo[1])<<8 | 
		         uint32(rtoInfo[2])<<16 | uint32(rtoInfo[3])<<24
		stats.RTO = time.Duration(rtoMs) * time.Millisecond
	}
	
	// Try to get MTU/segment size
	var maxseg int
	maxsegLen := uint32(4)
	
	_, _, errno = syscall.Syscall6(
		syscall.SYS_GETSOCKOPT,
		uintptr(c.fd),
		uintptr(SOL_SCTP),
		uintptr(SCTP_MAXSEG),
		uintptr(unsafe.Pointer(&maxseg)),
		uintptr(unsafe.Pointer(&maxsegLen)),
		0,
	)
	
	if errno == 0 && maxseg > 0 {
		stats.MTU = maxseg
	} else {
		stats.MTU = 1500 // Default MTU
	}
	
	return stats, nil
}

// setSCTPTimeout sets both send and receive timeout
func (c *SCTPConn) setSCTPTimeout(timeout time.Duration) error {
	if err := c.setReceiveTimeout(timeout); err != nil {
		return err
	}
	return c.setSendTimeout(timeout)
}

// setReceiveTimeout sets the socket receive timeout
func (c *SCTPConn) setReceiveTimeout(timeout time.Duration) error {
	tv := syscall.NsecToTimeval(timeout.Nanoseconds())
	
	_, _, errno := syscall.Syscall6(
		syscall.SYS_SETSOCKOPT,
		uintptr(c.fd),
		uintptr(syscall.SOL_SOCKET),
		uintptr(syscall.SO_RCVTIMEO),
		uintptr(unsafe.Pointer(&tv)),
		unsafe.Sizeof(tv),
		0,
	)
	
	if errno != 0 {
		return fmt.Errorf("failed to set receive timeout: %v", errno)
	}
	
	return nil
}

// setSendTimeout sets the socket send timeout
func (c *SCTPConn) setSendTimeout(timeout time.Duration) error {
	tv := syscall.NsecToTimeval(timeout.Nanoseconds())
	
	_, _, errno := syscall.Syscall6(
		syscall.SYS_SETSOCKOPT,
		uintptr(c.fd),
		uintptr(syscall.SOL_SOCKET),
		uintptr(syscall.SO_SNDTIMEO),
		uintptr(unsafe.Pointer(&tv)),
		unsafe.Sizeof(tv),
		0,
	)
	
	if errno != 0 {
		return fmt.Errorf("failed to set send timeout: %v", errno)
	}
	
	return nil
}
