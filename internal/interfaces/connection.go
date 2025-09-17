// Package interfaces defines core interfaces for the GoCat application
// following the dependency inversion principle and interface segregation principle.
package interfaces

import (
	"context"
	"io"
	"net"
	"time"
)

// Connection represents an enhanced network connection with statistics and context support
type Connection interface {
	net.Conn

	// Context returns the connection's context for cancellation
	Context() context.Context

	// Stats returns connection statistics
	Stats() ConnectionStats

	// SetMetrics enables metrics collection for this connection
	SetMetrics(collector MetricsCollector)

	// EnableCompression enables data compression if supported
	EnableCompression() error

	// SetRateLimit sets the rate limit for this connection
	SetRateLimit(bytesPerSecond int64) error
}

// ConnectionStats holds comprehensive connection statistics
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
	ErrorCount       int           `json:"error_count"`
	LastError        string        `json:"last_error,omitempty"`
	CompressionRatio float64       `json:"compression_ratio,omitempty"`
	RateLimit        int64         `json:"rate_limit,omitempty"`
}

// Dialer provides connection dialing capabilities with retry logic
type Dialer interface {
	// Dial connects to the specified address with context support
	Dial(ctx context.Context, address string) (Connection, error)

	// DialWithOptions connects using specific connection options
	DialWithOptions(ctx context.Context, options *ConnectionOptions) (Connection, error)
}

// Listener provides connection listening capabilities
type Listener interface {
	// Accept waits for and returns the next connection
	Accept() (Connection, error)

	// Close closes the listener
	Close() error

	// Addr returns the listener's network address
	Addr() net.Addr

	// Stats returns listener statistics
	Stats() ListenerStats
}

// ListenerStats holds listener statistics
type ListenerStats struct {
	StartedAt         time.Time `json:"started_at"`
	ConnectionsTotal  int64     `json:"connections_total"`
	ConnectionsActive int64     `json:"connections_active"`
	BytesRead         int64     `json:"bytes_read"`
	BytesWritten      int64     `json:"bytes_written"`
	ErrorCount        int64     `json:"error_count"`
}

// ConnectionPool manages a pool of reusable connections
type ConnectionPool interface {
	// Get retrieves a connection from the pool or creates a new one
	Get(ctx context.Context, address string) (Connection, error)

	// Put returns a connection to the pool
	Put(conn Connection) error

	// Close closes all connections in the pool
	Close() error

	// Stats returns pool statistics
	Stats() PoolStats
}

// PoolStats holds connection pool statistics
type PoolStats struct {
	TotalConnections     int `json:"total_connections"`
	ActiveConnections    int `json:"active_connections"`
	IdleConnections      int `json:"idle_connections"`
	PoolHits             int `json:"pool_hits"`
	PoolMisses           int `json:"pool_misses"`
	ConnectionsCreated   int `json:"connections_created"`
	ConnectionsDestroyed int `json:"connections_destroyed"`
}

// ConnectionOptions holds configuration for network connections
type ConnectionOptions struct {
	Host               string        `json:"host"`
	Port               int           `json:"port"`
	Protocol           string        `json:"protocol"`
	Timeout            time.Duration `json:"timeout"`
	KeepAlive          time.Duration `json:"keep_alive"`
	TLSConfig          interface{}   `json:"tls_config,omitempty"` // *tls.Config
	BindAddress        string        `json:"bind_address,omitempty"`
	BindPort           int           `json:"bind_port,omitempty"`
	ReuseAddr          bool          `json:"reuse_addr"`
	ReusePort          bool          `json:"reuse_port"`
	NoDelay            bool          `json:"no_delay"`
	BufferSize         int           `json:"buffer_size"`
	MaxConnections     int           `json:"max_connections"`
	ConnectionPoolSize int           `json:"connection_pool_size"`
	RetryAttempts      int           `json:"retry_attempts"`
	RetryDelay         time.Duration `json:"retry_delay"`
	IPv6               bool          `json:"ipv6"`
	IPv4               bool          `json:"ipv4"`
	EnableCompression  bool          `json:"enable_compression"`
	RateLimit          int64         `json:"rate_limit,omitempty"`
}

// MetricsCollector defines the interface for collecting metrics
type MetricsCollector interface {
	// IncrementCounter increments a counter metric
	IncrementCounter(name string, tags map[string]string)

	// RecordGauge records a gauge metric value
	RecordGauge(name string, value float64, tags map[string]string)

	// RecordHistogram records a histogram metric value
	RecordHistogram(name string, value float64, tags map[string]string)

	// RecordTimer records a timer metric
	RecordTimer(name string, duration time.Duration, tags map[string]string)
}

// DataPiper handles data transfer between connections
type DataPiper interface {
	// Pipe transfers data between two connections
	Pipe(ctx context.Context, conn1, conn2 io.ReadWriter) error

	// PipeWithBuffer transfers data with a custom buffer size
	PipeWithBuffer(ctx context.Context, dst io.Writer, src io.Reader, bufferSize int) error
}

// SecurityValidator provides input validation and sanitization
type SecurityValidator interface {
	// ValidateHostname validates a hostname or IP address
	ValidateHostname(hostname string) error

	// ValidatePort validates a port number
	ValidatePort(port string) (int, error)

	// ValidateCommand validates a command for execution
	ValidateCommand(command string) (string, error)

	// ValidateFilePath validates a file path
	ValidateFilePath(path string) error

	// SanitizeInput sanitizes general input
	SanitizeInput(input string) string
}

// RateLimiter provides rate limiting functionality
type RateLimiter interface {
	// Allow checks if a request is allowed
	Allow(identifier string) bool

	// Reset resets the rate limit for an identifier
	Reset(identifier string)

	// GetStats returns rate limiting statistics
	GetStats(identifier string) RateLimitStats
}

// RateLimitStats provides rate limiting statistics
type RateLimitStats struct {
	Identifier     string        `json:"identifier"`
	RequestCount   int           `json:"request_count"`
	WindowStart    time.Time     `json:"window_start"`
	WindowDuration time.Duration `json:"window_duration"`
	MaxRequests    int           `json:"max_requests"`
	IsBlocked      bool          `json:"is_blocked"`
	NextResetTime  time.Time     `json:"next_reset_time"`
	RemainingQuota int           `json:"remaining_quota"`
}
