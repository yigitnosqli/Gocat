package health

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// HealthStatus represents the health status of a component
type HealthStatus string

const (
	StatusHealthy   HealthStatus = "healthy"
	StatusUnhealthy HealthStatus = "unhealthy"
	StatusDegraded  HealthStatus = "degraded"
	StatusUnknown   HealthStatus = "unknown"
)

// String returns the string representation of HealthStatus
func (hs HealthStatus) String() string {
	return string(hs)
}

// HealthChecker defines the interface for health checking
type HealthChecker interface {
	// Check performs a health check and returns the result
	Check(ctx context.Context) HealthCheckResult

	// GetName returns the name of the health check
	GetName() string

	// GetTimeout returns the timeout for this health check
	GetTimeout() time.Duration

	// GetInterval returns the check interval
	GetInterval() time.Duration
}

// HealthCheckResult represents the result of a health check
type HealthCheckResult struct {
	Name      string                 `json:"name"`
	Status    HealthStatus           `json:"status"`
	Message   string                 `json:"message,omitempty"`
	Error     string                 `json:"error,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
	Duration  time.Duration          `json:"duration"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// IsHealthy returns true if the status is healthy
func (hcr HealthCheckResult) IsHealthy() bool {
	return hcr.Status == StatusHealthy
}

// IsDegraded returns true if the status is degraded
func (hcr HealthCheckResult) IsDegraded() bool {
	return hcr.Status == StatusDegraded
}

// IsUnhealthy returns true if the status is unhealthy
func (hcr HealthCheckResult) IsUnhealthy() bool {
	return hcr.Status == StatusUnhealthy
}

// HealthManager manages multiple health checkers
type HealthManager struct {
	checkers map[string]HealthChecker
	results  map[string]HealthCheckResult
	mu       sync.RWMutex

	// Monitoring
	isMonitoring bool
	stopChan     chan struct{}

	// Configuration
	globalTimeout time.Duration

	// Callbacks
	callbacks []HealthCallback
}

// HealthCallback is called when health status changes
type HealthCallback func(name string, result HealthCheckResult)

// AggregatedHealth represents the overall health status
type AggregatedHealth struct {
	Status    HealthStatus                 `json:"status"`
	Message   string                       `json:"message,omitempty"`
	Timestamp time.Time                    `json:"timestamp"`
	Checks    map[string]HealthCheckResult `json:"checks"`
	Summary   HealthSummary                `json:"summary"`
}

// HealthSummary provides a summary of health check results
type HealthSummary struct {
	Total     int `json:"total"`
	Healthy   int `json:"healthy"`
	Degraded  int `json:"degraded"`
	Unhealthy int `json:"unhealthy"`
	Unknown   int `json:"unknown"`
}

// NewHealthManager creates a new health manager
func NewHealthManager() *HealthManager {
	return &HealthManager{
		checkers:      make(map[string]HealthChecker),
		results:       make(map[string]HealthCheckResult),
		globalTimeout: 30 * time.Second,
		stopChan:      make(chan struct{}),
		callbacks:     make([]HealthCallback, 0),
	}
}

// RegisterChecker registers a health checker
func (hm *HealthManager) RegisterChecker(checker HealthChecker) {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	name := checker.GetName()
	hm.checkers[name] = checker

	// Initialize with unknown status
	hm.results[name] = HealthCheckResult{
		Name:      name,
		Status:    StatusUnknown,
		Timestamp: time.Now(),
	}
}

// UnregisterChecker unregisters a health checker
func (hm *HealthManager) UnregisterChecker(name string) {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	delete(hm.checkers, name)
	delete(hm.results, name)
}

// CheckAll performs all health checks
func (hm *HealthManager) CheckAll(ctx context.Context) map[string]HealthCheckResult {
	hm.mu.RLock()
	checkers := make(map[string]HealthChecker)
	for name, checker := range hm.checkers {
		checkers[name] = checker
	}
	hm.mu.RUnlock()

	results := make(map[string]HealthCheckResult)
	var wg sync.WaitGroup
	var mu sync.Mutex

	for name, checker := range checkers {
		wg.Add(1)
		go func(name string, checker HealthChecker) {
			defer wg.Done()

			// Create context with timeout
			timeout := checker.GetTimeout()
			if timeout == 0 {
				timeout = hm.globalTimeout
			}

			checkCtx, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()

			result := hm.performCheck(checkCtx, checker)

			mu.Lock()
			results[name] = result
			mu.Unlock()

			// Store result and trigger callbacks
			hm.storeResult(name, result)
		}(name, checker)
	}

	wg.Wait()
	return results
}

// CheckOne performs a single health check
func (hm *HealthManager) CheckOne(ctx context.Context, name string) (HealthCheckResult, error) {
	hm.mu.RLock()
	checker, exists := hm.checkers[name]
	hm.mu.RUnlock()

	if !exists {
		return HealthCheckResult{}, fmt.Errorf("health checker %q not found", name)
	}

	// Create context with timeout
	timeout := checker.GetTimeout()
	if timeout == 0 {
		timeout = hm.globalTimeout
	}

	checkCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	result := hm.performCheck(checkCtx, checker)
	hm.storeResult(name, result)

	return result, nil
}

// performCheck performs a single health check with error handling
func (hm *HealthManager) performCheck(ctx context.Context, checker HealthChecker) HealthCheckResult {
	start := time.Now()

	defer func() {
		if r := recover(); r != nil {
			// Handle panic in health check
			result := HealthCheckResult{
				Name:      checker.GetName(),
				Status:    StatusUnhealthy,
				Error:     fmt.Sprintf("panic during health check: %v", r),
				Timestamp: time.Now(),
				Duration:  time.Since(start),
			}
			hm.storeResult(checker.GetName(), result)
		}
	}()

	result := checker.Check(ctx)
	result.Duration = time.Since(start)
	result.Timestamp = time.Now()

	// Ensure name is set
	if result.Name == "" {
		result.Name = checker.GetName()
	}

	return result
}

// storeResult stores a health check result and triggers callbacks
func (hm *HealthManager) storeResult(name string, result HealthCheckResult) {
	hm.mu.Lock()
	oldResult, exists := hm.results[name]
	hm.results[name] = result
	callbacks := make([]HealthCallback, len(hm.callbacks))
	copy(callbacks, hm.callbacks)
	hm.mu.Unlock()

	// Trigger callbacks if status changed
	if !exists || oldResult.Status != result.Status {
		for _, callback := range callbacks {
			go callback(name, result)
		}
	}
}

// GetAggregatedHealth returns the overall health status
func (hm *HealthManager) GetAggregatedHealth() AggregatedHealth {
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	checks := make(map[string]HealthCheckResult)
	for name, result := range hm.results {
		checks[name] = result
	}

	summary := HealthSummary{
		Total: len(checks),
	}

	overallStatus := StatusHealthy
	var messages []string

	for _, result := range checks {
		switch result.Status {
		case StatusHealthy:
			summary.Healthy++
		case StatusDegraded:
			summary.Degraded++
			if overallStatus == StatusHealthy {
				overallStatus = StatusDegraded
			}
		case StatusUnhealthy:
			summary.Unhealthy++
			overallStatus = StatusUnhealthy
			if result.Error != "" {
				messages = append(messages, fmt.Sprintf("%s: %s", result.Name, result.Error))
			}
		case StatusUnknown:
			summary.Unknown++
			if overallStatus == StatusHealthy {
				overallStatus = StatusUnknown
			}
		}
	}

	message := ""
	if len(messages) > 0 {
		message = fmt.Sprintf("Unhealthy components: %v", messages)
	}

	return AggregatedHealth{
		Status:    overallStatus,
		Message:   message,
		Timestamp: time.Now(),
		Checks:    checks,
		Summary:   summary,
	}
}

// GetResult returns the latest result for a specific checker
func (hm *HealthManager) GetResult(name string) (HealthCheckResult, bool) {
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	result, exists := hm.results[name]
	return result, exists
}

// GetAllResults returns all latest results
func (hm *HealthManager) GetAllResults() map[string]HealthCheckResult {
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	results := make(map[string]HealthCheckResult)
	for name, result := range hm.results {
		results[name] = result
	}

	return results
}

// StartMonitoring starts continuous health monitoring
func (hm *HealthManager) StartMonitoring(ctx context.Context) {
	hm.mu.Lock()
	if hm.isMonitoring {
		hm.mu.Unlock()
		return
	}
	hm.isMonitoring = true
	hm.mu.Unlock()

	go hm.monitoringLoop(ctx)
}

// StopMonitoring stops continuous health monitoring
func (hm *HealthManager) StopMonitoring() {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	if !hm.isMonitoring {
		return
	}

	hm.isMonitoring = false
	close(hm.stopChan)
	hm.stopChan = make(chan struct{})
}

// monitoringLoop runs the continuous monitoring
func (hm *HealthManager) monitoringLoop(ctx context.Context) {
	// Create individual timers for each checker
	hm.mu.RLock()
	checkers := make(map[string]HealthChecker)
	for name, checker := range hm.checkers {
		checkers[name] = checker
	}
	hm.mu.RUnlock()

	timers := make(map[string]*time.Timer)

	// Start timers for each checker
	for name, checker := range checkers {
		interval := checker.GetInterval()
		if interval == 0 {
			interval = 30 * time.Second // Default interval
		}

		timer := time.NewTimer(interval)
		timers[name] = timer

		go func(name string, checker HealthChecker, timer *time.Timer) {
			for {
				select {
				case <-timer.C:
					// Perform health check
					timeout := checker.GetTimeout()
					if timeout == 0 {
						timeout = hm.globalTimeout
					}

					checkCtx, cancel := context.WithTimeout(ctx, timeout)
					result := hm.performCheck(checkCtx, checker)
					cancel()

					hm.storeResult(name, result)

					// Reset timer
					timer.Reset(checker.GetInterval())

				case <-hm.stopChan:
					timer.Stop()
					return
				case <-ctx.Done():
					timer.Stop()
					return
				}
			}
		}(name, checker, timer)
	}

	// Wait for stop signal
	select {
	case <-hm.stopChan:
	case <-ctx.Done():
	}

	// Clean up timers
	for _, timer := range timers {
		timer.Stop()
	}
}

// RegisterCallback registers a health status change callback
func (hm *HealthManager) RegisterCallback(callback HealthCallback) {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	hm.callbacks = append(hm.callbacks, callback)
}

// SetGlobalTimeout sets the global timeout for health checks
func (hm *HealthManager) SetGlobalTimeout(timeout time.Duration) {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	hm.globalTimeout = timeout
}

// GetCheckerNames returns the names of all registered checkers
func (hm *HealthManager) GetCheckerNames() []string {
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	names := make([]string, 0, len(hm.checkers))
	for name := range hm.checkers {
		names = append(names, name)
	}

	return names
}

// IsHealthy returns true if all components are healthy
func (hm *HealthManager) IsHealthy() bool {
	aggregated := hm.GetAggregatedHealth()
	return aggregated.Status == StatusHealthy
}

// Global health manager instance
var globalHealthManager = NewHealthManager()

// Global functions

// RegisterHealthChecker registers a health checker globally
func RegisterHealthChecker(checker HealthChecker) {
	globalHealthManager.RegisterChecker(checker)
}

// CheckAllHealth performs all health checks globally
func CheckAllHealth(ctx context.Context) map[string]HealthCheckResult {
	return globalHealthManager.CheckAll(ctx)
}

// GetGlobalHealth returns the global aggregated health
func GetGlobalHealth() AggregatedHealth {
	return globalHealthManager.GetAggregatedHealth()
}

// StartGlobalMonitoring starts global health monitoring
func StartGlobalMonitoring(ctx context.Context) {
	globalHealthManager.StartMonitoring(ctx)
}

// StopGlobalMonitoring stops global health monitoring
func StopGlobalMonitoring() {
	globalHealthManager.StopMonitoring()
}

// GetGlobalHealthManager returns the global health manager
func GetGlobalHealthManager() *HealthManager {
	return globalHealthManager
}
