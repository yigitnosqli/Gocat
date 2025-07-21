package metrics

import (
	"testing"
	"time"
)

func TestNewMetrics(t *testing.T) {
	m := NewMetrics()
	if m == nil {
		t.Fatal("NewMetrics returned nil")
	}

	if m.StartTime.IsZero() {
		t.Error("StartTime should be set")
	}

	// All counters should start at zero
	if m.ConnectionsActive != 0 {
		t.Errorf("Expected ConnectionsActive to be 0, got %d", m.ConnectionsActive)
	}
	if m.ConnectionsTotal != 0 {
		t.Errorf("Expected ConnectionsTotal to be 0, got %d", m.ConnectionsTotal)
	}
	if m.ErrorsTotal != 0 {
		t.Errorf("Expected ErrorsTotal to be 0, got %d", m.ErrorsTotal)
	}
}

func TestIncrementConnectionsActive(t *testing.T) {
	m := NewMetrics()

	m.IncrementConnectionsActive()
	if m.ConnectionsActive != 1 {
		t.Errorf("Expected ConnectionsActive to be 1, got %d", m.ConnectionsActive)
	}
	if m.ConnectionsTotal != 1 {
		t.Errorf("Expected ConnectionsTotal to be 1, got %d", m.ConnectionsTotal)
	}

	m.IncrementConnectionsActive()
	if m.ConnectionsActive != 2 {
		t.Errorf("Expected ConnectionsActive to be 2, got %d", m.ConnectionsActive)
	}
	if m.ConnectionsTotal != 2 {
		t.Errorf("Expected ConnectionsTotal to be 2, got %d", m.ConnectionsTotal)
	}
}

func TestDecrementConnectionsActive(t *testing.T) {
	m := NewMetrics()

	// Increment first
	m.IncrementConnectionsActive()
	m.IncrementConnectionsActive()

	// Then decrement
	m.DecrementConnectionsActive()
	if m.ConnectionsActive != 1 {
		t.Errorf("Expected ConnectionsActive to be 1, got %d", m.ConnectionsActive)
	}

	// Decrement to zero
	m.DecrementConnectionsActive()
	if m.ConnectionsActive != 0 {
		t.Errorf("Expected ConnectionsActive to be 0, got %d", m.ConnectionsActive)
	}

	// Should not go below zero
	m.DecrementConnectionsActive()
	if m.ConnectionsActive != 0 {
		t.Errorf("Expected ConnectionsActive to stay at 0, got %d", m.ConnectionsActive)
	}
}

func TestAddBytesTransferred(t *testing.T) {
	m := NewMetrics()

	m.AddBytesTransferred(100)
	if m.BytesTransferred != 100 {
		t.Errorf("Expected BytesTransferred to be 100, got %d", m.BytesTransferred)
	}

	m.AddBytesTransferred(50)
	if m.BytesTransferred != 150 {
		t.Errorf("Expected BytesTransferred to be 150, got %d", m.BytesTransferred)
	}
}

func TestAddBytesReceived(t *testing.T) {
	m := NewMetrics()

	m.AddBytesReceived(200)
	if m.BytesReceived != 200 {
		t.Errorf("Expected BytesReceived to be 200, got %d", m.BytesReceived)
	}
	if m.BytesTransferred != 200 {
		t.Errorf("Expected BytesTransferred to be 200, got %d", m.BytesTransferred)
	}
}

func TestAddBytesSent(t *testing.T) {
	m := NewMetrics()

	m.AddBytesSent(300)
	if m.BytesSent != 300 {
		t.Errorf("Expected BytesSent to be 300, got %d", m.BytesSent)
	}
	if m.BytesTransferred != 300 {
		t.Errorf("Expected BytesTransferred to be 300, got %d", m.BytesTransferred)
	}
}

