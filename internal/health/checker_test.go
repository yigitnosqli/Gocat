package health

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"
)

// MockHealthChecker for testing
type MockHealthChecker struct {
	name     string
	timeout  time.Duration
	interval time.Duration
	result   HealthCheckResult
	checkFn  func(ctx context.Context) HealthCheckResult
}

func NewMockHealthChecker(name string, status HealthStatus) *MockHealthChecker {
	return &MockHealthChecker{
		name:     name,
		timeout:  time.Second,
		interval: time.Second,
		result: HealthCheckResult{
			Name:   name,
			Status: status,
		},
	}
}

func (m *MockHealthChecker) GetName() string {
	return m.name
}

func (m *MockHealthChecker) GetTimeout() time.Duration {
	return m.timeout
}

func (m *MockHealthChecker) GetInterval() time.Duration {
	return m.interval
}

func (m *MockHealthChecker) Check(ctx context.Context) HealthCheckResult {
	if m.checkFn != nil {
		return m.checkFn(ctx)
	}
	return m.result
}

func TestNewHealthManager(t *testing.T) {
	hm := NewHealthManager()

	if hm == nil {
		t.Fatal("Expected health manager to be created")
	}

	if hm.checkers == nil {
		t.Error("Expected checkers map to be initialized")
	}

	if hm.results == nil {
		t.Error("Expected results map to be initialized")
	}
}

func TestRegisterChecker(t *testing.T) {
	hm := NewHealthManager()
	checker := NewMockHealthChecker("test", StatusHealthy)

	hm.RegisterChecker(checker)

	names := hm.GetCheckerNames()
	if len(names) != 1 {
		t.Errorf("Expected 1 checker, got %d", len(names))
	}

	if names[0] != "test" {
		t.Errorf("Expected checker name 'test', got '%s'", names[0])
	}

	// Check initial result
	result, exists := hm.GetResult("test")
	if !exists {
		t.Error("Expected initial result to exist")
	}

	if result.Status != StatusUnknown {
		t.Errorf("Expected initial status to be unknown, got %s", result.Status)
	}
}

func TestUnregisterChecker(t *testing.T) {
	hm := NewHealthManager()
	checker := NewMockHealthChecker("test", StatusHealthy)

	hm.RegisterChecker(checker)
	hm.UnregisterChecker("test")

	names := hm.GetCheckerNames()
	if len(names) != 0 {
		t.Errorf("Expected 0 checkers after unregister, got %d", len(names))
	}

	_, exists := hm.GetResult("test")
	if exists {
		t.Error("Expected result to be removed after unregister")
	}
}

func TestCheckOne(t *testing.T) {
	hm := NewHealthManager()
	checker := NewMockHealthChecker("test", StatusHealthy)

	hm.RegisterChecker(checker)

	ctx := context.Background()
	result, err := hm.CheckOne(ctx, "test")

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if result.Status != StatusHealthy {
		t.Errorf("Expected status healthy, got %s", result.Status)
	}

	if result.Name != "test" {
		t.Errorf("Expected name 'test', got '%s'", result.Name)
	}
}

func TestCheckOneNotFound(t *testing.T) {
	hm := NewHealthManager()

	ctx := context.Background()
	_, err := hm.CheckOne(ctx, "nonexistent")

	if err == nil {
		t.Error("Expected error for nonexistent checker")
	}
}

func TestCheckAll(t *testing.T) {
	hm := NewHealthManager()

	checker1 := NewMockHealthChecker("test1", StatusHealthy)
	checker2 := NewMockHealthChecker("test2", StatusUnhealthy)

	hm.RegisterChecker(checker1)
	hm.RegisterChecker(checker2)

	ctx := context.Background()
	results := hm.CheckAll(ctx)

	if len(results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(results))
	}

	result1, exists := results["test1"]
	if !exists {
		t.Error("Expected result for test1")
	}
	if result1.Status != StatusHealthy {
		t.Errorf("Expected test1 to be healthy, got %s", result1.Status)
	}

	result2, exists := results["test2"]
	if !exists {
		t.Error("Expected result for test2")
	}
	if result2.Status != StatusUnhealthy {
		t.Errorf("Expected test2 to be unhealthy, got %s", result2.Status)
	}
}

func TestGetAggregatedHealth(t *testing.T) {
	hm := NewHealthManager()

	// All healthy
	checker1 := NewMockHealthChecker("test1", StatusHealthy)
	checker2 := NewMockHealthChecker("test2", StatusHealthy)

	hm.RegisterChecker(checker1)
	hm.RegisterChecker(checker2)

	ctx := context.Background()
	hm.CheckAll(ctx)

	aggregated := hm.GetAggregatedHealth()

	if aggregated.Status != StatusHealthy {
		t.Errorf("Expected overall status healthy, got %s", aggregated.Status)
	}

	if aggregated.Summary.Total != 2 {
		t.Errorf("Expected 2 total checks, got %d", aggregated.Summary.Total)
	}

	if aggregated.Summary.Healthy != 2 {
		t.Errorf("Expected 2 healthy checks, got %d", aggregated.Summary.Healthy)
	}
}

func TestGetAggregatedHealthWithUnhealthy(t *testing.T) {
	hm := NewHealthManager()

	checker1 := NewMockHealthChecker("test1", StatusHealthy)
	checker2 := NewMockHealthChecker("test2", StatusUnhealthy)

	hm.RegisterChecker(checker1)
	hm.RegisterChecker(checker2)

	ctx := context.Background()
	hm.CheckAll(ctx)

	aggregated := hm.GetAggregatedHealth()

	if aggregated.Status != StatusUnhealthy {
		t.Errorf("Expected overall status unhealthy, got %s", aggregated.Status)
	}

	if aggregated.Summary.Healthy != 1 {
		t.Errorf("Expected 1 healthy check, got %d", aggregated.Summary.Healthy)
	}

	if aggregated.Summary.Unhealthy != 1 {
		t.Errorf("Expected 1 unhealthy check, got %d", aggregated.Summary.Unhealthy)
	}
}

