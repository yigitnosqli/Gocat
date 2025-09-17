package network

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ibrahmsql/gocat/internal/errors"
)

// ConnectionPool interface defines the contract for connection pooling
type ConnectionPool interface {
	Get(ctx context.Context, address string) (Connection, error)
	Put(conn Connection) error
	Close() error
	Stats() PoolStats
	HealthCheck() error
}

// PoolStats holds connection pool statistics
type PoolStats struct {
	TotalConnections   int64     `json:"total_connections"`
	ActiveConnections  int64     `json:"active_connections"`
	IdleConnections    int64     `json:"idle_connections"`
	PoolHits           int64     `json:"pool_hits"`
	PoolMisses         int64     `json:"pool_misses"`
	ConnectionsCreated int64     `json:"connections_created"`
	ConnectionsReused  int64     `json:"connections_reused"`
	ConnectionsExpired int64     `json:"connections_expired"`
	LastActivity       time.Time `json:"last_activity"`
	CreatedAt          time.Time `json:"created_at"`
}

// PoolConfig holds configuration for connection pool
type PoolConfig struct {
	MaxSize             int           `yaml:"max_size"`
	MinSize             int           `yaml:"min_size"`
	MaxIdleTime         time.Duration `yaml:"max_idle_time"`
	MaxLifetime         time.Duration `yaml:"max_lifetime"`
	HealthCheckInterval time.Duration `yaml:"health_check_interval"`
	ConnectionTimeout   time.Duration `yaml:"connection_timeout"`
	ValidationTimeout   time.Duration `yaml:"validation_timeout"`
	EnableHealthCheck   bool          `yaml:"enable_health_check"`
	EnableMetrics       bool          `yaml:"enable_metrics"`
}

// DefaultPoolConfig returns default pool configuration
func DefaultPoolConfig() *PoolConfig {
	return &PoolConfig{
		MaxSize:             10,
		MinSize:             2,
		MaxIdleTime:         5 * time.Minute,
		MaxLifetime:         30 * time.Minute,
		HealthCheckInterval: 1 * time.Minute,
		ConnectionTimeout:   30 * time.Second,
		ValidationTimeout:   5 * time.Second,
		EnableHealthCheck:   true,
		EnableMetrics:       true,
	}
}

// pooledConnection wraps an  connection with pool metadata
type pooledConnection struct {
	conn      Connection
	createdAt time.Time
	lastUsed  time.Time
	useCount  int64
	inUse     bool
	healthy   bool
}

// ConnectionPoolImpl implements the ConnectionPool interface
type ConnectionPoolImpl struct {
	config           *PoolConfig
	dialer           *Dialer
	connections      map[string][]*pooledConnection
	stats            PoolStats
	mu               sync.RWMutex
	closed           bool
	ctx              context.Context
	cancel           context.CancelFunc
	metricsCollector MetricsCollector
}

// NewConnectionPool creates a new connection pool
func NewConnectionPool(config *PoolConfig, dialer *Dialer) *ConnectionPoolImpl {
	if config == nil {
		config = DefaultPoolConfig()
	}

	ctx, cancel := context.WithCancel(context.Background())

	pool := &ConnectionPoolImpl{
		config:      config,
		dialer:      dialer,
		connections: make(map[string][]*pooledConnection),
		stats: PoolStats{
			CreatedAt: time.Now(),
		},
		ctx:    ctx,
		cancel: cancel,
	}

	// Start background maintenance
	go pool.maintenanceWorker()

	return pool
}

