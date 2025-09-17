package memory

import (
	"runtime"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"
)

// MemoryManager handles memory optimization and garbage collection
type MemoryManager struct {
	// Configuration
	maxMemoryMB       int64         // Maximum memory usage in MB
	gcThresholdMB     int64         // GC threshold in MB
	monitorInterval   time.Duration // Memory monitoring interval
	pressureThreshold float64       // Memory pressure threshold (0.0-1.0)

	// State
	isMonitoring bool      // Whether monitoring is active
	lastGCTime   time.Time // Last GC time
	gcCount      int64     // Number of forced GCs

	// Statistics
	stats *MemoryStats // Memory statistics

	// Object pools for frequently allocated objects
	stringPool    *sync.Pool // String pool
	byteSlicePool *sync.Pool // Byte slice pool
	mapPool       *sync.Pool // Map pool

	// Synchronization
	mu       sync.RWMutex  // Protects manager state
	stopChan chan struct{} // Stop monitoring channel

	// Callbacks
	pressureCallbacks []PressureCallback // Memory pressure callbacks
}

// MemoryStats tracks memory usage statistics
type MemoryStats struct {
	// Current memory usage
	AllocMB int64 // Currently allocated memory in MB
	SysMB   int64 // System memory in MB
	HeapMB  int64 // Heap memory in MB
	StackMB int64 // Stack memory in MB

	// GC statistics
	NumGC        uint32 // Number of GC cycles
	PauseNs      uint64 // Last GC pause in nanoseconds
	TotalPauseNs uint64 // Total GC pause time

	// Memory pressure
	PressureLevel    float64   // Current memory pressure (0.0-1.0)
	LastPressureTime time.Time // Last time pressure was detected

	// Pool statistics
	StringPoolHits   int64 // String pool hits
	StringPoolMisses int64 // String pool misses
	BytePoolHits     int64 // Byte pool hits
	BytePoolMisses   int64 // Byte pool misses
	MapPoolHits      int64 // Map pool hits
	MapPoolMisses    int64 // Map pool misses

	// Leak detection
	SuspectedLeaks int64     // Number of suspected memory leaks
	LastLeakCheck  time.Time // Last leak detection check
}

// PressureCallback is called when memory pressure is detected
type PressureCallback func(level float64, stats MemoryStats)

// MemoryConfig contains configuration for the memory manager
type MemoryConfig struct {
	MaxMemoryMB       int64
	GCThresholdMB     int64
	MonitorInterval   time.Duration
	PressureThreshold float64
	EnablePools       bool
}

// DefaultMemoryConfig returns a default memory configuration
func DefaultMemoryConfig() *MemoryConfig {
	return &MemoryConfig{
		MaxMemoryMB:       1024, // 1GB default
		GCThresholdMB:     512,  // 512MB GC threshold
		MonitorInterval:   5 * time.Second,
		PressureThreshold: 0.8, // 80% memory pressure threshold
		EnablePools:       true,
	}
}

// NewMemoryManager creates a new memory manager
func NewMemoryManager(config *MemoryConfig) *MemoryManager {
	if config == nil {
		config = DefaultMemoryConfig()
	}

	mm := &MemoryManager{
		maxMemoryMB:       config.MaxMemoryMB,
		gcThresholdMB:     config.GCThresholdMB,
		monitorInterval:   config.MonitorInterval,
		pressureThreshold: config.PressureThreshold,
		lastGCTime:        time.Now(),
		stats:             &MemoryStats{},
		stopChan:          make(chan struct{}),
		pressureCallbacks: make([]PressureCallback, 0),
	}

	// Initialize object pools if enabled
	if config.EnablePools {
		mm.initializePools()
	}

	return mm
}

// initializePools initializes object pools for memory optimization
func (mm *MemoryManager) initializePools() {
	// String pool for frequently used strings
	mm.stringPool = &sync.Pool{
		New: func() interface{} {
			atomic.AddInt64(&mm.stats.StringPoolMisses, 1)
			return make([]string, 0, 10)
		},
	}

	// Byte slice pool for network buffers
	mm.byteSlicePool = &sync.Pool{
		New: func() interface{} {
			atomic.AddInt64(&mm.stats.BytePoolMisses, 1)
			return make([]byte, 0, 4096)
		},
	}

	// Map pool for temporary maps
	mm.mapPool = &sync.Pool{
		New: func() interface{} {
			atomic.AddInt64(&mm.stats.MapPoolMisses, 1)
			return make(map[string]interface{})
		},
	}
}

