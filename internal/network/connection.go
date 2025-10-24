package network

import (
	"compress/gzip"
	"context"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ibrahmsql/gocat/internal/errors"
)

// Connection interface extends net.Conn with additional features
type Connection interface {
	net.Conn
	Stats() ConnectionStats
	Context() context.Context
	SetMetrics(collector MetricsCollector) error
	EnableCompression() error
	SetRateLimit(bytesPerSecond int64) error
	IsHealthy() bool
	GetID() string
}

// MetricsCollector interface for connection metrics
type MetricsCollector interface {
	IncrementCounter(name string, tags map[string]string)
	RecordGauge(name string, value float64, tags map[string]string)
	RecordHistogram(name string, value float64, tags map[string]string)
	RecordTimer(name string, duration time.Duration, tags map[string]string)
}

// ConnectionStats holds detailed connection statistics
type ConnectionStats struct {
	ID               string        `json:"id"`
	LocalAddr        string        `json:"local_addr"`
	RemoteAddr       string        `json:"remote_addr"`
	Protocol         string        `json:"protocol"`
	State            string        `json:"state"`
	ConnectedAt      time.Time     `json:"connected_at"`
	LastActivity     time.Time     `json:"last_activity"`
	BytesRead        int64         `json:"bytes_read"`
	BytesWritten     int64         `json:"bytes_written"`
	Duration         time.Duration `json:"duration"`
	ErrorCount       int64         `json:"error_count"`
	LastError        string        `json:"last_error,omitempty"`
	ReadOperations   int64         `json:"read_operations"`
	WriteOperations  int64         `json:"write_operations"`
	CompressionRatio float64       `json:"compression_ratio,omitempty"`
	RateLimitActive  bool          `json:"rate_limit_active"`
}

// ConnectionState represents the current state of a connection
type ConnectionState string

const (
	StateConnecting ConnectionState = "connecting"
	StateConnected  ConnectionState = "connected"
	StateClosing    ConnectionState = "closing"
	StateClosed     ConnectionState = "closed"
	StateError      ConnectionState = "error"
)

// ConnectionImpl implements the Connection interface
type ConnectionImpl struct {
	conn               net.Conn
	id                 string
	protocol           string
	state              ConnectionState
	connectedAt        time.Time
	lastActivity       time.Time
	bytesRead          int64
	bytesWritten       int64
	errorCount         int64
	readOps            int64
	writeOps           int64
	lastError          string
	compressionEnabled bool
	compressionRatio   float64
	rateLimitEnabled   bool
	rateLimitBPS       int64
	rateLimitBucket    *TokenBucket
	metricsCollector   MetricsCollector
	ctx                context.Context
	cancel             context.CancelFunc
	mu                 sync.RWMutex
	healthMu           sync.RWMutex
	healthy            bool
	lastHealthCheck    time.Time
}

// TokenBucket implements a simple token bucket for rate limiting
type TokenBucket struct {
	capacity   int64
	tokens     int64
	refillRate int64
	lastRefill time.Time
	mu         sync.Mutex
}

// NewTokenBucket creates a new token bucket
func NewTokenBucket(capacity, refillRate int64) *TokenBucket {
	return &TokenBucket{
		capacity:   capacity,
		tokens:     capacity,
		refillRate: refillRate,
		lastRefill: time.Now(),
	}
}

// Allow checks if the operation is allowed and consumes tokens
func (tb *TokenBucket) Allow(tokens int64) bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(tb.lastRefill)

	// Refill tokens based on elapsed time
	tokensToAdd := int64(elapsed.Seconds()) * tb.refillRate
	tb.tokens = min(tb.capacity, tb.tokens+tokensToAdd)
	tb.lastRefill = now

	if tb.tokens >= tokens {
		tb.tokens -= tokens
		return true
	}
	return false
}

// min returns the minimum of two int64 values
func min(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

// NewConnection creates a new  connection
func NewConnection(conn net.Conn, protocol string) *ConnectionImpl {
	ctx, cancel := context.WithCancel(context.Background())

	id := generateConnectionID()

	ec := &ConnectionImpl{
		conn:            conn,
		id:              id,
		protocol:        protocol,
		state:           StateConnected,
		connectedAt:     time.Now(),
		lastActivity:    time.Now(),
		ctx:             ctx,
		cancel:          cancel,
		healthy:         true,
		lastHealthCheck: time.Now(),
	}

	// Start health monitoring
	go ec.healthMonitor()

	return ec
}

// generateConnectionID generates a unique connection ID
func generateConnectionID() string {
	return time.Now().Format("20060102150405") + "-" + randomString(8)
}

// randomString generates a random string of specified length
func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(b)
}

