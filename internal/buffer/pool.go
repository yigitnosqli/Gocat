package buffer

import (
	"sync"
	"sync/atomic"
	"time"
)

// BufferPool manages a pool of reusable byte buffers with size-based allocation
type BufferPool struct {
	pools    map[int]*sync.Pool // Size-based pools
	sizes    []int              // Available buffer sizes
	stats    *PoolStats         // Pool statistics
	mu       sync.RWMutex       // Protects pools map and stats
	maxSize  int                // Maximum buffer size
	minSize  int                // Minimum buffer size
	adaptive bool               // Enable adaptive sizing
	patterns *SizePatterns      // Connection pattern tracking
}

// PoolStats tracks buffer pool usage statistics
type PoolStats struct {
	TotalAllocations   int64         // Total buffer allocations
	TotalDeallocations int64         // Total buffer returns
	ActiveBuffers      int64         // Currently active buffers
	PoolHits           int64         // Successful pool retrievals
	PoolMisses         int64         // Pool misses requiring new allocation
	BytesAllocated     int64         // Total bytes allocated
	BytesReused        int64         // Total bytes reused from pool
	LastCleanup        time.Time     // Last cleanup timestamp
	SizeDistribution   map[int]int64 // Distribution of buffer sizes used
}

// SizePatterns tracks connection patterns for adaptive sizing
type SizePatterns struct {
	mu              sync.RWMutex
	connectionSizes map[string][]int // Connection ID -> buffer sizes used
	avgSizes        map[string]int   // Connection type -> average size
	lastUpdate      time.Time
}

// BufferInfo contains metadata about a buffer
type BufferInfo struct {
	Size        int
	AllocatedAt time.Time
	UsageCount  int
	LastUsed    time.Time
}

// ManagedBuffer wraps a byte slice with metadata
type ManagedBuffer struct {
	Data []byte
	Info BufferInfo
	pool *BufferPool
}

// DefaultSizes defines standard buffer sizes for different use cases
var DefaultSizes = []int{
	1024,  // 1KB - small messages
	4096,  // 4KB - typical network buffer
	8192,  // 8KB - medium transfers
	16384, // 16KB - large transfers
	32768, // 32KB - bulk operations
	65536, // 64KB - maximum TCP window
}

// NewBufferPool creates a new buffer pool with specified configuration
func NewBufferPool(minSize, maxSize int, adaptive bool) *BufferPool {
	if minSize <= 0 {
		minSize = 1024
	}
	if maxSize <= minSize {
		maxSize = 65536
	}

	pool := &BufferPool{
		pools:    make(map[int]*sync.Pool),
		sizes:    make([]int, 0),
		minSize:  minSize,
		maxSize:  maxSize,
		adaptive: adaptive,
		stats: &PoolStats{
			SizeDistribution: make(map[int]int64),
		},
		patterns: &SizePatterns{
			connectionSizes: make(map[string][]int),
			avgSizes:        make(map[string]int),
			lastUpdate:      time.Now(),
		},
	}

	// Initialize with default sizes within range
	for _, size := range DefaultSizes {
		if size >= minSize && size <= maxSize {
			pool.addSizePool(size)
		}
	}

	return pool
}

// addSizePool adds a new size-specific pool
func (bp *BufferPool) addSizePool(size int) {
	bp.mu.Lock()
	defer bp.mu.Unlock()

	if _, exists := bp.pools[size]; exists {
		return
	}

	bp.pools[size] = &sync.Pool{
		New: func() interface{} {
			atomic.AddInt64(&bp.stats.PoolMisses, 1)
			atomic.AddInt64(&bp.stats.BytesAllocated, int64(size))
			return &ManagedBuffer{
				Data: make([]byte, size),
				Info: BufferInfo{
					Size:        size,
					AllocatedAt: time.Now(),
					UsageCount:  0,
				},
				pool: bp,
			}
		},
	}

	bp.sizes = append(bp.sizes, size)
	bp.stats.SizeDistribution[size] = 0
}

