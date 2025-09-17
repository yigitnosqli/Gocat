package buffer

import (
	"sync"
	"testing"
)

func TestNewBufferPool(t *testing.T) {
	pool := NewBufferPool(1024, 65536, true)

	if pool.minSize != 1024 {
		t.Errorf("Expected minSize 1024, got %d", pool.minSize)
	}

	if pool.maxSize != 65536 {
		t.Errorf("Expected maxSize 65536, got %d", pool.maxSize)
	}

	if !pool.adaptive {
		t.Error("Expected adaptive to be true")
	}

	if len(pool.sizes) == 0 {
		t.Error("Expected default sizes to be initialized")
	}
}

func TestBufferPoolGetPut(t *testing.T) {
	pool := NewBufferPool(1024, 65536, false)

	// Test getting a buffer
	buffer := pool.Get(4096)
	if buffer == nil {
		t.Fatal("Expected buffer, got nil")
	}

	if len(buffer.Data) < 4096 {
		t.Errorf("Expected buffer size >= 4096, got %d", len(buffer.Data))
	}

	// Test putting buffer back
	pool.Put(buffer)

	// Verify statistics
	stats := pool.GetStats()
	if stats.TotalAllocations != 1 {
		t.Errorf("Expected 1 allocation, got %d", stats.TotalAllocations)
	}

	if stats.TotalDeallocations != 1 {
		t.Errorf("Expected 1 deallocation, got %d", stats.TotalDeallocations)
	}
}

func TestBufferPoolOptimalSize(t *testing.T) {
	pool := NewBufferPool(1024, 65536, false)

	testCases := []struct {
		requested int
		expected  int
	}{
		{500, 1024},    // Below minimum
		{2000, 4096},   // Between sizes
		{4096, 4096},   // Exact match
		{70000, 65536}, // Above maximum
	}

	for _, tc := range testCases {
		optimal := pool.findOptimalSize(tc.requested)
		if optimal != tc.expected {
			t.Errorf("For requested size %d, expected %d, got %d",
				tc.requested, tc.expected, optimal)
		}
	}
}

func TestBufferPoolConcurrency(t *testing.T) {
	pool := NewBufferPool(1024, 65536, true)

	const numGoroutines = 100
	const buffersPerGoroutine = 10

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()

			buffers := make([]*ManagedBuffer, buffersPerGoroutine)

			// Get buffers
			for j := 0; j < buffersPerGoroutine; j++ {
				buffers[j] = pool.Get(4096)
				if buffers[j] == nil {
					t.Errorf("Goroutine %d: failed to get buffer %d", id, j)
					return
				}
			}

			// Put buffers back
			for j := 0; j < buffersPerGoroutine; j++ {
				pool.Put(buffers[j])
			}
		}(i)
	}

	wg.Wait()

	// Verify final statistics
	stats := pool.GetStats()
	expectedAllocations := int64(numGoroutines * buffersPerGoroutine)

	if stats.TotalAllocations < expectedAllocations {
		t.Errorf("Expected at least %d allocations, got %d",
			expectedAllocations, stats.TotalAllocations)
	}

	if stats.ActiveBuffers != 0 {
		t.Errorf("Expected 0 active buffers, got %d", stats.ActiveBuffers)
	}
}

func TestBufferPoolAdaptiveSizing(t *testing.T) {
	pool := NewBufferPool(1024, 65536, true)

	connectionID := "test-conn-1"

	// Record some patterns
	sizes := []int{2048, 2048, 4096, 2048, 4096}
	for _, size := range sizes {
		pool.RecordConnectionPattern(connectionID, size)
	}

	// Get optimal size
	optimal := pool.GetOptimalSizeForConnection(connectionID)

	// Should be around the average (2867), rounded up to next pool size
	if optimal < 2048 || optimal > 4096 {
		t.Errorf("Expected optimal size between 2048-4096, got %d", optimal)
	}
}

func TestManagedBufferOperations(t *testing.T) {
	pool := NewBufferPool(1024, 65536, false)

	buffer := pool.Get(4096)

	// Test basic operations
	if buffer.Len() != 4096 {
		t.Errorf("Expected length 4096, got %d", buffer.Len())
	}

	if buffer.Cap() < 4096 {
		t.Errorf("Expected capacity >= 4096, got %d", buffer.Cap())
	}

	// Test resize
	resized := buffer.Resize(8192)
	if resized.Len() < 8192 {
		t.Errorf("Expected resized length >= 8192, got %d", resized.Len())
	}

	// Clean up
	resized.Release()
}

func TestBufferPoolCleanup(t *testing.T) {
	pool := NewBufferPool(1024, 65536, true)

	// Test that cleanup doesn't panic and updates last cleanup time
	oldCleanup := pool.stats.LastCleanup
	pool.Cleanup()

	if !pool.stats.LastCleanup.After(oldCleanup) {
		t.Error("Expected LastCleanup to be updated")
	}

	// Test cleanup with adaptive sizing
	if !pool.adaptive {
		t.Error("Expected pool to be adaptive")
	}
}

func TestBufferPoolStats(t *testing.T) {
	pool := NewBufferPool(1024, 65536, false)

	// Get and put some buffers
	buffer1 := pool.Get(4096)
	buffer2 := pool.Get(8192)

	stats := pool.GetStats()

	if stats.TotalAllocations != 2 {
		t.Errorf("Expected 2 allocations, got %d", stats.TotalAllocations)
	}

	if stats.ActiveBuffers != 2 {
		t.Errorf("Expected 2 active buffers, got %d", stats.ActiveBuffers)
	}

	pool.Put(buffer1)
	pool.Put(buffer2)

	stats = pool.GetStats()

	if stats.TotalDeallocations != 2 {
		t.Errorf("Expected 2 deallocations, got %d", stats.TotalDeallocations)
	}

	if stats.ActiveBuffers != 0 {
		t.Errorf("Expected 0 active buffers, got %d", stats.ActiveBuffers)
	}
}

func TestBufferPoolClose(t *testing.T) {
	pool := NewBufferPool(1024, 65536, true)

	// Get a buffer
	buffer := pool.Get(4096)

	// Close the pool
	pool.Close()

	// Verify pools are cleared
	if len(pool.pools) != 0 {
		t.Error("Expected pools to be cleared after close")
	}

	if len(pool.sizes) != 0 {
		t.Error("Expected sizes to be cleared after close")
	}

	// Putting buffer back should not panic
	pool.Put(buffer)
}

func BenchmarkBufferPoolGet(b *testing.B) {
	pool := NewBufferPool(1024, 65536, false)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			buffer := pool.Get(4096)
			pool.Put(buffer)
		}
	})
}

func BenchmarkBufferPoolGetLarge(b *testing.B) {
	pool := NewBufferPool(1024, 65536, false)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			buffer := pool.Get(32768)
			pool.Put(buffer)
		}
	})
}

func BenchmarkDirectAllocation(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			buffer := make([]byte, 4096)
			_ = buffer
		}
	})
}
