package worker

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

// WorkerPool manages a pool of goroutines for handling tasks
type WorkerPool struct {
	// Configuration
	minWorkers  int           // Minimum number of workers
	maxWorkers  int           // Maximum number of workers
	idleTimeout time.Duration // Time before idle workers are terminated
	taskTimeout time.Duration // Maximum time for task execution

	// State
	workers       map[int]*Worker // Active workers
	workerCounter int64           // Counter for worker IDs
	taskQueue     chan Task       // Queue for incoming tasks
	ctx           context.Context // Pool context
	cancel        context.CancelFunc

	// Synchronization
	mu sync.RWMutex   // Protects workers map
	wg sync.WaitGroup // Waits for all workers to finish

	// Statistics
	stats *PoolStats // Pool statistics

	// Scaling
	scaleTimer *time.Timer // Timer for scaling decisions
	scaleMu    sync.Mutex  // Protects scaling operations
}

// Worker represents a single worker goroutine
type Worker struct {
	id           int
	pool         *WorkerPool
	taskChan     chan Task
	ctx          context.Context
	cancel       context.CancelFunc
	lastActive   time.Time
	tasksHandled int64
	isIdle       bool
	mu           sync.RWMutex
}

// Task represents work to be executed by a worker
type Task interface {
	Execute(ctx context.Context) error
	GetID() string
	GetPriority() int
	GetTimeout() time.Duration
}

// TaskFunc is a function that implements the Task interface
type TaskFunc struct {
	ID       string
	Priority int
	Timeout  time.Duration
	Fn       func(ctx context.Context) error
}

// Execute implements the Task interface
func (tf *TaskFunc) Execute(ctx context.Context) error {
	return tf.Fn(ctx)
}

// GetID implements the Task interface
func (tf *TaskFunc) GetID() string {
	return tf.ID
}

// GetPriority implements the Task interface
func (tf *TaskFunc) GetPriority() int {
	return tf.Priority
}

// GetTimeout implements the Task interface
func (tf *TaskFunc) GetTimeout() time.Duration {
	return tf.Timeout
}

// PoolStats tracks worker pool statistics
type PoolStats struct {
	ActiveWorkers      int64         // Currently active workers
	IdleWorkers        int64         // Currently idle workers
	TotalTasks         int64         // Total tasks processed
	CompletedTasks     int64         // Successfully completed tasks
	FailedTasks        int64         // Failed tasks
	QueuedTasks        int64         // Currently queued tasks
	AverageTaskTime    time.Duration // Average task execution time
	TotalTaskTime      int64         // Total time spent on tasks (nanoseconds)
	WorkerCreations    int64         // Total workers created
	WorkerTerminations int64         // Total workers terminated
	LastScaleEvent     time.Time     // Last scaling event
	PeakWorkers        int64         // Peak number of workers
}

// PoolConfig contains configuration for the worker pool
type PoolConfig struct {
	MinWorkers    int
	MaxWorkers    int
	QueueSize     int
	IdleTimeout   time.Duration
	TaskTimeout   time.Duration
	ScaleInterval time.Duration
}

// DefaultPoolConfig returns a default configuration
func DefaultPoolConfig() *PoolConfig {
	return &PoolConfig{
		MinWorkers:    2,
		MaxWorkers:    runtime.NumCPU() * 2,
		QueueSize:     1000,
		IdleTimeout:   30 * time.Second,
		TaskTimeout:   5 * time.Minute,
		ScaleInterval: 10 * time.Second,
	}
}

// NewWorkerPool creates a new worker pool with the given configuration
func NewWorkerPool(config *PoolConfig) *WorkerPool {
	if config == nil {
		config = DefaultPoolConfig()
	}

	ctx, cancel := context.WithCancel(context.Background())

	pool := &WorkerPool{
		minWorkers:  config.MinWorkers,
		maxWorkers:  config.MaxWorkers,
		idleTimeout: config.IdleTimeout,
		taskTimeout: config.TaskTimeout,
		workers:     make(map[int]*Worker),
		taskQueue:   make(chan Task, config.QueueSize),
		ctx:         ctx,
		cancel:      cancel,
		stats: &PoolStats{
			LastScaleEvent: time.Now(),
		},
	}

	// Start initial workers
	for i := 0; i < config.MinWorkers; i++ {
		pool.createWorker()
	}

	// Start scaling monitor
	pool.startScalingMonitor(config.ScaleInterval)

	return pool
}

// Submit submits a task to the worker pool
func (wp *WorkerPool) Submit(task Task) error {
	select {
	case wp.taskQueue <- task:
		atomic.AddInt64(&wp.stats.TotalTasks, 1)
		atomic.AddInt64(&wp.stats.QueuedTasks, 1)
		return nil
	case <-wp.ctx.Done():
		return fmt.Errorf("worker pool is shutting down")
	default:
		return fmt.Errorf("task queue is full")
	}
}

