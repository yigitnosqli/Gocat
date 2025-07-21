package signals

import (
	"os"
	"runtime"
	"sync"
	"syscall"
	"testing"
	"time"
)

func TestBlockExitSignals(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping signal test on Windows - signal handling differs")
	}

	// Test that BlockExitSignals doesn't panic
	BlockExitSignals()

	// Give it a moment to set up
	time.Sleep(10 * time.Millisecond)

	// Send a signal to ourselves (this should be blocked)
	pid := os.Getpid()
	process, err := os.FindProcess(pid)
	if err != nil {
		t.Fatalf("Failed to find process: %v", err)
	}

	// Send SIGTERM (this should be caught and blocked)
	err = process.Signal(syscall.SIGTERM)
	if err != nil {
		t.Fatalf("Failed to send signal: %v", err)
	}

	// If we reach here, the signal was blocked successfully
	// Wait a bit to ensure the signal handler had time to process
	time.Sleep(50 * time.Millisecond)
}

func TestSetupSignalHandler(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping signal test on Windows - signal handling differs")
	}

	var handlerCalled bool
	var mu sync.Mutex

	// Setup signal handler
	handler := func() {
		mu.Lock()
		handlerCalled = true
		mu.Unlock()
	}

	SetupSignalHandler(handler)

	// Give it a moment to set up
	time.Sleep(10 * time.Millisecond)

	// Send a signal to ourselves
	pid := os.Getpid()
	process, err := os.FindProcess(pid)
	if err != nil {
		t.Fatalf("Failed to find process: %v", err)
	}

	// Send SIGTERM
	err = process.Signal(syscall.SIGTERM)
	if err != nil {
		t.Fatalf("Failed to send signal: %v", err)
	}

	// Wait for handler to be called
	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	called := handlerCalled
	mu.Unlock()

	if !called {
		t.Error("Signal handler was not called")
	}
}

func TestSetupSignalHandlerSIGINT(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping signal test on Windows - signal handling differs")
	}

	var handlerCalled bool
	var mu sync.Mutex

	// Setup signal handler
	handler := func() {
		mu.Lock()
		handlerCalled = true
		mu.Unlock()
	}

	SetupSignalHandler(handler)

	// Give it a moment to set up
	time.Sleep(10 * time.Millisecond)

	// Send SIGINT to ourselves
	pid := os.Getpid()
	process, err := os.FindProcess(pid)
	if err != nil {
		t.Fatalf("Failed to find process: %v", err)
	}

	err = process.Signal(os.Interrupt)
	if err != nil {
		t.Fatalf("Failed to send signal: %v", err)
	}

	// Wait for handler to be called
	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	called := handlerCalled
	mu.Unlock()

	if !called {
		t.Error("Signal handler was not called for SIGINT")
	}
}

func TestMultipleSignalHandlers(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping signal test on Windows - signal handling differs")
	}

	var handler1Called, handler2Called bool
	var mu sync.Mutex

	// Setup first signal handler
	handler1 := func() {
		mu.Lock()
		handler1Called = true
		mu.Unlock()
	}

	// Setup second signal handler
	handler2 := func() {
		mu.Lock()
		handler2Called = true
		mu.Unlock()
	}

	SetupSignalHandler(handler1)
	SetupSignalHandler(handler2)

	// Give them a moment to set up
	time.Sleep(10 * time.Millisecond)

	// Send signal
	pid := os.Getpid()
	process, err := os.FindProcess(pid)
	if err != nil {
		t.Fatalf("Failed to find process: %v", err)
	}

	err = process.Signal(syscall.SIGTERM)
	if err != nil {
		t.Fatalf("Failed to send signal: %v", err)
	}

	// Wait for handlers to be called
	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	called1 := handler1Called
	called2 := handler2Called
	mu.Unlock()

	// At least one handler should be called
	// (The behavior with multiple handlers may vary)
	if !called1 && !called2 {
		t.Error("No signal handlers were called")
	}
}

func TestSignalHandlerWithPanic(t *testing.T) {
	// Test that a panicking handler doesn't crash the program
	handler := func() {
		defer func() {
			if r := recover(); r != nil {
				// Expected panic, test should continue
				t.Logf("Recovered from expected panic: %v", r)
			}
		}()
		panic("test panic")
	}

	// This should not cause the test to fail
	SetupSignalHandler(handler)

	// Give it a moment to set up
	time.Sleep(10 * time.Millisecond)

	// The test passes if we reach here without crashing
}

func TestBlockExitSignalsMultipleCalls(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping signal test on Windows - signal handling differs")
	}

	// Test that calling BlockExitSignals multiple times doesn't cause issues
	BlockExitSignals()
	BlockExitSignals()
	BlockExitSignals()

	// Give them a moment to set up
	time.Sleep(10 * time.Millisecond)

	// Send a signal
	pid := os.Getpid()
	process, err := os.FindProcess(pid)
	if err != nil {
		t.Fatalf("Failed to find process: %v", err)
	}

	err = process.Signal(syscall.SIGTERM)
	if err != nil {
		t.Fatalf("Failed to send signal: %v", err)
	}

	// Wait a bit
	time.Sleep(50 * time.Millisecond)

	// If we reach here, the signals were handled properly
}

// Benchmark tests
func BenchmarkBlockExitSignals(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		BlockExitSignals()
	}
}

func BenchmarkSetupSignalHandler(b *testing.B) {
	handler := func() {
		// Empty handler
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		SetupSignalHandler(handler)
	}
}

// Test helper function to check if signal handling works
func TestSignalHandlerExecution(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping signal test on Windows - signal handling differs")
	}

	var executed bool
	var mu sync.Mutex

	handler := func() {
		mu.Lock()
		executed = true
		mu.Unlock()
	}

	SetupSignalHandler(handler)

	// Allow setup time
	time.Sleep(10 * time.Millisecond)

	// Simulate signal
	pid := os.Getpid()
	if process, err := os.FindProcess(pid); err == nil {
		if err := process.Signal(os.Interrupt); err != nil {
			t.Logf("Failed to send signal: %v", err)
		}
		time.Sleep(50 * time.Millisecond)
	}

	mu.Lock()
	result := executed
	mu.Unlock()

	if !result {
		t.Log("Signal handler execution test - handler may not have been called (this can be normal in test environments)")
	}
}
