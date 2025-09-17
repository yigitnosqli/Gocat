package network

import (
	"context"
	"fmt"
	"net"
	"syscall"
	"time"

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
	fd     int
	laddr  *SCTPAddr
	raddr  *SCTPAddr
	config *SCTPConfig
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
	// SCTP doesn't support deadlines in the same way as TCP
	// This is a placeholder implementation
	logger.Warn("SCTP SetDeadline not fully implemented")
	return nil
}

// SetReadDeadline sets read deadline
func (c *SCTPConn) SetReadDeadline(t time.Time) error {
	logger.Warn("SCTP SetReadDeadline not fully implemented")
	return nil
}

// SetWriteDeadline sets write deadline
func (c *SCTPConn) SetWriteDeadline(t time.Time) error {
	logger.Warn("SCTP SetWriteDeadline not fully implemented")
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
	// This would require platform-specific implementation
	// For now, return basic info
	return &SCTPInfo{
		State:      "ESTABLISHED", // Placeholder
		Streams:    c.config.Streams,
		LocalAddr:  c.laddr,
		RemoteAddr: c.raddr,
	}, nil
}

// SCTPInfo contains SCTP connection information
type SCTPInfo struct {
	State      string
	Streams    int
	LocalAddr  *SCTPAddr
	RemoteAddr *SCTPAddr
}

// String returns string representation of SCTP info
func (info *SCTPInfo) String() string {
	return fmt.Sprintf("SCTP Connection: %s -> %s (State: %s, Streams: %d)",
		info.LocalAddr.String(),
		info.RemoteAddr.String(),
		info.State,
		info.Streams)
}