// SubmitFunc submits a function as a task
func (wp *WorkerPool) SubmitFunc(id string, fn func(ctx context.Context) error) error {
	task := &TaskFunc{
		ID:       id,
		Priority: 0,
		Timeout:  wp.taskTimeout,
		Fn:       fn,
	}
	return wp.Submit(task)
}

// createWorker creates a new worker and starts it
func (wp *WorkerPool) createWorker() *Worker {
	wp.mu.Lock()
	defer wp.mu.Unlock()

	workerID := int(atomic.AddInt64(&wp.workerCounter, 1))
	ctx, cancel := context.WithCancel(wp.ctx)

	worker := &Worker{
		id:         workerID,
		pool:       wp,
		taskChan:   make(chan Task, 1),
		ctx:        ctx,
		cancel:     cancel,
		lastActive: time.Now(),
		isIdle:     true,
	}

	wp.workers[workerID] = worker
	atomic.AddInt64(&wp.stats.WorkerCreations, 1)
	atomic.AddInt64(&wp.stats.ActiveWorkers, 1)
	atomic.AddInt64(&wp.stats.IdleWorkers, 1)

	// Update peak workers
	current := atomic.LoadInt64(&wp.stats.ActiveWorkers)
	for {
		peak := atomic.LoadInt64(&wp.stats.PeakWorkers)
		if current <= peak || atomic.CompareAndSwapInt64(&wp.stats.PeakWorkers, peak, current) {
			break
		}
	}

	wp.wg.Add(1)
	go worker.run()

	return worker
}

// terminateWorker terminates a worker
func (wp *WorkerPool) terminateWorker(workerID int) {
	wp.mu.Lock()
	defer wp.mu.Unlock()

	if worker, exists := wp.workers[workerID]; exists {
		worker.cancel()
		delete(wp.workers, workerID)
		atomic.AddInt64(&wp.stats.WorkerTerminations, 1)
		atomic.AddInt64(&wp.stats.ActiveWorkers, -1)

		worker.mu.RLock()
		if worker.isIdle {
			atomic.AddInt64(&wp.stats.IdleWorkers, -1)
		}
		worker.mu.RUnlock()
	}
}

// run is the main worker loop
func (w *Worker) run() {
	defer w.pool.wg.Done()

	idleTimer := time.NewTimer(w.pool.idleTimeout)
	defer idleTimer.Stop()

	for {
		select {
		case task := <-w.pool.taskQueue:
			w.handleTask(task)
			idleTimer.Reset(w.pool.idleTimeout)

		case <-idleTimer.C:
			// Worker has been idle too long, check if we can terminate
			if w.pool.canTerminateWorker() {
				w.pool.terminateWorker(w.id)
				return
			}
			idleTimer.Reset(w.pool.idleTimeout)

		case <-w.ctx.Done():
			return
		}
	}
}

// handleTask processes a single task
func (w *Worker) handleTask(task Task) {
	w.mu.Lock()
	w.isIdle = false
	w.lastActive = time.Now()
	w.mu.Unlock()

	atomic.AddInt64(&w.pool.stats.IdleWorkers, -1)
	atomic.AddInt64(&w.pool.stats.QueuedTasks, -1)

	defer func() {
		w.mu.Lock()
		w.isIdle = true
		w.mu.Unlock()
		atomic.AddInt64(&w.pool.stats.IdleWorkers, 1)
		atomic.AddInt64(&w.tasksHandled, 1)
	}()

	// Create task context with timeout
	taskCtx := w.ctx
	if timeout := task.GetTimeout(); timeout > 0 {
		var cancel context.CancelFunc
		taskCtx, cancel = context.WithTimeout(w.ctx, timeout)
		defer cancel()
	}

	startTime := time.Now()
	err := task.Execute(taskCtx)
	duration := time.Since(startTime)

	// Update statistics
	atomic.AddInt64(&w.pool.stats.TotalTaskTime, int64(duration))
	if err != nil {
		atomic.AddInt64(&w.pool.stats.FailedTasks, 1)
	} else {
		atomic.AddInt64(&w.pool.stats.CompletedTasks, 1)
	}
}

// canTerminateWorker checks if a worker can be terminated
func (wp *WorkerPool) canTerminateWorker() bool {
	wp.mu.RLock()
	defer wp.mu.RUnlock()

	activeWorkers := len(wp.workers)
	return activeWorkers > wp.minWorkers
}

