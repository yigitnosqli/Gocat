package worker

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestNewWorkerPool(t *testing.T) {
	config := DefaultPoolConfig()
	pool := NewWorkerPool(config)
	defer pool.ForceShutdown()

	if pool.minWorkers != config.MinWorkers {
		t.Errorf("Expected minWorkers %d, got %d", config.MinWorkers, pool.minWorkers)
	}

	if pool.maxWorkers != config.MaxWorkers {
		t.Errorf("Expected maxWorkers %d, got %d", config.MaxWorkers, pool.maxWorkers)
	}

	// Should have minimum workers running
	time.Sleep(100 * time.Millisecond) // Allow workers to start
	if pool.WorkerCount() != config.MinWorkers {
		t.Errorf("Expected %d workers, got %d", config.MinWorkers, pool.WorkerCount())
	}
}

func TestWorkerPoolSubmitTask(t *testing.T) {
	pool := NewWorkerPool(DefaultPoolConfig())
	defer pool.ForceShutdown()

	var executed int64
	task := &TaskFunc{
		ID:       "test-task",
		Priority: 0,
		Timeout:  time.Second,
		Fn: func(ctx context.Context) error {
			atomic.AddInt64(&executed, 1)
			return nil
		},
	}

	err := pool.Submit(task)
	if err != nil {
		t.Fatalf("Failed to submit task: %v", err)
	}

	// Wait for task to complete
	time.Sleep(200 * time.Millisecond)

	if atomic.LoadInt64(&executed) != 1 {
		t.Errorf("Expected task to be executed once, got %d", executed)
	}

	stats := pool.GetStats()
	if stats.CompletedTasks != 1 {
		t.Errorf("Expected 1 completed task, got %d", stats.CompletedTasks)
	}
}

func TestWorkerPoolSubmitFunc(t *testing.T) {
	pool := NewWorkerPool(DefaultPoolConfig())
	defer pool.ForceShutdown()

	var executed int64
	err := pool.SubmitFunc("test-func", func(ctx context.Context) error {
		atomic.AddInt64(&executed, 1)
		return nil
	})

	if err != nil {
		t.Fatalf("Failed to submit function: %v", err)
	}

	// Wait for task to complete
	time.Sleep(200 * time.Millisecond)

	if atomic.LoadInt64(&executed) != 1 {
		t.Errorf("Expected function to be executed once, got %d", executed)
	}
}

func TestWorkerPoolConcurrentTasks(t *testing.T) {
	config := DefaultPoolConfig()
	config.MinWorkers = 4
	config.MaxWorkers = 8
	pool := NewWorkerPool(config)
	defer pool.ForceShutdown()

	const numTasks = 20
	var completed int64
	var wg sync.WaitGroup

	for i := 0; i < numTasks; i++ {
		wg.Add(1)
		err := pool.SubmitFunc("concurrent-task", func(ctx context.Context) error {
			defer wg.Done()
			time.Sleep(50 * time.Millisecond) // Simulate work
			atomic.AddInt64(&completed, 1)
			return nil
		})

		if err != nil {
			t.Fatalf("Failed to submit task %d: %v", i, err)
		}
	}

	wg.Wait()

	if atomic.LoadInt64(&completed) != numTasks {
		t.Errorf("Expected %d completed tasks, got %d", numTasks, completed)
	}

	stats := pool.GetStats()
	if stats.CompletedTasks != numTasks {
		t.Errorf("Expected %d completed tasks in stats, got %d", numTasks, stats.CompletedTasks)
	}
}