func TestIncrementErrors(t *testing.T) {
	m := NewMetrics()

	m.IncrementErrors("network")
	if m.ErrorsTotal != 1 {
		t.Errorf("Expected ErrorsTotal to be 1, got %d", m.ErrorsTotal)
	}
	if m.NetworkErrors != 1 {
		t.Errorf("Expected NetworkErrors to be 1, got %d", m.NetworkErrors)
	}

	m.IncrementErrors("validation")
	if m.ErrorsTotal != 2 {
		t.Errorf("Expected ErrorsTotal to be 2, got %d", m.ErrorsTotal)
	}
	if m.ValidationErrors != 1 {
		t.Errorf("Expected ValidationErrors to be 1, got %d", m.ValidationErrors)
	}

	m.IncrementErrors("security")
	if m.SecurityErrors != 1 {
		t.Errorf("Expected SecurityErrors to be 1, got %d", m.SecurityErrors)
	}

	m.IncrementErrors("timeout")
	if m.TimeoutErrors != 1 {
		t.Errorf("Expected TimeoutErrors to be 1, got %d", m.TimeoutErrors)
	}

	// Unknown error type should still increment total
	m.IncrementErrors("unknown")
	if m.ErrorsTotal != 5 {
		t.Errorf("Expected ErrorsTotal to be 5, got %d", m.ErrorsTotal)
	}
}

func TestRecordRequestDuration(t *testing.T) {
	m := NewMetrics()

	// First request
	m.IncrementRequests()
	m.RecordRequestDuration(100 * time.Millisecond)
	if m.RequestDuration != 100*time.Millisecond {
		t.Errorf("Expected RequestDuration to be 100ms, got %v", m.RequestDuration)
	}

	// Second request (should average)
	m.IncrementRequests()
	m.RecordRequestDuration(200 * time.Millisecond)
	expected := 150 * time.Millisecond // Average of 100ms and 200ms
	if m.RequestDuration != expected {
		t.Errorf("Expected RequestDuration to be %v, got %v", expected, m.RequestDuration)
	}
}

func TestGetSnapshot(t *testing.T) {
	m := NewMetrics()

	// Add some data
	m.IncrementConnectionsActive()
	m.IncrementConnectionsFailed()
	m.AddBytesTransferred(1000)
	m.IncrementErrors("network")
	m.IncrementRequests()

	snapshot := m.GetSnapshot()

	if snapshot.ConnectionsActive != 1 {
		t.Errorf("Expected snapshot ConnectionsActive to be 1, got %d", snapshot.ConnectionsActive)
	}
	if snapshot.ConnectionsFailed != 1 {
		t.Errorf("Expected snapshot ConnectionsFailed to be 1, got %d", snapshot.ConnectionsFailed)
	}
	if snapshot.BytesTransferred != 1000 {
		t.Errorf("Expected snapshot BytesTransferred to be 1000, got %d", snapshot.BytesTransferred)
	}
	if snapshot.ErrorsTotal != 1 {
		t.Errorf("Expected snapshot ErrorsTotal to be 1, got %d", snapshot.ErrorsTotal)
	}
	if snapshot.RequestsTotal != 1 {
		t.Errorf("Expected snapshot RequestsTotal to be 1, got %d", snapshot.RequestsTotal)
	}

	if snapshot.Timestamp.IsZero() {
		t.Error("Snapshot timestamp should be set")
	}
	if snapshot.Uptime <= 0 {
		t.Error("Snapshot uptime should be positive")
	}
}

