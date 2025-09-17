package memory

import (
	"runtime"
	"sync"
	"testing"
	"time"
)

func TestNewMemoryManager(t *testing.T) {
	config := DefaultMemoryConfig()
	mm := NewMemoryManager(config)
	defer mm.Close()

	if mm.maxMemoryMB != config.MaxMemoryMB {
		t.Errorf("Expected maxMemoryMB %d, got %d", config.MaxMemoryMB, mm.maxMemoryMB)
	}

	if mm.gcThresholdMB != config.GCThresholdMB {
		t.Errorf("Expected gcThresholdMB %d, got %d", config.GCThresholdMB, mm.gcThresholdMB)
	}

	// Should have pools initialized
	if mm.stringPool == nil {
		t.Error("Expected string pool to be initialized")
	}

	if mm.byteSlicePool == nil {
		t.Error("Expected byte slice pool to be initialized")
	}

	if mm.mapPool == nil {
		t.Error("Expected map pool to be initialized")
	}
}

func TestMemoryManagerMonitoring(t *testing.T) {
	config := DefaultMemoryConfig()
	config.MonitorInterval = 100 * time.Millisecond
	mm := NewMemoryManager(config)
	defer mm.Close()

	// Start monitoring
	mm.StartMonitoring()

	if !mm.isMonitoring {
		t.Error("Expected monitoring to be active")
	}

	// Wait for a few monitoring cycles
	time.Sleep(300 * time.Millisecond)

	stats := mm.GetStats()
	if stats.AllocMB == 0 {
		t.Error("Expected memory stats to be updated")
	}

	// Stop monitoring
	mm.StopMonitoring()

	if mm.isMonitoring {
		t.Error("Expected monitoring to be stopped")
	}
}

func TestStringPool(t *testing.T) {
	mm := NewMemoryManager(DefaultMemoryConfig())
	defer mm.Close()

	// Get string slice from pool
	slice1 := mm.GetStringSlice()
	if slice1 == nil {
		t.Fatal("Expected string slice from pool")
	}

	// Use the slice
	slice1 = append(slice1, "test1", "test2")

	// Return to pool
	mm.PutStringSlice(slice1)

	// Get another slice (should be reused)
	slice2 := mm.GetStringSlice()
	if slice2 == nil {
		t.Fatal("Expected string slice from pool")
	}

	// Should be empty after reset
	if len(slice2) != 0 {
		t.Errorf("Expected empty slice, got length %d", len(slice2))
	}

	mm.PutStringSlice(slice2)

	// Check stats
	stats := mm.GetStats()
	if stats.StringPoolHits < 2 {
		t.Errorf("Expected at least 2 string pool hits, got %d", stats.StringPoolHits)
	}
}

func TestByteSlicePool(t *testing.T) {
	mm := NewMemoryManager(DefaultMemoryConfig())
	defer mm.Close()

	// Get byte slice from pool
	slice1 := mm.GetByteSlice()
	if slice1 == nil {
		t.Fatal("Expected byte slice from pool")
	}

	// Use the slice
	slice1 = append(slice1, []byte("test data")...)

	// Return to pool
	mm.PutByteSlice(slice1)

	// Get another slice (should be reused)
	slice2 := mm.GetByteSlice()
	if slice2 == nil {
		t.Fatal("Expected byte slice from pool")
	}

	// Should be empty after reset
	if len(slice2) != 0 {
		t.Errorf("Expected empty slice, got length %d", len(slice2))
	}

	mm.PutByteSlice(slice2)

	// Check stats
	stats := mm.GetStats()
	if stats.BytePoolHits < 2 {
		t.Errorf("Expected at least 2 byte pool hits, got %d", stats.BytePoolHits)
	}
}

func TestMapPool(t *testing.T) {
	mm := NewMemoryManager(DefaultMemoryConfig())
	defer mm.Close()

	// Get map from pool
	map1 := mm.GetMap()
	if map1 == nil {
		t.Fatal("Expected map from pool")
	}

	// Use the map
	map1["key1"] = "value1"
	map1["key2"] = "value2"

	// Return to pool
	mm.PutMap(map1)

	// Get another map (should be reused)
	map2 := mm.GetMap()
	if map2 == nil {
		t.Fatal("Expected map from pool")
	}

	// Should be empty after reset
	if len(map2) != 0 {
		t.Errorf("Expected empty map, got length %d", len(map2))
	}

	mm.PutMap(map2)

	// Check stats
	stats := mm.GetStats()
	if stats.MapPoolHits < 2 {
		t.Errorf("Expected at least 2 map pool hits, got %d", stats.MapPoolHits)
	}
}

func TestForceGC(t *testing.T) {
	mm := NewMemoryManager(DefaultMemoryConfig())
	defer mm.Close()

	initialGCCount := mm.gcCount

	// Force GC
	mm.ForceGC()

	if mm.gcCount != initialGCCount+1 {
		t.Errorf("Expected GC count to increase by 1, got %d", mm.gcCount-initialGCCount)
	}

	// Should update last GC time
	if time.Since(mm.lastGCTime) > time.Second {
		t.Error("Expected lastGCTime to be recent")
	}
}

