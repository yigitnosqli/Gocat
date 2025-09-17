package health

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"runtime"
	"sync/atomic"
	"time"
)

// MemoryHealthChecker checks memory usage
type MemoryHealthChecker struct {
	name               string
	timeout            time.Duration
	interval           time.Duration
	maxMemoryMB        int64
	warningThresholdMB int64
}

// NewMemoryHealthChecker creates a new memory health checker
func NewMemoryHealthChecker(maxMemoryMB, warningThresholdMB int64) *MemoryHealthChecker {
	return &MemoryHealthChecker{
		name:               "memory",
		timeout:            5 * time.Second,
		interval:           30 * time.Second,
		maxMemoryMB:        maxMemoryMB,
		warningThresholdMB: warningThresholdMB,
	}
}

// GetName returns the checker name
func (mhc *MemoryHealthChecker) GetName() string {
	return mhc.name
}

// GetTimeout returns the check timeout
func (mhc *MemoryHealthChecker) GetTimeout() time.Duration {
	return mhc.timeout
}

// GetInterval returns the check interval
func (mhc *MemoryHealthChecker) GetInterval() time.Duration {
	return mhc.interval
}

// Check performs the memory health check
func (mhc *MemoryHealthChecker) Check(ctx context.Context) HealthCheckResult {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	allocMB := int64(m.Alloc / 1024 / 1024)
	sysMB := int64(m.Sys / 1024 / 1024)

	metadata := map[string]interface{}{
		"alloc_mb":   allocMB,
		"sys_mb":     sysMB,
		"heap_mb":    int64(m.HeapAlloc / 1024 / 1024),
		"goroutines": runtime.NumGoroutine(),
		"gc_cycles":  m.NumGC,
	}

	result := HealthCheckResult{
		Name:     mhc.name,
		Metadata: metadata,
	}

	if allocMB > mhc.maxMemoryMB {
		result.Status = StatusUnhealthy
		result.Error = fmt.Sprintf("Memory usage %dMB exceeds maximum %dMB", allocMB, mhc.maxMemoryMB)
	} else if allocMB > mhc.warningThresholdMB {
		result.Status = StatusDegraded
		result.Message = fmt.Sprintf("Memory usage %dMB exceeds warning threshold %dMB", allocMB, mhc.warningThresholdMB)
	} else {
		result.Status = StatusHealthy
		result.Message = fmt.Sprintf("Memory usage %dMB is within limits", allocMB)
	}

	return result
}

// GoroutineHealthChecker checks goroutine count
type GoroutineHealthChecker struct {
	name             string
	timeout          time.Duration
	interval         time.Duration
	maxGoroutines    int
	warningThreshold int
}

// NewGoroutineHealthChecker creates a new goroutine health checker
func NewGoroutineHealthChecker(maxGoroutines, warningThreshold int) *GoroutineHealthChecker {
	return &GoroutineHealthChecker{
		name:             "goroutines",
		timeout:          5 * time.Second,
		interval:         30 * time.Second,
		maxGoroutines:    maxGoroutines,
		warningThreshold: warningThreshold,
	}
}

// GetName returns the checker name
func (ghc *GoroutineHealthChecker) GetName() string {
	return ghc.name
}

// GetTimeout returns the check timeout
func (ghc *GoroutineHealthChecker) GetTimeout() time.Duration {
	return ghc.timeout
}

// GetInterval returns the check interval
func (ghc *GoroutineHealthChecker) GetInterval() time.Duration {
	return ghc.interval
}

// Check performs the goroutine health check
func (ghc *GoroutineHealthChecker) Check(ctx context.Context) HealthCheckResult {
	count := runtime.NumGoroutine()

	metadata := map[string]interface{}{
		"count":     count,
		"max":       ghc.maxGoroutines,
		"warning":   ghc.warningThreshold,
		"cpu_cores": runtime.NumCPU(),
	}

	result := HealthCheckResult{
		Name:     ghc.name,
		Metadata: metadata,
	}

	if count > ghc.maxGoroutines {
		result.Status = StatusUnhealthy
		result.Error = fmt.Sprintf("Goroutine count %d exceeds maximum %d", count, ghc.maxGoroutines)
	} else if count > ghc.warningThreshold {
		result.Status = StatusDegraded
		result.Message = fmt.Sprintf("Goroutine count %d exceeds warning threshold %d", count, ghc.warningThreshold)
	} else {
		result.Status = StatusHealthy
		result.Message = fmt.Sprintf("Goroutine count %d is within limits", count)
	}

	return result
}