// Get retrieves a connection from the pool or creates a new one
func (p *ConnectionPoolImpl) Get(ctx context.Context, address string) (Connection, error) {
	if p.isClosed() {
		return nil, errors.NetworkError("NET030", "Connection pool is closed").WithUserFriendly("Connection pool has been closed")
	}

	// Try to get an existing connection from the pool
	if conn := p.getFromPool(address); conn != nil {
		atomic.AddInt64(&p.stats.PoolHits, 1)
		atomic.AddInt64(&p.stats.ConnectionsReused, 1)
		p.updateLastActivity()

		if p.metricsCollector != nil {
			p.metricsCollector.IncrementCounter("pool_hits", map[string]string{
				"address": address,
			})
		}

		return conn, nil
	}

	// Pool miss - create new connection
	atomic.AddInt64(&p.stats.PoolMisses, 1)

	if p.metricsCollector != nil {
		p.metricsCollector.IncrementCounter("pool_misses", map[string]string{
			"address": address,
		})
	}

	// Check if we can create a new connection
	if !p.canCreateConnection(address) {
		return nil, errors.NetworkError("NET031", "Connection pool is full").WithUserFriendly("Maximum number of connections reached")
	}

	// Create new connection with timeout
	connCtx, cancel := context.WithTimeout(ctx, p.config.ConnectionTimeout)
	defer cancel()

	conn, err := p.dialer.Dial(connCtx, address)
	if err != nil {
		return nil, errors.WrapError(err, errors.ErrorTypeNetwork, errors.SeverityHigh, "NET032", "Failed to create pooled connection")
	}

	// Wrap connection for pool management
	pooledConn := &pooledConnection{
		conn:      conn,
		createdAt: time.Now(),
		lastUsed:  time.Now(),
		useCount:  1,
		inUse:     true,
		healthy:   true,
	}

	// Add to pool tracking
	p.mu.Lock()
	if p.connections[address] == nil {
		p.connections[address] = make([]*pooledConnection, 0)
	}
	p.connections[address] = append(p.connections[address], pooledConn)
	atomic.AddInt64(&p.stats.TotalConnections, 1)
	atomic.AddInt64(&p.stats.ActiveConnections, 1)
	atomic.AddInt64(&p.stats.ConnectionsCreated, 1)
	p.mu.Unlock()

	p.updateLastActivity()

	if p.metricsCollector != nil {
		p.metricsCollector.IncrementCounter("connections_created", map[string]string{
			"address": address,
		})
	}

	return conn, nil
}

// Put returns a connection to the pool
func (p *ConnectionPoolImpl) Put(conn Connection) error {
	if p.isClosed() {
		return conn.Close()
	}

	if conn == nil {
		return errors.ValidationError("VAL017", "Connection cannot be nil")
	}

	// Find the pooled connection
	p.mu.Lock()
	defer p.mu.Unlock()

	var found *pooledConnection
	var address string

	for addr, conns := range p.connections {
		for _, pc := range conns {
			if pc.conn.GetID() == conn.GetID() {
				found = pc
				address = addr
				break
			}
		}
		if found != nil {
			break
		}
	}

	if found == nil {
		// Connection not from this pool, just close it
		return conn.Close()
	}

	// Check if connection is still healthy and within lifetime limits
	now := time.Now()
	if !conn.IsHealthy() ||
		now.Sub(found.createdAt) > p.config.MaxLifetime ||
		now.Sub(found.lastUsed) > p.config.MaxIdleTime {

		// Remove from pool and close
		p.removeConnection(address, found)
		atomic.AddInt64(&p.stats.ConnectionsExpired, 1)

		if p.metricsCollector != nil {
			p.metricsCollector.IncrementCounter("connections_expired", map[string]string{
				"address": address,
				"reason":  "unhealthy_or_expired",
			})
		}

		return conn.Close()
	}

	// Return to pool
	found.inUse = false
	found.lastUsed = now
	atomic.AddInt64(&found.useCount, 1)
	atomic.AddInt64(&p.stats.ActiveConnections, -1)
	atomic.AddInt64(&p.stats.IdleConnections, 1)

	p.updateLastActivity()

	if p.metricsCollector != nil {
		p.metricsCollector.IncrementCounter("connections_returned", map[string]string{
			"address": address,
		})
	}

	return nil
}

