package network

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/ibrahmsql/gocat/internal/errors"
	"github.com/ibrahmsql/gocat/internal/logger"
	"github.com/ibrahmsql/gocat/internal/security"
)

// ConnectionType represents the type of network connection
type ConnectionType string

const (
	ConnectionTypeTCP  ConnectionType = "tcp"
	ConnectionTypeUDP  ConnectionType = "udp"
	ConnectionTypeTLS  ConnectionType = "tls"
	ConnectionTypeUnix ConnectionType = "unix"
)

// ConnectionOptions holds configuration for network connections
type ConnectionOptions struct {
	Host               string
	Port               int
	Protocol           ConnectionType
	Timeout            time.Duration
	KeepAlive          time.Duration
	TLSConfig          *tls.Config
	BindAddress        string
	BindPort           int
	ReuseAddr          bool
	ReusePort          bool
	NoDelay            bool
	BufferSize         int
	MaxConnections     int
	ConnectionPoolSize int
	RetryAttempts      int
	RetryDelay         time.Duration
	IPv6               bool
	IPv4               bool
}

// DefaultConnectionOptions returns default connection options
func DefaultConnectionOptions() *ConnectionOptions {
	return &ConnectionOptions{
		Timeout:            30 * time.Second,
		KeepAlive:          30 * time.Second,
		BufferSize:         4096,
		MaxConnections:     100,
		ConnectionPoolSize: 10,
		RetryAttempts:      3,
		RetryDelay:         time.Second,
		IPv4:               true,
		IPv6:               true,
		NoDelay:            true,
		ReuseAddr:          true,
	}
}

// Connection represents a network connection with enhanced features
type Connection struct {
	conn         net.Conn
	options      *ConnectionOptions
	validator    *security.InputValidator
	mu           sync.RWMutex
	closed       bool
	connectedAt  time.Time
	bytesRead    int64
	bytesWritten int64
	lastActivity time.Time
	ctx          context.Context
	cancel       context.CancelFunc
}

// NewConnection creates a new enhanced connection
func NewConnection(conn net.Conn, options *ConnectionOptions) *Connection {
	if options == nil {
		options = DefaultConnectionOptions()
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &Connection{
		conn:         conn,
		options:      options,
		validator:    security.NewInputValidator(),
		connectedAt:  time.Now(),
		lastActivity: time.Now(),
		ctx:          ctx,
		cancel:       cancel,
	}
}

// Read reads data from the connection
func (c *Connection) Read(b []byte) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return 0, errors.NetworkError("NET007", "Connection is closed").WithUserFriendly("Connection has been closed")
	}

	// Set read deadline if configured
	if c.options.Timeout > 0 {
		if err := c.conn.SetReadDeadline(time.Now().Add(c.options.Timeout)); err != nil {
			return 0, errors.WrapError(err, errors.ErrorTypeNetwork, errors.SeverityMedium, "NET011", "Failed to set read deadline")
		}
	}

	n, err := c.conn.Read(b)
	if err != nil {
		// Mark connection as closed on certain errors
		if isConnectionClosedError(err) {
			c.closed = true
		}
		return n, errors.WrapError(err, errors.ErrorTypeNetwork, errors.SeverityHigh, "NET008", "Failed to read from connection")
	}

	c.bytesRead += int64(n)
	c.lastActivity = time.Now()
	return n, nil
}

// Write writes data to the connection
func (c *Connection) Write(b []byte) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return 0, errors.NetworkError("NET009", "Connection is closed").WithUserFriendly("Connection has been closed")
	}

	// Set write deadline if configured
	if c.options.Timeout > 0 {
		if err := c.conn.SetWriteDeadline(time.Now().Add(c.options.Timeout)); err != nil {
			return 0, errors.WrapError(err, errors.ErrorTypeNetwork, errors.SeverityMedium, "NET012", "Failed to set write deadline")
		}
	}

	n, err := c.conn.Write(b)
	if err != nil {
		// Mark connection as closed on certain errors
		if isConnectionClosedError(err) {
			c.closed = true
		}
		return n, errors.WrapError(err, errors.ErrorTypeNetwork, errors.SeverityHigh, "NET010", "Failed to write to connection")
	}

	c.bytesWritten += int64(n)
	c.lastActivity = time.Now()
	return n, nil
}

// Close closes the connection
func (c *Connection) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil
	}

	c.closed = true
	c.cancel() // Cancel context to signal shutdown
	return c.conn.Close()
}