// Read implements net.Conn.Read with  features
func (ec *ConnectionImpl) Read(b []byte) (int, error) {
	ec.mu.Lock()
	if ec.state == StateClosed || ec.state == StateClosing {
		ec.mu.Unlock()
		return 0, errors.NetworkError("NET021", "Connection is closed").WithUserFriendly("Connection has been closed")
	}
	ec.mu.Unlock()

	// Check context cancellation
	select {
	case <-ec.ctx.Done():
		ec.setState(StateClosed)
		return 0, errors.NetworkError("NET022", "Connection context cancelled").WithUserFriendly("Connection was cancelled")
	default:
	}

	// Apply rate limiting if enabled
	if ec.rateLimitEnabled && ec.rateLimitBucket != nil {
		if !ec.rateLimitBucket.Allow(int64(len(b))) {
			// Wait a bit and try again
			time.Sleep(10 * time.Millisecond)
			if !ec.rateLimitBucket.Allow(int64(len(b))) {
				return 0, errors.NetworkError("NET023", "Rate limit exceeded").WithUserFriendly("Connection rate limit exceeded")
			}
		}
	}

	n, err := ec.conn.Read(b)

	// Update statistics
	atomic.AddInt64(&ec.bytesRead, int64(n))
	atomic.AddInt64(&ec.readOps, 1)
	ec.updateLastActivity()

	if err != nil {
		atomic.AddInt64(&ec.errorCount, 1)
		ec.setLastError(err.Error())

		if isConnectionClosedError(err) {
			ec.setState(StateClosed)
			ec.cancel()
		}

		// Record metrics
		if ec.metricsCollector != nil {
			ec.metricsCollector.IncrementCounter("connection_read_errors", map[string]string{
				"connection_id": ec.id,
				"protocol":      ec.protocol,
			})
		}

		return n, errors.WrapError(err, errors.ErrorTypeNetwork, errors.SeverityHigh, "NET024", "Failed to read from connection")
	}

	// Record successful read metrics
	if ec.metricsCollector != nil {
		ec.metricsCollector.RecordHistogram("connection_bytes_read", float64(n), map[string]string{
			"connection_id": ec.id,
			"protocol":      ec.protocol,
		})
	}

	return n, nil
}

// Write implements net.Conn.Write with  features
func (ec *ConnectionImpl) Write(b []byte) (int, error) {
	ec.mu.Lock()
	if ec.state == StateClosed || ec.state == StateClosing {
		ec.mu.Unlock()
		return 0, errors.NetworkError("NET025", "Connection is closed").WithUserFriendly("Connection has been closed")
	}
	ec.mu.Unlock()

	// Check context cancellation
	select {
	case <-ec.ctx.Done():
		ec.setState(StateClosed)
		return 0, errors.NetworkError("NET026", "Connection context cancelled").WithUserFriendly("Connection was cancelled")
	default:
	}

	// Apply rate limiting if enabled
	if ec.rateLimitEnabled && ec.rateLimitBucket != nil {
		if !ec.rateLimitBucket.Allow(int64(len(b))) {
			// Wait a bit and try again
			time.Sleep(10 * time.Millisecond)
			if !ec.rateLimitBucket.Allow(int64(len(b))) {
				return 0, errors.NetworkError("NET027", "Rate limit exceeded").WithUserFriendly("Connection rate limit exceeded")
			}
		}
	}

	n, err := ec.conn.Write(b)

	// Update statistics
	atomic.AddInt64(&ec.bytesWritten, int64(n))
	atomic.AddInt64(&ec.writeOps, 1)
	ec.updateLastActivity()

	if err != nil {
		atomic.AddInt64(&ec.errorCount, 1)
		ec.setLastError(err.Error())

		if isConnectionClosedError(err) {
			ec.setState(StateClosed)
			ec.cancel()
		}

		// Record metrics
		if ec.metricsCollector != nil {
			ec.metricsCollector.IncrementCounter("connection_write_errors", map[string]string{
				"connection_id": ec.id,
				"protocol":      ec.protocol,
			})
		}

		return n, errors.WrapError(err, errors.ErrorTypeNetwork, errors.SeverityHigh, "NET028", "Failed to write to connection")
	}

	// Record successful write metrics
	if ec.metricsCollector != nil {
		ec.metricsCollector.RecordHistogram("connection_bytes_written", float64(n), map[string]string{
			"connection_id": ec.id,
			"protocol":      ec.protocol,
		})
	}

	return n, nil
}