// Get retrieves a buffer of at least the specified size
func (bp *BufferPool) Get(size int) *ManagedBuffer {
	targetSize := bp.findOptimalSize(size)

	bp.mu.RLock()
	pool, exists := bp.pools[targetSize]
	bp.mu.RUnlock()

	if !exists {
		if bp.adaptive && targetSize <= bp.maxSize {
			bp.addSizePool(targetSize)
			bp.mu.RLock()
			pool = bp.pools[targetSize]
			bp.mu.RUnlock()
		} else {
			// Fallback to next larger size
			targetSize = bp.findOptimalSize(size)
			bp.mu.RLock()
			pool = bp.pools[targetSize]
			bp.mu.RUnlock()
		}
	}

	buffer := pool.Get().(*ManagedBuffer)
	buffer.Info.LastUsed = time.Now()
	buffer.Info.UsageCount++

	// Update statistics
	atomic.AddInt64(&bp.stats.TotalAllocations, 1)
	atomic.AddInt64(&bp.stats.ActiveBuffers, 1)
	atomic.AddInt64(&bp.stats.PoolHits, 1)
	atomic.AddInt64(&bp.stats.BytesReused, int64(targetSize))

	// Update size distribution safely
	bp.mu.Lock()
	bp.stats.SizeDistribution[targetSize]++
	bp.mu.Unlock()

	return buffer
}

// Put returns a buffer to the pool
func (bp *BufferPool) Put(buffer *ManagedBuffer) {
	if buffer == nil || buffer.pool != bp {
		return
	}

	// Reset buffer for reuse
	buffer.Data = buffer.Data[:cap(buffer.Data)]
	for i := range buffer.Data {
		buffer.Data[i] = 0
	}

	size := buffer.Info.Size
	bp.mu.RLock()
	pool, exists := bp.pools[size]
	bp.mu.RUnlock()

	if exists {
		pool.Put(buffer)
		atomic.AddInt64(&bp.stats.TotalDeallocations, 1)
		atomic.AddInt64(&bp.stats.ActiveBuffers, -1)
	}
}

// findOptimalSize finds the best buffer size for the requested size
func (bp *BufferPool) findOptimalSize(requestedSize int) int {
	if requestedSize > bp.maxSize {
		return bp.maxSize
	}
	if requestedSize < bp.minSize {
		return bp.minSize
	}

	// Find the smallest buffer size that can accommodate the request
	bp.mu.RLock()
	defer bp.mu.RUnlock()

	bestSize := bp.maxSize
	for _, size := range bp.sizes {
		if size >= requestedSize && size < bestSize {
			bestSize = size
		}
	}

	return bestSize
}

// RecordConnectionPattern records buffer usage patterns for a connection
func (bp *BufferPool) RecordConnectionPattern(connectionID string, bufferSize int) {
	if !bp.adaptive {
		return
	}

	bp.patterns.mu.Lock()
	defer bp.patterns.mu.Unlock()

	if bp.patterns.connectionSizes[connectionID] == nil {
		bp.patterns.connectionSizes[connectionID] = make([]int, 0, 10)
	}

	bp.patterns.connectionSizes[connectionID] = append(
		bp.patterns.connectionSizes[connectionID],
		bufferSize,
	)

	// Keep only recent patterns (last 10 sizes)
	if len(bp.patterns.connectionSizes[connectionID]) > 10 {
		bp.patterns.connectionSizes[connectionID] =
			bp.patterns.connectionSizes[connectionID][1:]
	}

	bp.patterns.lastUpdate = time.Now()
}

// GetOptimalSizeForConnection returns the optimal buffer size for a connection
func (bp *BufferPool) GetOptimalSizeForConnection(connectionID string) int {
	if !bp.adaptive {
		return bp.minSize
	}

	bp.patterns.mu.RLock()
	defer bp.patterns.mu.RUnlock()

	sizes, exists := bp.patterns.connectionSizes[connectionID]
	if !exists || len(sizes) == 0 {
		return bp.minSize
	}

	// Calculate average size
	total := 0
	for _, size := range sizes {
		total += size
	}
	avgSize := total / len(sizes)

	// Round up to next available pool size
	return bp.findOptimalSize(avgSize)
}

