package terminal

import (
	"runtime"
	"testing"
)

func TestGetState(t *testing.T) {
	tests := []struct {
		name string
		fd   int
	}{
		{
			name: "stdin fd",
			fd:   0,
		},
		{
			name: "stdout fd",
			fd:   1,
		},
		{
			name: "stderr fd",
			fd:   2,
		},
		{
			name: "invalid fd",
			fd:   -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state, err := GetState(tt.fd)
			if err != nil {
				t.Errorf("GetState() error = %v", err)
				return
			}
			if state == nil {
				t.Error("GetState() returned nil state")
			}
		})
	}
}

func TestTerminalStateRestore(t *testing.T) {
	state := &TerminalState{}
	err := state.Restore()
	if err != nil {
		t.Errorf("Restore() error = %v", err)
	}
}

func TestMakeRaw(t *testing.T) {
	tests := []struct {
		name string
		fd   int
	}{
		{
			name: "stdin fd",
			fd:   0,
		},
		{
			name: "stdout fd",
			fd:   1,
		},
		{
			name: "stderr fd",
			fd:   2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state, err := MakeRaw(tt.fd)
			if err != nil {
				t.Errorf("MakeRaw() error = %v", err)
				return
			}
			if state == nil {
				t.Error("MakeRaw() returned nil state")
			}
		})
	}
}

func TestSetupTerminal(t *testing.T) {
	state, err := SetupTerminal()

	if runtime.GOOS == "windows" {
		// On Windows, should succeed
		if err != nil {
			t.Errorf("SetupTerminal() error = %v, expected nil on Windows", err)
		}
		if state == nil {
			t.Error("SetupTerminal() returned nil state on Windows")
		}
	} else {
		// On Unix systems, should also succeed
		if err != nil {
			t.Errorf("SetupTerminal() error = %v, expected nil on Unix", err)
		}
		if state == nil {
			t.Error("SetupTerminal() returned nil state on Unix")
		}
	}
}

func TestTerminalStateLifecycle(t *testing.T) {
	// Test the complete lifecycle: GetState -> MakeRaw -> Restore
	originalState, err := GetState(0)
	if err != nil {
		t.Fatalf("GetState() error = %v", err)
	}

	rawState, err := MakeRaw(0)
	if err != nil {
		t.Fatalf("MakeRaw() error = %v", err)
	}

	// Restore the raw state
	err = rawState.Restore()
	if err != nil {
		t.Errorf("rawState.Restore() error = %v", err)
	}

	// Restore the original state
	err = originalState.Restore()
	if err != nil {
		t.Errorf("originalState.Restore() error = %v", err)
	}
}

func TestMultipleGetState(t *testing.T) {
	// Test that multiple calls to GetState work
	state1, err1 := GetState(0)
	state2, err2 := GetState(0)

	if err1 != nil {
		t.Errorf("First GetState() error = %v", err1)
	}
	if err2 != nil {
		t.Errorf("Second GetState() error = %v", err2)
	}

	if state1 == nil {
		t.Error("First GetState() returned nil")
	}
	if state2 == nil {
		t.Error("Second GetState() returned nil")
	}
}

func TestMultipleMakeRaw(t *testing.T) {
	// Test that multiple calls to MakeRaw work
	state1, err1 := MakeRaw(0)
	state2, err2 := MakeRaw(0)

	if err1 != nil {
		t.Errorf("First MakeRaw() error = %v", err1)
	}
	if err2 != nil {
		t.Errorf("Second MakeRaw() error = %v", err2)
	}

	if state1 == nil {
		t.Error("First MakeRaw() returned nil")
	}
	if state2 == nil {
		t.Error("Second MakeRaw() returned nil")
	}

	// Restore both states
	if err := state1.Restore(); err != nil {
		t.Errorf("state1.Restore() error = %v", err)
	}
	if err := state2.Restore(); err != nil {
		t.Errorf("state2.Restore() error = %v", err)
	}
}

func TestSetupTerminalMultipleCalls(t *testing.T) {
	// Test that multiple calls to SetupTerminal work
	state1, err1 := SetupTerminal()
	state2, err2 := SetupTerminal()

	if err1 != nil {
		t.Errorf("First SetupTerminal() error = %v", err1)
	}
	if err2 != nil {
		t.Errorf("Second SetupTerminal() error = %v", err2)
	}

	if state1 == nil {
		t.Error("First SetupTerminal() returned nil")
	}
	if state2 == nil {
		t.Error("Second SetupTerminal() returned nil")
	}
}

// Benchmark tests
func BenchmarkGetState(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = GetState(0)
	}
}

func BenchmarkMakeRaw(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = MakeRaw(0)
	}
}

func BenchmarkRestore(b *testing.B) {
	state := &TerminalState{}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = state.Restore()
	}
}

func BenchmarkSetupTerminal(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = SetupTerminal()
	}
}

// Test edge cases
func TestGetStateWithInvalidFd(t *testing.T) {
	invalidFds := []int{-1, -100, 999999}

	for _, fd := range invalidFds {
		t.Run("invalid_fd", func(t *testing.T) {
			state, err := GetState(fd)
			// The current implementation doesn't validate fd,
			// so it should still return a state without error
			if err != nil {
				t.Logf("GetState(%d) error = %v (this may be expected)", fd, err)
			}
			if state == nil && err == nil {
				t.Errorf("GetState(%d) returned nil state without error", fd)
			}
		})
	}
}

func TestMakeRawWithInvalidFd(t *testing.T) {
	invalidFds := []int{-1, -100, 999999}

	for _, fd := range invalidFds {
		t.Run("invalid_fd", func(t *testing.T) {
			state, err := MakeRaw(fd)
			// The current implementation doesn't validate fd,
			// so it should still return a state without error
			if err != nil {
				t.Logf("MakeRaw(%d) error = %v (this may be expected)", fd, err)
			}
			if state == nil && err == nil {
				t.Errorf("MakeRaw(%d) returned nil state without error", fd)
			}
		})
	}
}