// startScalingMonitor starts the automatic scaling monitor
func (wp *WorkerPool) startScalingMonitor(interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				wp.scaleWorkers()
			case <-wp.ctx.Done():
				return
			}
		}
	}()
}

// scaleWorkers automatically scales the worker pool based on load
func (wp *WorkerPool) scaleWorkers() {
	wp.scaleMu.Lock()
	defer wp.scaleMu.Unlock()

	queuedTasks := atomic.LoadInt64(&wp.stats.QueuedTasks)
	activeWorkers := atomic.LoadInt64(&wp.stats.ActiveWorkers)
	idleWorkers := atomic.LoadInt64(&wp.stats.IdleWorkers)

	// Scale up if queue is building up and we have capacity
	if queuedTasks > activeWorkers && int(activeWorkers) < wp.maxWorkers {
		needed := queuedTasks - activeWorkers
		maxToCreate := int64(wp.maxWorkers) - activeWorkers

		if needed > maxToCreate {
			needed = maxToCreate
		}

		for i := int64(0); i < needed; i++ {
			wp.createWorker()
		}

		wp.stats.LastScaleEvent = time.Now()
	}

	// Scale down if we have too many idle workers
	if idleWorkers > 2 && int(activeWorkers) > wp.minWorkers {
		excessIdle := idleWorkers - 2
		maxToRemove := activeWorkers - int64(wp.minWorkers)

		if excessIdle > maxToRemove {
			excessIdle = maxToRemove
		}

		// Find idle workers to terminate
		wp.mu.RLock()
		toTerminate := make([]int, 0, excessIdle)
		for id, worker := range wp.workers {
			if len(toTerminate) >= int(excessIdle) {
				break
			}

			worker.mu.RLock()
			if worker.isIdle && time.Since(worker.lastActive) > wp.idleTimeout {
				toTerminate = append(toTerminate, id)
			}
			worker.mu.RUnlock()
		}
		wp.mu.RUnlock()

		// Terminate selected workers
		for _, id := range toTerminate {
			wp.terminateWorker(id)
		}

		if len(toTerminate) > 0 {
			wp.stats.LastScaleEvent = time.Now()
		}
	}
}

// GetStats returns current pool statistics
func (wp *WorkerPool) GetStats() PoolStats {
	totalTasks := atomic.LoadInt64(&wp.stats.CompletedTasks) + atomic.LoadInt64(&wp.stats.FailedTasks)
	totalTime := time.Duration(atomic.LoadInt64(&wp.stats.TotalTaskTime))

	var avgTime time.Duration
	if totalTasks > 0 {
		avgTime = totalTime / time.Duration(totalTasks)
	}

	return PoolStats{
		ActiveWorkers:      atomic.LoadInt64(&wp.stats.ActiveWorkers),
		IdleWorkers:        atomic.LoadInt64(&wp.stats.IdleWorkers),
		TotalTasks:         atomic.LoadInt64(&wp.stats.TotalTasks),
		CompletedTasks:     atomic.LoadInt64(&wp.stats.CompletedTasks),
		FailedTasks:        atomic.LoadInt64(&wp.stats.FailedTasks),
		QueuedTasks:        atomic.LoadInt64(&wp.stats.QueuedTasks),
		AverageTaskTime:    avgTime,
		TotalTaskTime:      atomic.LoadInt64(&wp.stats.TotalTaskTime),
		WorkerCreations:    atomic.LoadInt64(&wp.stats.WorkerCreations),
		WorkerTerminations: atomic.LoadInt64(&wp.stats.WorkerTerminations),
		LastScaleEvent:     wp.stats.LastScaleEvent,
		PeakWorkers:        atomic.LoadInt64(&wp.stats.PeakWorkers),
	}
}

// Shutdown gracefully shuts down the worker pool
func (wp *WorkerPool) Shutdown(timeout time.Duration) error {
	// Stop accepting new tasks
	wp.cancel()

	// Wait for all workers to finish with timeout
	done := make(chan struct{})
	go func() {
		wp.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-time.After(timeout):
		return fmt.Errorf("shutdown timeout exceeded")
	}
}

// ForceShutdown forcefully shuts down the worker pool
func (wp *WorkerPool) ForceShutdown() {
	wp.cancel()

	// Terminate all workers
	wp.mu.Lock()
	for id := range wp.workers {
		wp.terminateWorker(id)
	}
	wp.mu.Unlock()
}

// QueueSize returns the current queue size
func (wp *WorkerPool) QueueSize() int {
	return len(wp.taskQueue)
}

// WorkerCount returns the current number of workers
func (wp *WorkerPool) WorkerCount() int {
	wp.mu.RLock()
	defer wp.mu.RUnlock()
	return len(wp.workers)
}