// Close closes the connection pool and all connections
func (p *ConnectionPoolImpl) Close() error {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return nil
	}
	p.closed = true
	p.mu.Unlock()

	// Cancel background workers
	p.cancel()

	// Close all connections
	p.mu.Lock()
	defer p.mu.Unlock()

	var lastErr error
	for address, conns := range p.connections {
		for _, pc := range conns {
			if err := pc.conn.Close(); err != nil {
				lastErr = err
			}
		}
		delete(p.connections, address)
	}

	// Reset stats
	atomic.StoreInt64(&p.stats.TotalConnections, 0)
	atomic.StoreInt64(&p.stats.ActiveConnections, 0)
	atomic.StoreInt64(&p.stats.IdleConnections, 0)

	if p.metricsCollector != nil {
		p.metricsCollector.IncrementCounter("pool_closed", map[string]string{})
	}

	return lastErr
}

// Stats returns current pool statistics
func (p *ConnectionPoolImpl) Stats() PoolStats {
	p.mu.RLock()
	defer p.mu.RUnlock()

	stats := p.stats
	stats.TotalConnections = atomic.LoadInt64(&p.stats.TotalConnections)
	stats.ActiveConnections = atomic.LoadInt64(&p.stats.ActiveConnections)
	stats.IdleConnections = atomic.LoadInt64(&p.stats.IdleConnections)
	stats.PoolHits = atomic.LoadInt64(&p.stats.PoolHits)
	stats.PoolMisses = atomic.LoadInt64(&p.stats.PoolMisses)
	stats.ConnectionsCreated = atomic.LoadInt64(&p.stats.ConnectionsCreated)
	stats.ConnectionsReused = atomic.LoadInt64(&p.stats.ConnectionsReused)
	stats.ConnectionsExpired = atomic.LoadInt64(&p.stats.ConnectionsExpired)

	return stats
}

// HealthCheck performs a health check on the pool
func (p *ConnectionPoolImpl) HealthCheck() error {
	if p.isClosed() {
		return errors.NetworkError("NET033", "Connection pool is closed")
	}

	p.mu.RLock()
	defer p.mu.RUnlock()

	unhealthyCount := 0
	totalCount := 0

	for _, conns := range p.connections {
		for _, pc := range conns {
			totalCount++
			if !pc.conn.IsHealthy() {
				unhealthyCount++
			}
		}
	}

	if totalCount > 0 && float64(unhealthyCount)/float64(totalCount) > 0.5 {
		return errors.NetworkError("NET034", "More than 50% of pool connections are unhealthy").WithContext("unhealthy_count", unhealthyCount).WithContext("total_count", totalCount)
	}

	return nil
}

// SetMetrics sets the metrics collector for the pool
func (p *ConnectionPoolImpl) SetMetrics(collector MetricsCollector) {
	p.mu.Lock()
	p.metricsCollector = collector
	p.mu.Unlock()
}

// getFromPool attempts to get an available connection from the pool
func (p *ConnectionPoolImpl) getFromPool(address string) Connection {
	p.mu.Lock()
	defer p.mu.Unlock()

	conns, exists := p.connections[address]
	if !exists || len(conns) == 0 {
		return nil
	}

	// Find an available healthy connection
	for i, pc := range conns {
		if !pc.inUse && pc.healthy && pc.conn.IsHealthy() {
			// Check if connection is still within limits
			now := time.Now()
			if now.Sub(pc.createdAt) <= p.config.MaxLifetime &&
				now.Sub(pc.lastUsed) <= p.config.MaxIdleTime {

				// Mark as in use
				pc.inUse = true
				pc.lastUsed = now
				atomic.AddInt64(&pc.useCount, 1)
				atomic.AddInt64(&p.stats.ActiveConnections, 1)
				atomic.AddInt64(&p.stats.IdleConnections, -1)

				return pc.conn
			} else {
				// Connection expired, remove it
				p.removeConnectionAtIndex(address, i)
				atomic.AddInt64(&p.stats.ConnectionsExpired, 1)
				pc.conn.Close()
			}
		}
	}

	return nil
}