// LocalAddr returns the local network address
func (c *Connection) LocalAddr() net.Addr {
	return c.conn.LocalAddr()
}

// RemoteAddr returns the remote network address
func (c *Connection) RemoteAddr() net.Addr {
	return c.conn.RemoteAddr()
}

// SetDeadline sets the read and write deadlines
func (c *Connection) SetDeadline(t time.Time) error {
	return c.conn.SetDeadline(t)
}

// SetReadDeadline sets the deadline for future Read calls
func (c *Connection) SetReadDeadline(t time.Time) error {
	return c.conn.SetReadDeadline(t)
}

// SetWriteDeadline sets the deadline for future Write calls
func (c *Connection) SetWriteDeadline(t time.Time) error {
	return c.conn.SetWriteDeadline(t)
}

// Stats returns connection statistics
func (c *Connection) Stats() ConnectionStats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return ConnectionStats{
		ConnectedAt:  c.connectedAt,
		LastActivity: c.lastActivity,
		BytesRead:    c.bytesRead,
		BytesWritten: c.bytesWritten,
		Duration:     time.Since(c.connectedAt),
		LocalAddr:    c.LocalAddr().String(),
		RemoteAddr:   c.RemoteAddr().String(),
		Closed:       c.closed,
	}
}

// Context returns the connection's context
func (c *Connection) Context() context.Context {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.ctx
}

// isConnectionClosedError checks if the error indicates a closed connection
func isConnectionClosedError(err error) bool {
	if err == nil {
		return false
	}

	// Check for common connection closed errors
	if err == io.EOF {
		return true
	}

	// Check for network errors
	if netErr, ok := err.(net.Error); ok {
		if netErr.Timeout() {
			return false // Timeout is not necessarily a closed connection
		}
	}

	// Check for syscall errors
	if opErr, ok := err.(*net.OpError); ok {
		if opErr.Err == syscall.ECONNRESET || opErr.Err == syscall.EPIPE {
			return true
		}
	}

	// Check error message for common patterns
	errorMsg := strings.ToLower(err.Error())
	return strings.Contains(errorMsg, "connection reset") ||
		strings.Contains(errorMsg, "broken pipe") ||
		strings.Contains(errorMsg, "connection closed") ||
		strings.Contains(errorMsg, "use of closed network connection")
}

// ConnectionStats holds connection statistics
type ConnectionStats struct {
	ConnectedAt  time.Time     `json:"connected_at"`
	LastActivity time.Time     `json:"last_activity"`
	BytesRead    int64         `json:"bytes_read"`
	BytesWritten int64         `json:"bytes_written"`
	Duration     time.Duration `json:"duration"`
	LocalAddr    string        `json:"local_addr"`
	RemoteAddr   string        `json:"remote_addr"`
	Closed       bool          `json:"closed"`
}

// Dialer provides enhanced dialing capabilities
type Dialer struct {
	options   *ConnectionOptions
	validator *security.InputValidator
	logger    *logger.Logger
}

// NewDialer creates a new enhanced dialer
func NewDialer(options *ConnectionOptions) *Dialer {
	if options == nil {
		options = DefaultConnectionOptions()
	}

	return &Dialer{
		options:   options,
		validator: security.NewInputValidator(),
		logger:    logger.GetDefaultLogger(),
	}
}

// Dial connects to the specified address
func (d *Dialer) Dial(ctx context.Context, address string) (*Connection, error) {
	host, portStr, err := net.SplitHostPort(address)
	if err != nil {
		return nil, errors.ValidationError("VAL006", fmt.Sprintf("Invalid address format: %s", address)).WithSuggestion("Use format 'host:port'")
	}

	// Validate hostname
	if err := d.validator.ValidateHostname(host); err != nil {
		return nil, errors.WrapError(err, errors.ErrorTypeValidation, errors.SeverityMedium, "VAL007", "Invalid hostname")
	}

	// Validate port
	port, err := d.validator.ValidatePort(portStr)
	if err != nil {
		return nil, errors.WrapError(err, errors.ErrorTypeValidation, errors.SeverityMedium, "VAL008", "Invalid port")
	}

	d.options.Host = host
	d.options.Port = port

	return d.dialWithRetry(ctx)
}

