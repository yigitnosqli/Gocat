package metrics

import (
	"sync"
	"time"
)

// Error type constants for improved type safety
const (
	ErrorTypeNetwork    = "network"
	ErrorTypeValidation = "validation"
	ErrorTypeSecurity   = "security"
	ErrorTypeTimeout    = "timeout"
)

// Metrics holds application metrics
type Metrics struct {
	mu                sync.RWMutex
	ConnectionsActive int64
	ConnectionsTotal  int64
	ConnectionsFailed int64
	BytesTransferred  int64
	BytesReceived     int64
	BytesSent         int64
	ErrorsTotal       int64
	RequestsTotal     int64
	RequestDuration   time.Duration
	LastActivity      time.Time
	StartTime         time.Time
	NetworkErrors     int64
	ValidationErrors  int64
	SecurityErrors    int64
	TimeoutErrors     int64
	RetryAttempts     int64
	SuccessfulRetries int64
	FailedRetries     int64
}

// NewMetrics creates a new metrics instance
func NewMetrics() *Metrics {
	return &Metrics{
		StartTime: time.Now(),
	}
}

// IncrementConnectionsActive increments active connections counter
func (m *Metrics) IncrementConnectionsActive() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ConnectionsActive++
	m.ConnectionsTotal++
	m.LastActivity = time.Now()
}

// DecrementConnectionsActive decrements active connections counter
func (m *Metrics) DecrementConnectionsActive() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.ConnectionsActive > 0 {
		m.ConnectionsActive--
	}
	m.LastActivity = time.Now()
}

// IncrementConnectionsFailed increments failed connections counter
func (m *Metrics) IncrementConnectionsFailed() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ConnectionsFailed++
	m.LastActivity = time.Now()
}

// AddBytesTransferred adds to bytes transferred counter
func (m *Metrics) AddBytesTransferred(bytes int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.BytesTransferred += bytes
	m.LastActivity = time.Now()
}

// AddBytesReceived adds to bytes received counter
func (m *Metrics) AddBytesReceived(bytes int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.BytesReceived += bytes
	m.BytesTransferred += bytes
	m.LastActivity = time.Now()
}

// AddBytesSent adds to bytes sent counter
func (m *Metrics) AddBytesSent(bytes int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.BytesSent += bytes
	m.BytesTransferred += bytes
	m.LastActivity = time.Now()
}

// IncrementErrors increments error counters by type
func (m *Metrics) IncrementErrors(errorType string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ErrorsTotal++

	switch errorType {
	case ErrorTypeNetwork:
		m.NetworkErrors++
	case ErrorTypeValidation:
		m.ValidationErrors++
	case ErrorTypeSecurity:
		m.SecurityErrors++
	case ErrorTypeTimeout:
		m.TimeoutErrors++
	}

	m.LastActivity = time.Now()
}

// IncrementRequests increments request counter
func (m *Metrics) IncrementRequests() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.RequestsTotal++
	m.LastActivity = time.Now()
}

// RecordRequestDuration records request duration
func (m *Metrics) RecordRequestDuration(duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	// Calculate average using float64 to prevent overflow and preserve precision
	if m.RequestsTotal > 0 {
		// Convert to float64 for precise calculation
		currentAvg := float64(m.RequestDuration)
		newDuration := float64(duration)
		totalRequests := float64(m.RequestsTotal)

		// Calculate weighted average: ((current_avg * (n-1)) + new_duration) / n
		newAvg := (currentAvg*(totalRequests-1) + newDuration) / totalRequests
		m.RequestDuration = time.Duration(newAvg)
	} else {
		m.RequestDuration = duration
	}
	m.LastActivity = time.Now()
}

// IncrementRetryAttempts increments retry attempts counter
func (m *Metrics) IncrementRetryAttempts() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.RetryAttempts++
	m.LastActivity = time.Now()
}

// IncrementSuccessfulRetries increments successful retries counter
func (m *Metrics) IncrementSuccessfulRetries() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.SuccessfulRetries++
	m.LastActivity = time.Now()
}

// IncrementFailedRetries increments failed retries counter
func (m *Metrics) IncrementFailedRetries() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.FailedRetries++
	m.LastActivity = time.Now()
}