func TestMemoryPressureCallback(t *testing.T) {
	config := DefaultMemoryConfig()
	config.PressureThreshold = 0.1 // Very low threshold for testing
	mm := NewMemoryManager(config)
	defer mm.Close()

	var callbackCalled bool
	var callbackLevel float64
	var mu sync.Mutex

	// Register pressure callback
	mm.RegisterPressureCallback(func(level float64, stats MemoryStats) {
		mu.Lock()
		callbackCalled = true
		callbackLevel = level
		mu.Unlock()
	})

	// Start monitoring
	mm.StartMonitoring()

	// Allocate some memory to trigger pressure
	data := make([][]byte, 1000)
	for i := range data {
		data[i] = make([]byte, 1024*1024) // 1MB each
	}

	// Wait for monitoring to detect pressure
	time.Sleep(200 * time.Millisecond)

	mu.Lock()
	called := callbackCalled
	level := callbackLevel
	mu.Unlock()

	if !called {
		t.Error("Expected pressure callback to be called")
	}

	if level <= config.PressureThreshold {
		t.Errorf("Expected pressure level > %f, got %f", config.PressureThreshold, level)
	}

	// Clean up
	data = nil
	runtime.GC()
}

func TestTriggerGCIfNeeded(t *testing.T) {
	config := DefaultMemoryConfig()
	config.GCThresholdMB = 1 // Very low threshold for testing
	mm := NewMemoryManager(config)
	defer mm.Close()

	// Allocate memory to exceed threshold
	data := make([]byte, 2*1024*1024) // 2MB
	_ = data

	// Should trigger GC
	triggered := mm.TriggerGCIfNeeded()
	if !triggered {
		t.Error("Expected GC to be triggered")
	}

	// Clean up
	data = nil
	runtime.GC()
}

func TestMemoryPressureDetection(t *testing.T) {
	config := DefaultMemoryConfig()
	config.MaxMemoryMB = 10 // Very low limit for testing
	mm := NewMemoryManager(config)
	defer mm.Close()

	// Should not be under pressure initially
	if mm.IsUnderPressure() {
		t.Error("Expected not to be under pressure initially")
	}

	// Allocate memory to create pressure
	data := make([]byte, 8*1024*1024) // 8MB
	_ = data

	// Update stats manually
	mm.updateStats()

	// Should be under pressure now
	if !mm.IsUnderPressure() {
		t.Error("Expected to be under pressure after allocation")
	}

	pressure := mm.GetMemoryPressure()
	if pressure <= config.PressureThreshold {
		t.Errorf("Expected pressure > %f, got %f", config.PressureThreshold, pressure)
	}

	// Clean up
	data = nil
	runtime.GC()
}

func TestOptimizeGC(t *testing.T) {
	mm := NewMemoryManager(DefaultMemoryConfig())
	defer mm.Close()

	// Test GC optimization
	mm.OptimizeGC()

	// Should not panic and should complete
	// Actual GC percent changes are hard to test reliably
}

func TestConcurrentPoolUsage(t *testing.T) {
	mm := NewMemoryManager(DefaultMemoryConfig())
	defer mm.Close()

	const numGoroutines = 10
	const operationsPerGoroutine = 100

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()

			for j := 0; j < operationsPerGoroutine; j++ {
				// Test string pool
				strSlice := mm.GetStringSlice()
				strSlice = append(strSlice, "test")
				mm.PutStringSlice(strSlice)

				// Test byte pool
				byteSlice := mm.GetByteSlice()
				byteSlice = append(byteSlice, byte(j))
				mm.PutByteSlice(byteSlice)

				// Test map pool
				m := mm.GetMap()
				m["key"] = j
				mm.PutMap(m)
			}
		}()
	}

	wg.Wait()

	// Check that pools were used
	stats := mm.GetStats()
	expectedOps := int64(numGoroutines * operationsPerGoroutine)

	if stats.StringPoolHits < expectedOps {
		t.Errorf("Expected at least %d string pool hits, got %d", expectedOps, stats.StringPoolHits)
	}

	if stats.BytePoolHits < expectedOps {
		t.Errorf("Expected at least %d byte pool hits, got %d", expectedOps, stats.BytePoolHits)
	}

	if stats.MapPoolHits < expectedOps {
		t.Errorf("Expected at least %d map pool hits, got %d", expectedOps, stats.MapPoolHits)
	}
}

func BenchmarkStringPool(b *testing.B) {
	mm := NewMemoryManager(DefaultMemoryConfig())
	defer mm.Close()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			slice := mm.GetStringSlice()
			slice = append(slice, "benchmark")
			mm.PutStringSlice(slice)
		}
	})
}

func BenchmarkByteSlicePool(b *testing.B) {
	mm := NewMemoryManager(DefaultMemoryConfig())
	defer mm.Close()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			slice := mm.GetByteSlice()
			slice = append(slice, []byte("benchmark")...)
			mm.PutByteSlice(slice)
		}
	})
}

func BenchmarkMapPool(b *testing.B) {
	mm := NewMemoryManager(DefaultMemoryConfig())
	defer mm.Close()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			m := mm.GetMap()
			m["benchmark"] = true
			mm.PutMap(m)
		}
	})
}