// NetworkHealthChecker checks network connectivity
type NetworkHealthChecker struct {
	name     string
	timeout  time.Duration
	interval time.Duration
	host     string
	port     string
}

// NewNetworkHealthChecker creates a new network health checker
func NewNetworkHealthChecker(host, port string) *NetworkHealthChecker {
	return &NetworkHealthChecker{
		name:     fmt.Sprintf("network_%s_%s", host, port),
		timeout:  10 * time.Second,
		interval: 60 * time.Second,
		host:     host,
		port:     port,
	}
}

// GetName returns the checker name
func (nhc *NetworkHealthChecker) GetName() string {
	return nhc.name
}

// GetTimeout returns the check timeout
func (nhc *NetworkHealthChecker) GetTimeout() time.Duration {
	return nhc.timeout
}

// GetInterval returns the check interval
func (nhc *NetworkHealthChecker) GetInterval() time.Duration {
	return nhc.interval
}

// Check performs the network health check
func (nhc *NetworkHealthChecker) Check(ctx context.Context) HealthCheckResult {
	start := time.Now()

	address := net.JoinHostPort(nhc.host, nhc.port)
	conn, err := net.DialTimeout("tcp", address, nhc.timeout)

	duration := time.Since(start)

	metadata := map[string]interface{}{
		"host":            nhc.host,
		"port":            nhc.port,
		"address":         address,
		"connect_time_ms": duration.Milliseconds(),
	}

	result := HealthCheckResult{
		Name:     nhc.name,
		Metadata: metadata,
	}

	if err != nil {
		result.Status = StatusUnhealthy
		result.Error = fmt.Sprintf("Failed to connect to %s: %v", address, err)
	} else {
		conn.Close()
		result.Status = StatusHealthy
		result.Message = fmt.Sprintf("Successfully connected to %s in %v", address, duration)
	}

	return result
}

// HTTPHealthChecker checks HTTP endpoint health
type HTTPHealthChecker struct {
	name           string
	timeout        time.Duration
	interval       time.Duration
	url            string
	expectedStatus int
	client         *http.Client
}