// StartMonitoring starts memory monitoring
func (mm *MemoryManager) StartMonitoring() {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	if mm.isMonitoring {
		return
	}

	mm.isMonitoring = true
	go mm.monitorLoop()
}

// StopMonitoring stops memory monitoring
func (mm *MemoryManager) StopMonitoring() {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	if !mm.isMonitoring {
		return
	}

	mm.isMonitoring = false
	close(mm.stopChan)
	mm.stopChan = make(chan struct{})
}

// monitorLoop is the main monitoring loop
func (mm *MemoryManager) monitorLoop() {
	ticker := time.NewTicker(mm.monitorInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			mm.updateStats()
			mm.checkMemoryPressure()
			mm.checkForLeaks()

		case <-mm.stopChan:
			return
		}
	}
}

// updateStats updates memory statistics
func (mm *MemoryManager) updateStats() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	mm.mu.Lock()
	defer mm.mu.Unlock()

	// Update memory usage
	mm.stats.AllocMB = int64(m.Alloc / 1024 / 1024)
	mm.stats.SysMB = int64(m.Sys / 1024 / 1024)
	mm.stats.HeapMB = int64(m.HeapAlloc / 1024 / 1024)
	mm.stats.StackMB = int64(m.StackSys / 1024 / 1024)

	// Update GC statistics
	mm.stats.NumGC = m.NumGC
	if len(m.PauseNs) > 0 {
		mm.stats.PauseNs = m.PauseNs[(m.NumGC+255)%256]
	}
	mm.stats.TotalPauseNs = m.PauseTotalNs

	// Calculate memory pressure
	if mm.maxMemoryMB > 0 {
		mm.stats.PressureLevel = float64(mm.stats.AllocMB) / float64(mm.maxMemoryMB)
	}
}

// checkMemoryPressure checks for memory pressure and triggers callbacks
func (mm *MemoryManager) checkMemoryPressure() {
	mm.mu.RLock()
	pressureLevel := mm.stats.PressureLevel
	stats := *mm.stats
	mm.mu.RUnlock()

	if pressureLevel > mm.pressureThreshold {
		mm.mu.Lock()
		mm.stats.LastPressureTime = time.Now()
		mm.mu.Unlock()

		// Trigger pressure callbacks
		for _, callback := range mm.pressureCallbacks {
			go callback(pressureLevel, stats)
		}

		// Force GC if memory usage is high
		if mm.stats.AllocMB > mm.gcThresholdMB {
			mm.ForceGC()
		}
	}
}

// checkForLeaks performs basic memory leak detection
func (mm *MemoryManager) checkForLeaks() {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	now := time.Now()

	// Check if memory is consistently growing
	if now.Sub(mm.stats.LastLeakCheck) > 30*time.Second {
		// Simple heuristic: if memory usage increased significantly without GC
		if mm.stats.AllocMB > mm.maxMemoryMB/2 &&
			time.Since(mm.lastGCTime) > time.Minute {
			mm.stats.SuspectedLeaks++
		}

		mm.stats.LastLeakCheck = now
	}
}

// ForceGC forces garbage collection
func (mm *MemoryManager) ForceGC() {
	mm.mu.Lock()
	mm.lastGCTime = time.Now()
	mm.gcCount++
	mm.mu.Unlock()

	runtime.GC()
	debug.FreeOSMemory()
}

// GetStringSlice gets a string slice from the pool
func (mm *MemoryManager) GetStringSlice() []string {
	if mm.stringPool == nil {
		return make([]string, 0, 10)
	}

	atomic.AddInt64(&mm.stats.StringPoolHits, 1)
	slice := mm.stringPool.Get().([]string)
	return slice[:0] // Reset length but keep capacity
}

// PutStringSlice returns a string slice to the pool
func (mm *MemoryManager) PutStringSlice(slice []string) {
	if mm.stringPool == nil || slice == nil {
		return
	}

	// Clear the slice
	for i := range slice {
		slice[i] = ""
	}
	slice = slice[:0]

	mm.stringPool.Put(slice)
}

// GetByteSlice gets a byte slice from the pool
func (mm *MemoryManager) GetByteSlice() []byte {
	if mm.byteSlicePool == nil {
		return make([]byte, 0, 4096)
	}

	atomic.AddInt64(&mm.stats.BytePoolHits, 1)
	slice := mm.byteSlicePool.Get().([]byte)
	return slice[:0] // Reset length but keep capacity
}