// dialWithRetry attempts to dial with retry logic
func (d *Dialer) dialWithRetry(ctx context.Context) (*Connection, error) {
	var lastErr error

	for attempt := 0; attempt <= d.options.RetryAttempts; attempt++ {
		if attempt > 0 {
			d.logger.DebugWithFields("Retrying connection", map[string]interface{}{
				"attempt": attempt,
				"host":    d.options.Host,
				"port":    d.options.Port,
			})

			select {
			case <-ctx.Done():
				return nil, errors.TimeoutError("NET011", "Connection cancelled").WithCause(ctx.Err())
			case <-time.After(d.options.RetryDelay):
			}
		}

		conn, err := d.dialOnce(ctx)
		if err == nil {
			return conn, nil
		}

		lastErr = err
		if !errors.IsRetryable(err) {
			break
		}
	}

	return nil, lastErr
}

// dialOnce performs a single dial attempt
func (d *Dialer) dialOnce(ctx context.Context) (*Connection, error) {
	address := net.JoinHostPort(d.options.Host, strconv.Itoa(d.options.Port))

	dialer := &net.Dialer{
		Timeout:   d.options.Timeout,
		KeepAlive: d.options.KeepAlive,
	}

	// Set local address if specified
	if d.options.BindAddress != "" {
		localAddr := &net.TCPAddr{
			IP:   net.ParseIP(d.options.BindAddress),
			Port: d.options.BindPort,
		}
		dialer.LocalAddr = localAddr
	}

	var conn net.Conn
	var err error

	switch d.options.Protocol {
	case ConnectionTypeTCP:
		conn, err = dialer.DialContext(ctx, "tcp", address)
	case ConnectionTypeUDP:
		conn, err = dialer.DialContext(ctx, "udp", address)
	case ConnectionTypeTLS:
		tlsConfig := d.options.TLSConfig
		if tlsConfig == nil {
			tlsConfig = &tls.Config{
				ServerName: d.options.Host,
			}
		}
		conn, err = tls.DialWithDialer(dialer, "tcp", address, tlsConfig)
	case ConnectionTypeUnix:
		conn, err = dialer.DialContext(ctx, "unix", d.options.Host)
	default:
		return nil, errors.ValidationError("VAL009", fmt.Sprintf("Unsupported protocol: %s", d.options.Protocol))
	}

	if err != nil {
		return nil, d.wrapDialError(err)
	}

	// Configure TCP-specific options
	if tcpConn, ok := conn.(*net.TCPConn); ok {
		if d.options.NoDelay {
			tcpConn.SetNoDelay(true)
		}
	}

	d.logger.InfoWithFields("Connection established", map[string]interface{}{
		"local_addr":  conn.LocalAddr().String(),
		"remote_addr": conn.RemoteAddr().String(),
		"protocol":    d.options.Protocol,
	})

	return NewConnection(conn, d.options), nil
}

// wrapDialError wraps dial errors with appropriate error types
func (d *Dialer) wrapDialError(err error) error {
	errStr := err.Error()

	switch {
	case strings.Contains(errStr, "connection refused"):
		return errors.WrapError(err, errors.ErrorTypeNetwork, errors.SeverityHigh, "NET012", "Connection refused").SetRetryable(true).WithSuggestion("Check if the service is running and accessible")
	case strings.Contains(errStr, "timeout"):
		return errors.WrapError(err, errors.ErrorTypeTimeout, errors.SeverityHigh, "NET013", "Connection timeout").SetRetryable(true).WithSuggestion("Check network connectivity and increase timeout")
	case strings.Contains(errStr, "no route to host"):
		return errors.WrapError(err, errors.ErrorTypeNetwork, errors.SeverityHigh, "NET014", "Host unreachable").WithSuggestion("Check network routing and firewall settings")
	case strings.Contains(errStr, "network is unreachable"):
		return errors.WrapError(err, errors.ErrorTypeNetwork, errors.SeverityHigh, "NET015", "Network unreachable").WithSuggestion("Check network configuration")
	case strings.Contains(errStr, "no such host"):
		return errors.WrapError(err, errors.ErrorTypeNetwork, errors.SeverityMedium, "NET016", "DNS resolution failed").WithSuggestion("Check hostname and DNS configuration")
	default:
		return errors.WrapError(err, errors.ErrorTypeNetwork, errors.SeverityHigh, "NET017", "Connection failed").SetRetryable(true)
	}
}