func TestWorkerPoolTaskTimeout(t *testing.T) {
	pool := NewWorkerPool(DefaultPoolConfig())
	defer pool.ForceShutdown()

	task := &TaskFunc{
		ID:       "timeout-task",
		Priority: 0,
		Timeout:  100 * time.Millisecond,
		Fn: func(ctx context.Context) error {
			select {
			case <-time.After(200 * time.Millisecond):
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		},
	}

	err := pool.Submit(task)
	if err != nil {
		t.Fatalf("Failed to submit task: %v", err)
	}

	// Wait for task to timeout
	time.Sleep(300 * time.Millisecond)

	stats := pool.GetStats()
	if stats.FailedTasks != 1 {
		t.Errorf("Expected 1 failed task, got %d", stats.FailedTasks)
	}
}

func TestWorkerPoolTaskError(t *testing.T) {
	pool := NewWorkerPool(DefaultPoolConfig())
	defer pool.ForceShutdown()

	expectedError := errors.New("task error")
	task := &TaskFunc{
		ID:       "error-task",
		Priority: 0,
		Timeout:  time.Second,
		Fn: func(ctx context.Context) error {
			return expectedError
		},
	}

	err := pool.Submit(task)
	if err != nil {
		t.Fatalf("Failed to submit task: %v", err)
	}

	// Wait for task to complete
	time.Sleep(200 * time.Millisecond)

	stats := pool.GetStats()
	if stats.FailedTasks != 1 {
		t.Errorf("Expected 1 failed task, got %d", stats.FailedTasks)
	}
}

func TestWorkerPoolScaling(t *testing.T) {
	config := DefaultPoolConfig()
	config.MinWorkers = 2
	config.MaxWorkers = 6
	config.ScaleInterval = 100 * time.Millisecond
	pool := NewWorkerPool(config)
	defer pool.ForceShutdown()

	// Submit many tasks to trigger scaling
	const numTasks = 20
	for i := 0; i < numTasks; i++ {
		pool.SubmitFunc("scale-task", func(ctx context.Context) error {
			time.Sleep(200 * time.Millisecond) // Simulate work
			return nil
		})
	}

	// Wait for scaling to occur
	time.Sleep(300 * time.Millisecond)

	// Should have scaled up
	if pool.WorkerCount() <= config.MinWorkers {
		t.Errorf("Expected worker count > %d, got %d", config.MinWorkers, pool.WorkerCount())
	}

	// Wait for tasks to complete and scaling down
	time.Sleep(2 * time.Second)

	stats := pool.GetStats()
	if stats.PeakWorkers <= int64(config.MinWorkers) {
		t.Errorf("Expected peak workers > %d, got %d", config.MinWorkers, stats.PeakWorkers)
	}
}

func TestWorkerPoolShutdown(t *testing.T) {
	pool := NewWorkerPool(DefaultPoolConfig())

	// Submit a task
	var executed int64
	pool.SubmitFunc("shutdown-task", func(ctx context.Context) error {
		atomic.AddInt64(&executed, 1)
		return nil
	})

	// Shutdown with timeout
	err := pool.Shutdown(time.Second)
	if err != nil {
		t.Errorf("Shutdown failed: %v", err)
	}

	// Task should have completed
	if atomic.LoadInt64(&executed) != 1 {
		t.Errorf("Expected task to complete before shutdown, got %d", executed)
	}

	// Should not accept new tasks
	err = pool.SubmitFunc("post-shutdown", func(ctx context.Context) error {
		return nil
	})
	if err == nil {
		t.Error("Expected error when submitting task after shutdown")
	}
}

func TestWorkerPoolForceShutdown(t *testing.T) {
	pool := NewWorkerPool(DefaultPoolConfig())

	// Submit a long-running task
	pool.SubmitFunc("long-task", func(ctx context.Context) error {
		time.Sleep(time.Second)
		return nil
	})

	// Force shutdown immediately
	pool.ForceShutdown()

	// Should not accept new tasks
	err := pool.SubmitFunc("post-shutdown", func(ctx context.Context) error {
		return nil
	})
	if err == nil {
		t.Error("Expected error when submitting task after force shutdown")
	}
}

func TestWorkerPoolStats(t *testing.T) {
	pool := NewWorkerPool(DefaultPoolConfig())
	defer pool.ForceShutdown()

	// Submit some tasks
	const numTasks = 5
	for i := 0; i < numTasks; i++ {
		pool.SubmitFunc("stats-task", func(ctx context.Context) error {
			time.Sleep(10 * time.Millisecond)
			return nil
		})
	}

	// Wait for tasks to complete
	time.Sleep(200 * time.Millisecond)

	stats := pool.GetStats()

	if stats.TotalTasks != numTasks {
		t.Errorf("Expected %d total tasks, got %d", numTasks, stats.TotalTasks)
	}

	if stats.CompletedTasks != numTasks {
		t.Errorf("Expected %d completed tasks, got %d", numTasks, stats.CompletedTasks)
	}

	if stats.ActiveWorkers <= 0 {
		t.Errorf("Expected active workers > 0, got %d", stats.ActiveWorkers)
	}

	if stats.WorkerCreations <= 0 {
		t.Errorf("Expected worker creations > 0, got %d", stats.WorkerCreations)
	}
}

func TestTaskFuncInterface(t *testing.T) {
	task := &TaskFunc{
		ID:       "interface-test",
		Priority: 5,
		Timeout:  time.Minute,
		Fn: func(ctx context.Context) error {
			return nil
		},
	}

	if task.GetID() != "interface-test" {
		t.Errorf("Expected ID 'interface-test', got '%s'", task.GetID())
	}

	if task.GetPriority() != 5 {
		t.Errorf("Expected priority 5, got %d", task.GetPriority())
	}

	if task.GetTimeout() != time.Minute {
		t.Errorf("Expected timeout 1m, got %v", task.GetTimeout())
	}

	err := task.Execute(context.Background())
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

func BenchmarkWorkerPoolSubmit(b *testing.B) {
	pool := NewWorkerPool(DefaultPoolConfig())
	defer pool.ForceShutdown()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			pool.SubmitFunc("bench-task", func(ctx context.Context) error {
				return nil
			})
		}
	})
}

func BenchmarkWorkerPoolExecution(b *testing.B) {
	pool := NewWorkerPool(DefaultPoolConfig())
	defer pool.ForceShutdown()

	var wg sync.WaitGroup

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		wg.Add(1)
		pool.SubmitFunc("bench-exec", func(ctx context.Context) error {
			defer wg.Done()
			// Simulate minimal work
			time.Sleep(time.Microsecond)
			return nil
		})
	}

	wg.Wait()
}