// Close implements net.Conn.Close with  cleanup
func (ec *ConnectionImpl) Close() error {
	ec.mu.Lock()
	if ec.state == StateClosed {
		ec.mu.Unlock()
		return nil
	}

	ec.state = StateClosing
	ec.mu.Unlock()

	// Cancel context to signal shutdown
	ec.cancel()

	// Close underlying connection
	err := ec.conn.Close()

	ec.setState(StateClosed)

	// Record metrics
	if ec.metricsCollector != nil {
		duration := time.Since(ec.connectedAt)
		ec.metricsCollector.RecordTimer("connection_duration", duration, map[string]string{
			"connection_id": ec.id,
			"protocol":      ec.protocol,
		})
		ec.metricsCollector.IncrementCounter("connection_closed", map[string]string{
			"connection_id": ec.id,
			"protocol":      ec.protocol,
		})
	}

	if err != nil {
		return errors.WrapError(err, errors.ErrorTypeNetwork, errors.SeverityMedium, "NET029", "Failed to close connection")
	}

	return nil
}

// LocalAddr implements net.Conn.LocalAddr
func (ec *ConnectionImpl) LocalAddr() net.Addr {
	return ec.conn.LocalAddr()
}

// RemoteAddr implements net.Conn.RemoteAddr
func (ec *ConnectionImpl) RemoteAddr() net.Addr {
	return ec.conn.RemoteAddr()
}

// SetDeadline implements net.Conn.SetDeadline
func (ec *ConnectionImpl) SetDeadline(t time.Time) error {
	return ec.conn.SetDeadline(t)
}

// SetReadDeadline implements net.Conn.SetReadDeadline
func (ec *ConnectionImpl) SetReadDeadline(t time.Time) error {
	return ec.conn.SetReadDeadline(t)
}

// SetWriteDeadline implements net.Conn.SetWriteDeadline
func (ec *ConnectionImpl) SetWriteDeadline(t time.Time) error {
	return ec.conn.SetWriteDeadline(t)
}

// Stats returns detailed connection statistics
func (ec *ConnectionImpl) Stats() ConnectionStats {
	ec.mu.RLock()
	defer ec.mu.RUnlock()

	return ConnectionStats{
		ID:               ec.id,
		LocalAddr:        ec.conn.LocalAddr().String(),
		RemoteAddr:       ec.conn.RemoteAddr().String(),
		Protocol:         ec.protocol,
		State:            string(ec.state),
		ConnectedAt:      ec.connectedAt,
		LastActivity:     ec.lastActivity,
		BytesRead:        atomic.LoadInt64(&ec.bytesRead),
		BytesWritten:     atomic.LoadInt64(&ec.bytesWritten),
		Duration:         time.Since(ec.connectedAt),
		ErrorCount:       atomic.LoadInt64(&ec.errorCount),
		LastError:        ec.lastError,
		ReadOperations:   atomic.LoadInt64(&ec.readOps),
		WriteOperations:  atomic.LoadInt64(&ec.writeOps),
		CompressionRatio: ec.compressionRatio,
		RateLimitActive:  ec.rateLimitEnabled,
	}
}

// Context returns the connection's context
func (ec *ConnectionImpl) Context() context.Context {
	return ec.ctx
}

// SetMetrics sets the metrics collector for the connection
func (ec *ConnectionImpl) SetMetrics(collector MetricsCollector) error {
	if collector == nil {
		return errors.ValidationError("VAL015", "Metrics collector cannot be nil")
	}

	ec.mu.Lock()
	ec.metricsCollector = collector
	ec.mu.Unlock()

	return nil
}

// EnableCompression enables compression for the connection
// It wraps the underlying connection with gzip compression for both read and write operations
func (ec *ConnectionImpl) EnableCompression() error {
	ec.mu.Lock()
	defer ec.mu.Unlock()

	if ec.compressionEnabled {
		return errors.ValidationError("VAL017", "Compression is already enabled")
	}

	// Wrap the connection with compression
	compressedConn, err := NewCompressedConn(ec.conn)
	if err != nil {
		return errors.NetworkError("CONN010", "Failed to enable compression")
	}

	ec.conn = compressedConn
	ec.compressionEnabled = true
	ec.compressionRatio = 0.0 // Will be calculated during actual compression

	return nil
}

// SetRateLimit sets the rate limit for the connection
func (ec *ConnectionImpl) SetRateLimit(bytesPerSecond int64) error {
	if bytesPerSecond <= 0 {
		return errors.ValidationError("VAL016", "Rate limit must be positive")
	}

	ec.mu.Lock()
	defer ec.mu.Unlock()

	ec.rateLimitEnabled = true
	ec.rateLimitBPS = bytesPerSecond
	ec.rateLimitBucket = NewTokenBucket(bytesPerSecond, bytesPerSecond)

	return nil
}

// IsHealthy returns the current health status of the connection
func (ec *ConnectionImpl) IsHealthy() bool {
	ec.healthMu.RLock()
	defer ec.healthMu.RUnlock()
	return ec.healthy
}