// PutByteSlice returns a byte slice to the pool
func (mm *MemoryManager) PutByteSlice(slice []byte) {
	if mm.byteSlicePool == nil || slice == nil {
		return
	}

	// Clear the slice
	for i := range slice {
		slice[i] = 0
	}
	slice = slice[:0]

	mm.byteSlicePool.Put(slice)
}

// GetMap gets a map from the pool
func (mm *MemoryManager) GetMap() map[string]interface{} {
	if mm.mapPool == nil {
		return make(map[string]interface{})
	}

	atomic.AddInt64(&mm.stats.MapPoolHits, 1)
	m := mm.mapPool.Get().(map[string]interface{})

	// Clear the map
	for k := range m {
		delete(m, k)
	}

	return m
}

// PutMap returns a map to the pool
func (mm *MemoryManager) PutMap(m map[string]interface{}) {
	if mm.mapPool == nil || m == nil {
		return
	}

	// Clear the map
	for k := range m {
		delete(m, k)
	}

	mm.mapPool.Put(m)
}

// RegisterPressureCallback registers a callback for memory pressure events
func (mm *MemoryManager) RegisterPressureCallback(callback PressureCallback) {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	mm.pressureCallbacks = append(mm.pressureCallbacks, callback)
}

// GetStats returns current memory statistics
func (mm *MemoryManager) GetStats() MemoryStats {
	mm.mu.RLock()
	defer mm.mu.RUnlock()

	stats := *mm.stats
	stats.StringPoolHits = atomic.LoadInt64(&mm.stats.StringPoolHits)
	stats.StringPoolMisses = atomic.LoadInt64(&mm.stats.StringPoolMisses)
	stats.BytePoolHits = atomic.LoadInt64(&mm.stats.BytePoolHits)
	stats.BytePoolMisses = atomic.LoadInt64(&mm.stats.BytePoolMisses)
	stats.MapPoolHits = atomic.LoadInt64(&mm.stats.MapPoolHits)
	stats.MapPoolMisses = atomic.LoadInt64(&mm.stats.MapPoolMisses)

	return stats
}

// SetGCPercent sets the garbage collection target percentage
func (mm *MemoryManager) SetGCPercent(percent int) int {
	return debug.SetGCPercent(percent)
}

// SetMemoryLimit sets a soft memory limit for the runtime
func (mm *MemoryManager) SetMemoryLimit(limitMB int64) int64 {
	if limitMB <= 0 {
		return 0
	}

	limitBytes := limitMB * 1024 * 1024
	return debug.SetMemoryLimit(limitBytes) / 1024 / 1024
}

// TriggerGCIfNeeded triggers GC if memory usage exceeds threshold
func (mm *MemoryManager) TriggerGCIfNeeded() bool {
	mm.mu.RLock()
	allocMB := mm.stats.AllocMB
	threshold := mm.gcThresholdMB
	mm.mu.RUnlock()

	if allocMB > threshold {
		mm.ForceGC()
		return true
	}

	return false
}

// GetMemoryPressure returns current memory pressure level
func (mm *MemoryManager) GetMemoryPressure() float64 {
	mm.mu.RLock()
	defer mm.mu.RUnlock()

	return mm.stats.PressureLevel
}

// IsUnderPressure returns true if memory is under pressure
func (mm *MemoryManager) IsUnderPressure() bool {
	return mm.GetMemoryPressure() > mm.pressureThreshold
}

// OptimizeGC optimizes garbage collection settings based on current usage
func (mm *MemoryManager) OptimizeGC() {
	stats := mm.GetStats()

	// Adjust GC percentage based on memory pressure
	if stats.PressureLevel > 0.9 {
		// High pressure: more aggressive GC
		debug.SetGCPercent(50)
	} else if stats.PressureLevel > 0.7 {
		// Medium pressure: moderate GC
		debug.SetGCPercent(75)
	} else {
		// Low pressure: less frequent GC
		debug.SetGCPercent(100)
	}
}

// Close shuts down the memory manager
func (mm *MemoryManager) Close() {
	mm.StopMonitoring()

	// Clear pools
	if mm.stringPool != nil {
		mm.stringPool = nil
	}
	if mm.byteSlicePool != nil {
		mm.byteSlicePool = nil
	}
	if mm.mapPool != nil {
		mm.mapPool = nil
	}
}