// NewHTTPHealthChecker creates a new HTTP health checker
func NewHTTPHealthChecker(name, url string, expectedStatus int) *HTTPHealthChecker {
	return &HTTPHealthChecker{
		name:           name,
		timeout:        15 * time.Second,
		interval:       60 * time.Second,
		url:            url,
		expectedStatus: expectedStatus,
		client: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

// GetName returns the checker name
func (hhc *HTTPHealthChecker) GetName() string {
	return hhc.name
}

// GetTimeout returns the check timeout
func (hhc *HTTPHealthChecker) GetTimeout() time.Duration {
	return hhc.timeout
}

// GetInterval returns the check interval
func (hhc *HTTPHealthChecker) GetInterval() time.Duration {
	return hhc.interval
}

// Check performs the HTTP health check
func (hhc *HTTPHealthChecker) Check(ctx context.Context) HealthCheckResult {
	start := time.Now()

	req, err := http.NewRequestWithContext(ctx, "GET", hhc.url, nil)
	if err != nil {
		return HealthCheckResult{
			Name:   hhc.name,
			Status: StatusUnhealthy,
			Error:  fmt.Sprintf("Failed to create request: %v", err),
		}
	}

	resp, err := hhc.client.Do(req)
	duration := time.Since(start)

	metadata := map[string]interface{}{
		"url":              hhc.url,
		"expected_status":  hhc.expectedStatus,
		"response_time_ms": duration.Milliseconds(),
	}

	result := HealthCheckResult{
		Name:     hhc.name,
		Metadata: metadata,
	}

	if err != nil {
		result.Status = StatusUnhealthy
		result.Error = fmt.Sprintf("HTTP request failed: %v", err)
		return result
	}

	defer resp.Body.Close()
	metadata["actual_status"] = resp.StatusCode

	if resp.StatusCode != hhc.expectedStatus {
		result.Status = StatusUnhealthy
		result.Error = fmt.Sprintf("Expected status %d, got %d", hhc.expectedStatus, resp.StatusCode)
	} else {
		result.Status = StatusHealthy
		result.Message = fmt.Sprintf("HTTP endpoint %s returned expected status %d in %v", hhc.url, resp.StatusCode, duration)
	}

	return result
}

// DiskSpaceHealthChecker checks disk space usage
type DiskSpaceHealthChecker struct {
	name            string
	timeout         time.Duration
	interval        time.Duration
	path            string
	maxUsagePercent float64
	warningPercent  float64
}

// NewDiskSpaceHealthChecker creates a new disk space health checker
func NewDiskSpaceHealthChecker(path string, maxUsagePercent, warningPercent float64) *DiskSpaceHealthChecker {
	return &DiskSpaceHealthChecker{
		name:            fmt.Sprintf("disk_space_%s", path),
		timeout:         5 * time.Second,
		interval:        300 * time.Second, // 5 minutes
		path:            path,
		maxUsagePercent: maxUsagePercent,
		warningPercent:  warningPercent,
	}
}

// GetName returns the checker name
func (dshc *DiskSpaceHealthChecker) GetName() string {
	return dshc.name
}

// GetTimeout returns the check timeout
func (dshc *DiskSpaceHealthChecker) GetTimeout() time.Duration {
	return dshc.timeout
}

// GetInterval returns the check interval
func (dshc *DiskSpaceHealthChecker) GetInterval() time.Duration {
	return dshc.interval
}

// Check performs the disk space health check
func (dshc *DiskSpaceHealthChecker) Check(ctx context.Context) HealthCheckResult {
	// This is a simplified implementation
	// In a real implementation, you would use syscalls to get actual disk usage

	metadata := map[string]interface{}{
		"path":          dshc.path,
		"max_usage":     dshc.maxUsagePercent,
		"warning_usage": dshc.warningPercent,
	}

	result := HealthCheckResult{
		Name:     dshc.name,
		Metadata: metadata,
		Status:   StatusHealthy,
		Message:  fmt.Sprintf("Disk space check for %s completed (simplified implementation)", dshc.path),
	}

	return result
}

// CustomHealthChecker allows for custom health check functions
type CustomHealthChecker struct {
	name     string
	timeout  time.Duration
	interval time.Duration
	checkFn  func(ctx context.Context) HealthCheckResult
}

// NewCustomHealthChecker creates a new custom health checker
func NewCustomHealthChecker(name string, timeout, interval time.Duration, checkFn func(ctx context.Context) HealthCheckResult) *CustomHealthChecker {
	return &CustomHealthChecker{
		name:     name,
		timeout:  timeout,
		interval: interval,
		checkFn:  checkFn,
	}
}

// GetName returns the checker name
func (chc *CustomHealthChecker) GetName() string {
	return chc.name
}

// GetTimeout returns the check timeout
func (chc *CustomHealthChecker) GetTimeout() time.Duration {
	return chc.timeout
}

// GetInterval returns the check interval
func (chc *CustomHealthChecker) GetInterval() time.Duration {
	return chc.interval
}

// Check performs the custom health check
func (chc *CustomHealthChecker) Check(ctx context.Context) HealthCheckResult {
	if chc.checkFn == nil {
		return HealthCheckResult{
			Name:   chc.name,
			Status: StatusUnhealthy,
			Error:  "No check function provided",
		}
	}

	return chc.checkFn(ctx)
}

// ConnectionPoolHealthChecker checks connection pool health
type ConnectionPoolHealthChecker struct {
	name           string
	timeout        time.Duration
	interval       time.Duration
	poolName       string
	getPoolStats   func() (active, idle, total int)
	maxConnections int
}

// NewConnectionPoolHealthChecker creates a new connection pool health checker
func NewConnectionPoolHealthChecker(poolName string, maxConnections int, getPoolStats func() (active, idle, total int)) *ConnectionPoolHealthChecker {
	return &ConnectionPoolHealthChecker{
		name:           fmt.Sprintf("connection_pool_%s", poolName),
		timeout:        5 * time.Second,
		interval:       30 * time.Second,
		poolName:       poolName,
		getPoolStats:   getPoolStats,
		maxConnections: maxConnections,
	}
}

// GetName returns the checker name
func (cphc *ConnectionPoolHealthChecker) GetName() string {
	return cphc.name
}

// GetTimeout returns the check timeout
func (cphc *ConnectionPoolHealthChecker) GetTimeout() time.Duration {
	return cphc.timeout
}

// GetInterval returns the check interval
func (cphc *ConnectionPoolHealthChecker) GetInterval() time.Duration {
	return cphc.interval
}

// Check performs the connection pool health check
func (cphc *ConnectionPoolHealthChecker) Check(ctx context.Context) HealthCheckResult {
	if cphc.getPoolStats == nil {
		return HealthCheckResult{
			Name:   cphc.name,
			Status: StatusUnhealthy,
			Error:  "No pool stats function provided",
		}
	}

	active, idle, total := cphc.getPoolStats()
	usagePercent := float64(total) / float64(cphc.maxConnections) * 100

	metadata := map[string]interface{}{
		"pool_name":     cphc.poolName,
		"active":        active,
		"idle":          idle,
		"total":         total,
		"max":           cphc.maxConnections,
		"usage_percent": usagePercent,
	}

	result := HealthCheckResult{
		Name:     cphc.name,
		Metadata: metadata,
	}

	if total >= cphc.maxConnections {
		result.Status = StatusUnhealthy
		result.Error = fmt.Sprintf("Connection pool %s is at maximum capacity (%d/%d)", cphc.poolName, total, cphc.maxConnections)
	} else if usagePercent > 80 {
		result.Status = StatusDegraded
		result.Message = fmt.Sprintf("Connection pool %s usage is high: %.1f%% (%d/%d)", cphc.poolName, usagePercent, total, cphc.maxConnections)
	} else {
		result.Status = StatusHealthy
		result.Message = fmt.Sprintf("Connection pool %s is healthy: %d active, %d idle, %d total", cphc.poolName, active, idle, total)
	}

	return result
}

// MetricsHealthChecker checks if metrics are being collected
type MetricsHealthChecker struct {
	name           string
	timeout        time.Duration
	interval       time.Duration
	getMetricCount func() int64
	lastCount      int64
	staleThreshold time.Duration
	lastUpdate     time.Time
}

// NewMetricsHealthChecker creates a new metrics health checker
func NewMetricsHealthChecker(getMetricCount func() int64) *MetricsHealthChecker {
	return &MetricsHealthChecker{
		name:           "metrics",
		timeout:        5 * time.Second,
		interval:       60 * time.Second,
		getMetricCount: getMetricCount,
		staleThreshold: 5 * time.Minute,
		lastUpdate:     time.Now(),
	}
}

// GetName returns the checker name
func (mhc *MetricsHealthChecker) GetName() string {
	return mhc.name
}

// GetTimeout returns the check timeout
func (mhc *MetricsHealthChecker) GetTimeout() time.Duration {
	return mhc.timeout
}

// GetInterval returns the check interval
func (mhc *MetricsHealthChecker) GetInterval() time.Duration {
	return mhc.interval
}

// Check performs the metrics health check
func (mhc *MetricsHealthChecker) Check(ctx context.Context) HealthCheckResult {
	if mhc.getMetricCount == nil {
		return HealthCheckResult{
			Name:   mhc.name,
			Status: StatusUnhealthy,
			Error:  "No metric count function provided",
		}
	}

	currentCount := mhc.getMetricCount()
	now := time.Now()

	metadata := map[string]interface{}{
		"current_count": currentCount,
		"last_count":    atomic.LoadInt64(&mhc.lastCount),
		"last_update":   mhc.lastUpdate.Format(time.RFC3339),
	}

	result := HealthCheckResult{
		Name:     mhc.name,
		Metadata: metadata,
	}

	// Check if metrics are being updated
	if currentCount > atomic.LoadInt64(&mhc.lastCount) {
		mhc.lastUpdate = now
		atomic.StoreInt64(&mhc.lastCount, currentCount)
		result.Status = StatusHealthy
		result.Message = fmt.Sprintf("Metrics are being collected: %d total", currentCount)
	} else if now.Sub(mhc.lastUpdate) > mhc.staleThreshold {
		result.Status = StatusDegraded
		result.Message = fmt.Sprintf("Metrics appear stale: no updates for %v", now.Sub(mhc.lastUpdate))
	} else {
		result.Status = StatusHealthy
		result.Message = fmt.Sprintf("Metrics collection is stable: %d total", currentCount)
	}

	return result
}