func TestGetAggregatedHealthWithDegraded(t *testing.T) {
	hm := NewHealthManager()

	checker1 := NewMockHealthChecker("test1", StatusHealthy)
	checker2 := NewMockHealthChecker("test2", StatusDegraded)

	hm.RegisterChecker(checker1)
	hm.RegisterChecker(checker2)

	ctx := context.Background()
	hm.CheckAll(ctx)

	aggregated := hm.GetAggregatedHealth()

	if aggregated.Status != StatusDegraded {
		t.Errorf("Expected overall status degraded, got %s", aggregated.Status)
	}

	if aggregated.Summary.Healthy != 1 {
		t.Errorf("Expected 1 healthy check, got %d", aggregated.Summary.Healthy)
	}

	if aggregated.Summary.Degraded != 1 {
		t.Errorf("Expected 1 degraded check, got %d", aggregated.Summary.Degraded)
	}
}

func TestHealthCheckTimeout(t *testing.T) {
	hm := NewHealthManager()

	// Create a checker that takes longer than timeout
	checker := &MockHealthChecker{
		name:     "slow",
		timeout:  100 * time.Millisecond,
		interval: time.Second,
		checkFn: func(ctx context.Context) HealthCheckResult {
			select {
			case <-time.After(200 * time.Millisecond):
				return HealthCheckResult{
					Name:   "slow",
					Status: StatusHealthy,
				}
			case <-ctx.Done():
				return HealthCheckResult{
					Name:   "slow",
					Status: StatusUnhealthy,
					Error:  "timeout",
				}
			}
		},
	}

	hm.RegisterChecker(checker)

	ctx := context.Background()
	result, err := hm.CheckOne(ctx, "slow")

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// The check should have been cancelled due to timeout
	if result.Status == StatusHealthy {
		t.Error("Expected check to be cancelled due to timeout")
	}
}

func TestHealthCheckPanicRecovery(t *testing.T) {
	hm := NewHealthManager()

	// Create a checker that panics
	checker := &MockHealthChecker{
		name:     "panic",
		timeout:  time.Second,
		interval: time.Second,
		checkFn: func(ctx context.Context) HealthCheckResult {
			panic("test panic")
		},
	}

	hm.RegisterChecker(checker)

	ctx := context.Background()
	result, err := hm.CheckOne(ctx, "panic")

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if result.Status != StatusUnhealthy {
		t.Errorf("Expected status unhealthy after panic, got %s", result.Status)
	}

	if result.Error == "" {
		t.Error("Expected error message after panic")
	}
}

func TestHealthCallback(t *testing.T) {
	hm := NewHealthManager()

	var callbackCalled bool
	var callbackResult HealthCheckResult
	var mu sync.Mutex

	hm.RegisterCallback(func(name string, result HealthCheckResult) {
		mu.Lock()
		callbackCalled = true
		callbackResult = result
		mu.Unlock()
	})

	checker := NewMockHealthChecker("test", StatusHealthy)
	hm.RegisterChecker(checker)

	ctx := context.Background()
	hm.CheckOne(ctx, "test")

	// Wait a bit for callback to be called
	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	called := callbackCalled
	result := callbackResult
	mu.Unlock()

	if !called {
		t.Error("Expected callback to be called")
	}

	if result.Name != "test" {
		t.Errorf("Expected callback result name 'test', got '%s'", result.Name)
	}
}

func TestIsHealthy(t *testing.T) {
	hm := NewHealthManager()

	// All healthy
	checker1 := NewMockHealthChecker("test1", StatusHealthy)
	checker2 := NewMockHealthChecker("test2", StatusHealthy)

	hm.RegisterChecker(checker1)
	hm.RegisterChecker(checker2)

	ctx := context.Background()
	hm.CheckAll(ctx)

	if !hm.IsHealthy() {
		t.Error("Expected IsHealthy to return true when all checks are healthy")
	}

	// Add unhealthy checker
	checker3 := NewMockHealthChecker("test3", StatusUnhealthy)
	hm.RegisterChecker(checker3)
	hm.CheckAll(ctx)

	if hm.IsHealthy() {
		t.Error("Expected IsHealthy to return false when any check is unhealthy")
	}
}

func TestGlobalFunctions(t *testing.T) {
	// Test global functions
	checker := NewMockHealthChecker("global_test", StatusHealthy)
	RegisterHealthChecker(checker)

	ctx := context.Background()
	results := CheckAllHealth(ctx)

	if len(results) == 0 {
		t.Error("Expected global health check to return results")
	}

	aggregated := GetGlobalHealth()
	if aggregated.Summary.Total == 0 {
		t.Error("Expected global aggregated health to have checks")
	}
}

func BenchmarkHealthCheck(b *testing.B) {
	hm := NewHealthManager()
	checker := NewMockHealthChecker("bench", StatusHealthy)
	hm.RegisterChecker(checker)

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		hm.CheckOne(ctx, "bench")
	}
}

func BenchmarkHealthCheckConcurrent(b *testing.B) {
	hm := NewHealthManager()

	// Register multiple checkers
	for i := 0; i < 10; i++ {
		checker := NewMockHealthChecker(fmt.Sprintf("bench_%d", i), StatusHealthy)
		hm.RegisterChecker(checker)
	}

	ctx := context.Background()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			hm.CheckAll(ctx)
		}
	})
}