// GetSnapshot returns a snapshot of current metrics
func (m *Metrics) GetSnapshot() MetricsSnapshot {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return MetricsSnapshot{
		ConnectionsActive: m.ConnectionsActive,
		ConnectionsTotal:  m.ConnectionsTotal,
		ConnectionsFailed: m.ConnectionsFailed,
		BytesTransferred:  m.BytesTransferred,
		BytesReceived:     m.BytesReceived,
		BytesSent:         m.BytesSent,
		ErrorsTotal:       m.ErrorsTotal,
		RequestsTotal:     m.RequestsTotal,
		RequestDuration:   m.RequestDuration,
		LastActivity:      m.LastActivity,
		Uptime:            time.Since(m.StartTime),
		NetworkErrors:     m.NetworkErrors,
		ValidationErrors:  m.ValidationErrors,
		SecurityErrors:    m.SecurityErrors,
		TimeoutErrors:     m.TimeoutErrors,
		RetryAttempts:     m.RetryAttempts,
		SuccessfulRetries: m.SuccessfulRetries,
		FailedRetries:     m.FailedRetries,
		Timestamp:         time.Now(),
	}
}

// MetricsSnapshot represents a point-in-time snapshot of metrics
type MetricsSnapshot struct {
	ConnectionsActive int64         `json:"connections_active"`
	ConnectionsTotal  int64         `json:"connections_total"`
	ConnectionsFailed int64         `json:"connections_failed"`
	BytesTransferred  int64         `json:"bytes_transferred"`
	BytesReceived     int64         `json:"bytes_received"`
	BytesSent         int64         `json:"bytes_sent"`
	ErrorsTotal       int64         `json:"errors_total"`
	RequestsTotal     int64         `json:"requests_total"`
	RequestDuration   time.Duration `json:"request_duration"`
	LastActivity      time.Time     `json:"last_activity"`
	Uptime            time.Duration `json:"uptime"`
	NetworkErrors     int64         `json:"network_errors"`
	ValidationErrors  int64         `json:"validation_errors"`
	SecurityErrors    int64         `json:"security_errors"`
	TimeoutErrors     int64         `json:"timeout_errors"`
	RetryAttempts     int64         `json:"retry_attempts"`
	SuccessfulRetries int64         `json:"successful_retries"`
	FailedRetries     int64         `json:"failed_retries"`
	Timestamp         time.Time     `json:"timestamp"`
}

// SuccessRate returns the success rate as a percentage
func (s MetricsSnapshot) SuccessRate() float64 {
	if s.ConnectionsTotal == 0 {
		return 0
	}
	successful := s.ConnectionsTotal - s.ConnectionsFailed
	return float64(successful) / float64(s.ConnectionsTotal) * 100
}

// ErrorRate returns the error rate as a percentage
func (s MetricsSnapshot) ErrorRate() float64 {
	if s.RequestsTotal == 0 {
		return 0
	}
	return float64(s.ErrorsTotal) / float64(s.RequestsTotal) * 100
}

// RetrySuccessRate returns the retry success rate as a percentage
func (s MetricsSnapshot) RetrySuccessRate() float64 {
	if s.RetryAttempts == 0 {
		return 0
	}
	return float64(s.SuccessfulRetries) / float64(s.RetryAttempts) * 100
}

// ThroughputBytesPerSecond returns throughput in bytes per second
func (s MetricsSnapshot) ThroughputBytesPerSecond() float64 {
	if s.Uptime.Seconds() == 0 {
		return 0
	}
	return float64(s.BytesTransferred) / s.Uptime.Seconds()
}

// RequestsPerSecond returns requests per second
func (s MetricsSnapshot) RequestsPerSecond() float64 {
	if s.Uptime.Seconds() == 0 {
		return 0
	}
	return float64(s.RequestsTotal) / s.Uptime.Seconds()
}

// Global metrics instance
var globalMetrics = NewMetrics()

// GetGlobalMetrics returns the global metrics instance
func GetGlobalMetrics() *Metrics {
	return globalMetrics
}

// Reset resets all metrics to zero
func (m *Metrics) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.ConnectionsActive = 0
	m.ConnectionsTotal = 0
	m.ConnectionsFailed = 0
	m.BytesTransferred = 0
	m.BytesReceived = 0
	m.BytesSent = 0
	m.ErrorsTotal = 0
	m.RequestsTotal = 0
	m.RequestDuration = 0
	m.NetworkErrors = 0
	m.ValidationErrors = 0
	m.SecurityErrors = 0
	m.TimeoutErrors = 0
	m.RetryAttempts = 0
	m.SuccessfulRetries = 0
	m.FailedRetries = 0
	m.StartTime = time.Now()
	m.LastActivity = time.Time{}
}