// Listener provides enhanced listening capabilities
type Listener struct {
	listener  net.Listener
	options   *ConnectionOptions
	validator *security.InputValidator
	logger    *logger.Logger
	mu        sync.RWMutex
	closed    bool
	stats     ListenerStats
}

// ListenerStats holds listener statistics
type ListenerStats struct {
	StartedAt         time.Time `json:"started_at"`
	ConnectionsTotal  int64     `json:"connections_total"`
	ConnectionsActive int64     `json:"connections_active"`
	BytesRead         int64     `json:"bytes_read"`
	BytesWritten      int64     `json:"bytes_written"`
}

// NewListener creates a new enhanced listener
func NewListener(listener net.Listener, options *ConnectionOptions) *Listener {
	if options == nil {
		options = DefaultConnectionOptions()
	}

	return &Listener{
		listener:  listener,
		options:   options,
		validator: security.NewInputValidator(),
		logger:    logger.GetDefaultLogger(),
		stats: ListenerStats{
			StartedAt: time.Now(),
		},
	}
}

// Accept waits for and returns the next connection
func (l *Listener) Accept() (*Connection, error) {
	l.mu.RLock()
	closed := l.closed
	l.mu.RUnlock()

	if closed {
		return nil, errors.NetworkError("NET018", "Listener is closed")
	}

	conn, err := l.listener.Accept()
	if err != nil {
		return nil, errors.WrapError(err, errors.ErrorTypeNetwork, errors.SeverityHigh, "NET019", "Failed to accept connection")
	}

	l.mu.Lock()
	l.stats.ConnectionsTotal++
	l.stats.ConnectionsActive++
	l.mu.Unlock()

	l.logger.InfoWithFields("Connection accepted", map[string]interface{}{
		"remote_addr": conn.RemoteAddr().String(),
		"local_addr":  conn.LocalAddr().String(),
	})

	return NewConnection(conn, l.options), nil
}

// Close closes the listener
func (l *Listener) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.closed {
		return nil
	}

	l.closed = true
	return l.listener.Close()
}

// Addr returns the listener's network address
func (l *Listener) Addr() net.Addr {
	return l.listener.Addr()
}

// Stats returns listener statistics
func (l *Listener) Stats() ListenerStats {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.stats
}

// Listen creates a new listener on the specified address
func Listen(ctx context.Context, address string, options *ConnectionOptions) (*Listener, error) {
	if options == nil {
		options = DefaultConnectionOptions()
	}

	validator := security.NewInputValidator()

	host, portStr, err := net.SplitHostPort(address)
	if err != nil {
		return nil, errors.ValidationError("VAL010", fmt.Sprintf("Invalid address format: %s", address)).WithSuggestion("Use format 'host:port'")
	}

	// Validate hostname
	if host != "" {
		if err := validator.ValidateHostname(host); err != nil {
			return nil, errors.WrapError(err, errors.ErrorTypeValidation, errors.SeverityMedium, "VAL011", "Invalid hostname")
		}
	}

	// Validate port
	port, err := validator.ValidatePort(portStr)
	if err != nil {
		return nil, errors.WrapError(err, errors.ErrorTypeValidation, errors.SeverityMedium, "VAL012", "Invalid port")
	}

	options.Host = host
	options.Port = port

	var listener net.Listener

	switch options.Protocol {
	case ConnectionTypeTCP:
		listener, err = net.Listen("tcp", address)
	case ConnectionTypeUDP:
		return nil, errors.ValidationError("VAL013", "UDP listening not supported with this method").WithSuggestion("Use ListenUDP for UDP connections")
	case ConnectionTypeTLS:
		if options.TLSConfig == nil {
			return nil, errors.ConfigError("CFG004", "TLS configuration required for TLS listener")
		}
		listener, err = tls.Listen("tcp", address, options.TLSConfig)
	case ConnectionTypeUnix:
		listener, err = net.Listen("unix", host)
	default:
		return nil, errors.ValidationError("VAL014", fmt.Sprintf("Unsupported protocol: %s", options.Protocol))
	}

	if err != nil {
		return nil, errors.WrapError(err, errors.ErrorTypeNetwork, errors.SeverityHigh, "NET020", "Failed to create listener")
	}

	logger.GetDefaultLogger().InfoWithFields("Listener created", map[string]interface{}{
		"address":  address,
		"protocol": options.Protocol,
	})

	return NewListener(listener, options), nil
}