// GetID returns the unique connection ID
func (ec *ConnectionImpl) GetID() string {
	return ec.id
}

// setState safely updates the connection state
func (ec *ConnectionImpl) setState(state ConnectionState) {
	ec.mu.Lock()
	ec.state = state
	ec.mu.Unlock()
}

// updateLastActivity updates the last activity timestamp
func (ec *ConnectionImpl) updateLastActivity() {
	ec.mu.Lock()
	ec.lastActivity = time.Now()
	ec.mu.Unlock()
}

// setLastError safely sets the last error
func (ec *ConnectionImpl) setLastError(errMsg string) {
	ec.mu.Lock()
	ec.lastError = errMsg
	ec.mu.Unlock()
}

// healthMonitor runs periodic health checks on the connection
func (ec *ConnectionImpl) healthMonitor() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ec.ctx.Done():
			return
		case <-ticker.C:
			ec.performHealthCheck()
		}
	}
}

// performHealthCheck performs a health check on the connection
func (ec *ConnectionImpl) performHealthCheck() {
	ec.healthMu.Lock()
	defer ec.healthMu.Unlock()

	// Simple health check based on connection state and recent activity
	now := time.Now()
	ec.lastHealthCheck = now

	// Check if connection is in a valid state
	ec.mu.RLock()
	state := ec.state
	lastActivity := ec.lastActivity
	ec.mu.RUnlock()

	if state == StateClosed || state == StateError {
		ec.healthy = false
		return
	}

	// Check if connection has been inactive for too long (5 minutes)
	if now.Sub(lastActivity) > 5*time.Minute {
		ec.healthy = false
		return
	}

	ec.healthy = true
}

// CompressedConnection wraps a net.Conn with compression
type CompressedConnection struct {
	conn   net.Conn
	reader io.Reader
	writer io.Writer
	level  int
}

// NewCompressedConnection creates a new compressed connection wrapper
func NewCompressedConnection(conn net.Conn, level int) *CompressedConnection {
	if level < 1 || level > 9 {
		level = 6 // Default compression level
	}

	return &CompressedConnection{
		conn:  conn,
		level: level,
	}
}

// NewCompressedConn creates a new compressed connection with default compression level
func NewCompressedConn(conn net.Conn) (net.Conn, error) {
	if conn == nil {
		return nil, errors.ValidationError("VAL018", "Connection cannot be nil")
	}
	return NewCompressedConnection(conn, 6), nil
}

// Read implements net.Conn.Read with decompression
func (c *CompressedConnection) Read(b []byte) (n int, err error) {
	if c.reader == nil {
		c.reader, err = gzip.NewReader(c.conn)
		if err != nil {
			return 0, err
		}
	}
	return c.reader.Read(b)
}

// Write implements net.Conn.Write with compression
func (c *CompressedConnection) Write(b []byte) (n int, err error) {
	if c.writer == nil {
		gzWriter, err := gzip.NewWriterLevel(c.conn, c.level)
		if err != nil {
			return 0, err
		}
		c.writer = gzWriter
	}

	n, err = c.writer.Write(b)
	if err != nil {
		return n, err
	}

	// Flush the gzip writer to ensure data is sent
	if gzWriter, ok := c.writer.(*gzip.Writer); ok {
		err = gzWriter.Flush()
	}

	return n, err
}

// Close implements net.Conn.Close
func (c *CompressedConnection) Close() error {
	// Close the gzip writer if it exists
	if gzWriter, ok := c.writer.(*gzip.Writer); ok {
		gzWriter.Close()
	}

	// Close the gzip reader if it exists
	if gzReader, ok := c.reader.(*gzip.Reader); ok {
		gzReader.Close()
	}

	return c.conn.Close()
}

// LocalAddr implements net.Conn.LocalAddr
func (c *CompressedConnection) LocalAddr() net.Addr {
	return c.conn.LocalAddr()
}

// RemoteAddr implements net.Conn.RemoteAddr
func (c *CompressedConnection) RemoteAddr() net.Addr {
	return c.conn.RemoteAddr()
}

// SetDeadline implements net.Conn.SetDeadline
func (c *CompressedConnection) SetDeadline(t time.Time) error {
	return c.conn.SetDeadline(t)
}

// SetReadDeadline implements net.Conn.SetReadDeadline
func (c *CompressedConnection) SetReadDeadline(t time.Time) error {
	return c.conn.SetReadDeadline(t)
}

// SetWriteDeadline implements net.Conn.SetWriteDeadline
func (c *CompressedConnection) SetWriteDeadline(t time.Time) error {
	return c.conn.SetWriteDeadline(t)
}