// canCreateConnection checks if a new connection can be created
func (p *ConnectionPoolImpl) canCreateConnection(address string) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()

	conns := p.connections[address]
	if len(conns) >= p.config.MaxSize {
		return false
	}

	return true
}

// removeConnection removes a specific connection from the pool
func (p *ConnectionPoolImpl) removeConnection(address string, target *pooledConnection) {
	conns := p.connections[address]
	for i, pc := range conns {
		if pc == target {
			p.removeConnectionAtIndex(address, i)
			break
		}
	}
}

// removeConnectionAtIndex removes a connection at a specific index
func (p *ConnectionPoolImpl) removeConnectionAtIndex(address string, index int) {
	conns := p.connections[address]
	if index < 0 || index >= len(conns) {
		return
	}

	// Remove from slice
	conns[index] = conns[len(conns)-1]
	conns = conns[:len(conns)-1]
	p.connections[address] = conns

	atomic.AddInt64(&p.stats.TotalConnections, -1)
	if conns[index].inUse {
		atomic.AddInt64(&p.stats.ActiveConnections, -1)
	} else {
		atomic.AddInt64(&p.stats.IdleConnections, -1)
	}
}

// isClosed checks if the pool is closed
func (p *ConnectionPoolImpl) isClosed() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.closed
}

// updateLastActivity updates the last activity timestamp
func (p *ConnectionPoolImpl) updateLastActivity() {
	p.mu.Lock()
	p.stats.LastActivity = time.Now()
	p.mu.Unlock()
}

// maintenanceWorker runs background maintenance tasks
func (p *ConnectionPoolImpl) maintenanceWorker() {
	ticker := time.NewTicker(p.config.HealthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-p.ctx.Done():
			return
		case <-ticker.C:
			p.performMaintenance()
		}
	}
}

// performMaintenance performs periodic maintenance on the pool
func (p *ConnectionPoolImpl) performMaintenance() {
	if p.isClosed() {
		return
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	now := time.Now()
	expiredCount := 0

	// Check each address pool
	for address, conns := range p.connections {
		// Remove expired and unhealthy connections
		validConns := make([]*pooledConnection, 0, len(conns))

		for _, pc := range conns {
			shouldRemove := false

			// Check if connection is expired
			if now.Sub(pc.createdAt) > p.config.MaxLifetime {
				shouldRemove = true
			}

			// Check if idle connection is too old
			if !pc.inUse && now.Sub(pc.lastUsed) > p.config.MaxIdleTime {
				shouldRemove = true
			}

			// Check health if enabled
			if p.config.EnableHealthCheck && !pc.conn.IsHealthy() {
				shouldRemove = true
				pc.healthy = false
			}

			if shouldRemove {
				expiredCount++
				pc.conn.Close()

				if pc.inUse {
					atomic.AddInt64(&p.stats.ActiveConnections, -1)
				} else {
					atomic.AddInt64(&p.stats.IdleConnections, -1)
				}
				atomic.AddInt64(&p.stats.TotalConnections, -1)
			} else {
				validConns = append(validConns, pc)
			}
		}

		p.connections[address] = validConns

		// Ensure minimum connections if configured
		if len(validConns) < p.config.MinSize && p.config.MinSize > 0 {
			// This would require creating new connections in background
			// For now, we'll just log this condition
		}
	}

	if expiredCount > 0 {
		atomic.AddInt64(&p.stats.ConnectionsExpired, int64(expiredCount))

		if p.metricsCollector != nil {
			p.metricsCollector.IncrementCounter("maintenance_expired_connections", map[string]string{
				"count": string(rune(expiredCount)),
			})
		}
	}
}