func TestMetricsSnapshotCalculations(t *testing.T) {
	snapshot := MetricsSnapshot{
		ConnectionsTotal:  100,
		ConnectionsFailed: 10,
		RequestsTotal:     200,
		ErrorsTotal:       20,
		RetryAttempts:     50,
		SuccessfulRetries: 40,
		BytesTransferred:  10000,
		Uptime:            10 * time.Second,
	}

	// Test success rate
	successRate := snapshot.SuccessRate()
	expectedSuccessRate := 90.0 // (100-10)/100 * 100
	if successRate != expectedSuccessRate {
		t.Errorf("Expected success rate %.1f%%, got %.1f%%", expectedSuccessRate, successRate)
	}

	// Test error rate
	errorRate := snapshot.ErrorRate()
	expectedErrorRate := 10.0 // 20/200 * 100
	if errorRate != expectedErrorRate {
		t.Errorf("Expected error rate %.1f%%, got %.1f%%", expectedErrorRate, errorRate)
	}

	// Test retry success rate
	retrySuccessRate := snapshot.RetrySuccessRate()
	expectedRetrySuccessRate := 80.0 // 40/50 * 100
	if retrySuccessRate != expectedRetrySuccessRate {
		t.Errorf("Expected retry success rate %.1f%%, got %.1f%%", expectedRetrySuccessRate, retrySuccessRate)
	}

	// Test throughput
	throughput := snapshot.ThroughputBytesPerSecond()
	expectedThroughput := 1000.0 // 10000/10
	if throughput != expectedThroughput {
		t.Errorf("Expected throughput %.1f bytes/sec, got %.1f bytes/sec", expectedThroughput, throughput)
	}

	// Test requests per second
	rps := snapshot.RequestsPerSecond()
	expectedRPS := 20.0 // 200/10
	if rps != expectedRPS {
		t.Errorf("Expected RPS %.1f, got %.1f", expectedRPS, rps)
	}
}

func TestMetricsSnapshotEdgeCases(t *testing.T) {
	// Test with zero values
	snapshot := MetricsSnapshot{}

	if snapshot.SuccessRate() != 0 {
		t.Error("Success rate should be 0 when no connections")
	}
	if snapshot.ErrorRate() != 0 {
		t.Error("Error rate should be 0 when no requests")
	}
	if snapshot.RetrySuccessRate() != 0 {
		t.Error("Retry success rate should be 0 when no retries")
	}
	if snapshot.ThroughputBytesPerSecond() != 0 {
		t.Error("Throughput should be 0 when uptime is 0")
	}
	if snapshot.RequestsPerSecond() != 0 {
		t.Error("RPS should be 0 when uptime is 0")
	}
}

func TestReset(t *testing.T) {
	m := NewMetrics()

	// Add some data
	m.IncrementConnectionsActive()
	m.IncrementConnectionsFailed()
	m.AddBytesTransferred(1000)
	m.IncrementErrors("network")
	m.IncrementRequests()

	// Reset
	m.Reset()

	// Check all values are zero
	if m.ConnectionsActive != 0 {
		t.Errorf("Expected ConnectionsActive to be 0 after reset, got %d", m.ConnectionsActive)
	}
	if m.ConnectionsTotal != 0 {
		t.Errorf("Expected ConnectionsTotal to be 0 after reset, got %d", m.ConnectionsTotal)
	}
	if m.BytesTransferred != 0 {
		t.Errorf("Expected BytesTransferred to be 0 after reset, got %d", m.BytesTransferred)
	}
	if m.ErrorsTotal != 0 {
		t.Errorf("Expected ErrorsTotal to be 0 after reset, got %d", m.ErrorsTotal)
	}
	if m.RequestsTotal != 0 {
		t.Errorf("Expected RequestsTotal to be 0 after reset, got %d", m.RequestsTotal)
	}

	// StartTime should be updated
	if m.StartTime.IsZero() {
		t.Error("StartTime should be set after reset")
	}
}

func TestGetGlobalMetrics(t *testing.T) {
	m1 := GetGlobalMetrics()
	m2 := GetGlobalMetrics()

	// Should return the same instance
	if m1 != m2 {
		t.Error("GetGlobalMetrics should return the same instance")
	}

	// Test that it's functional
	m1.IncrementConnectionsActive()
	if m2.ConnectionsActive != 1 {
		t.Error("Global metrics should be shared")
	}
}

func BenchmarkIncrementConnectionsActive(b *testing.B) {
	m := NewMetrics()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		m.IncrementConnectionsActive()
	}
}

func BenchmarkAddBytesTransferred(b *testing.B) {
	m := NewMetrics()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		m.AddBytesTransferred(1024)
	}
}

func BenchmarkGetSnapshot(b *testing.B) {
	m := NewMetrics()
	// Add some data first
	m.IncrementConnectionsActive()
	m.AddBytesTransferred(1000)
	m.IncrementErrors("network")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.GetSnapshot()
	}
}