// GetStats returns current pool statistics
func (bp *BufferPool) GetStats() PoolStats {
	bp.mu.RLock()
	defer bp.mu.RUnlock()

	stats := PoolStats{
		TotalAllocations:   atomic.LoadInt64(&bp.stats.TotalAllocations),
		TotalDeallocations: atomic.LoadInt64(&bp.stats.TotalDeallocations),
		ActiveBuffers:      atomic.LoadInt64(&bp.stats.ActiveBuffers),
		PoolHits:           atomic.LoadInt64(&bp.stats.PoolHits),
		PoolMisses:         atomic.LoadInt64(&bp.stats.PoolMisses),
		BytesAllocated:     atomic.LoadInt64(&bp.stats.BytesAllocated),
		BytesReused:        atomic.LoadInt64(&bp.stats.BytesReused),
		LastCleanup:        bp.stats.LastCleanup,
		SizeDistribution:   make(map[int]int64),
	}

	// Copy size distribution
	for size, count := range bp.stats.SizeDistribution {
		stats.SizeDistribution[size] = count
	}

	return stats
}

// Cleanup performs garbage collection optimization and cleanup
func (bp *BufferPool) Cleanup() {
	bp.mu.Lock()
	defer bp.mu.Unlock()

	// Clear unused pools if adaptive sizing is enabled
	if bp.adaptive {
		now := time.Now()
		for size, pool := range bp.pools {
			// Check if this size hasn't been used recently
			if count := bp.stats.SizeDistribution[size]; count == 0 {
				// Remove unused pool after 5 minutes of inactivity
				if now.Sub(bp.stats.LastCleanup) > 5*time.Minute {
					delete(bp.pools, size)
					delete(bp.stats.SizeDistribution, size)

					// Remove from sizes slice
					for i, s := range bp.sizes {
						if s == size {
							bp.sizes = append(bp.sizes[:i], bp.sizes[i+1:]...)
							break
						}
					}
				}
			} else {
				// Reset counter for next cleanup cycle
				bp.stats.SizeDistribution[size] = 0
			}

			// Force garbage collection on the pool
			pool.New = pool.New
		}
	}

	bp.stats.LastCleanup = time.Now()
}

// Close shuts down the buffer pool and releases resources
func (bp *BufferPool) Close() {
	bp.mu.Lock()
	defer bp.mu.Unlock()

	// Clear all pools
	for size := range bp.pools {
		delete(bp.pools, size)
	}
	bp.sizes = nil

	// Clear patterns
	bp.patterns.mu.Lock()
	bp.patterns.connectionSizes = make(map[string][]int)
	bp.patterns.avgSizes = make(map[string]int)
	bp.patterns.mu.Unlock()
}

// Release returns the buffer to the pool (convenience method)
func (mb *ManagedBuffer) Release() {
	if mb.pool != nil {
		mb.pool.Put(mb)
	}
}

// Resize resizes the buffer if needed, potentially getting a new buffer from pool
func (mb *ManagedBuffer) Resize(newSize int) *ManagedBuffer {
	if newSize <= len(mb.Data) {
		mb.Data = mb.Data[:newSize]
		return mb
	}

	// Need a larger buffer
	newBuffer := mb.pool.Get(newSize)
	copy(newBuffer.Data, mb.Data)
	mb.Release()

	return newBuffer
}

// Bytes returns the underlying byte slice
func (mb *ManagedBuffer) Bytes() []byte {
	return mb.Data
}

// Cap returns the capacity of the buffer
func (mb *ManagedBuffer) Cap() int {
	return cap(mb.Data)
}

// Len returns the current length of the buffer
func (mb *ManagedBuffer) Len() int {
	return len(mb.Data)
}
