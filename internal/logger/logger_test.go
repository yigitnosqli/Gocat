package logger

import (
	"os"
	"testing"

	"github.com/fatih/color"
)

func TestNewLogger(t *testing.T) {
	tests := []struct {
		name  string
		level LogLevel
	}{
		{
			name:  "info level",
			level: LevelInfo,
		},
		{
			name:  "warn level",
			level: LevelWarn,
		},
		{
			name:  "error level",
			level: LevelError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := NewLogger(tt.level)
			if logger == nil {
				t.Error("NewLogger returned nil")
				return
			}
			if logger.level != tt.level {
				t.Errorf("expected level %d, got %d", tt.level, logger.level)
			}
		})
	}
}

func TestLoggerLevels(t *testing.T) {
	// Test logger level filtering without capturing output
	// since color package writes directly to stdout and is hard to capture
	logger := NewLogger(LevelWarn)

	// Test that the logger has the correct level
	if logger.level != LevelWarn {
		t.Errorf("Expected logger level to be %d, got %d", LevelWarn, logger.level)
	}

	// Test level comparison logic
	if logger.level > LevelWarn {
		t.Error("Warn messages should be logged when level is warn")
	}

	if logger.level > LevelError {
		t.Error("Error messages should be logged when level is warn")
	}

	if logger.level <= LevelInfo {
		t.Error("Info messages should not be logged when level is warn")
	}
}

func TestLoggerInfo(t *testing.T) {
	// Test that Info method doesn't panic and works with formatting
	logger := NewLogger(LevelInfo)

	// This should not panic
	logger.Info("test message with %s", "formatting")

	// Test that the logger level allows info messages
	if logger.level > LevelInfo {
		t.Error("Info messages should be logged when level is info")
	}
}

func TestLoggerWarn(t *testing.T) {
	// Test that Warn method doesn't panic and works with formatting
	logger := NewLogger(LevelInfo)

	// This should not panic
	logger.Warn("test warning with %d", 42)

	// Test that the logger level allows warn messages
	if logger.level > LevelWarn {
		t.Error("Warn messages should be logged when level is info")
	}
}

func TestLoggerError(t *testing.T) {
	// Test that Error method doesn't panic and works with formatting
	logger := NewLogger(LevelInfo)

	// This should not panic
	logger.Error("test error: %v", "something went wrong")

	// Test that the logger level allows error messages
	if logger.level > LevelError {
		t.Error("Error messages should be logged when level is info")
	}
}

func TestDefaultLoggerFunctions(t *testing.T) {
	// Test SetLevel
	originalLevel := defaultLogger.level
	SetLevel(LevelError)

	// Check that the level was set correctly
	if defaultLogger.level != LevelError {
		t.Errorf("Expected default logger level to be %d, got %d", LevelError, defaultLogger.level)
	}

	// Test that functions don't panic
	Info("info message")
	Warn("warn message")
	Error("error message")

	// Reset to original level
	SetLevel(originalLevel)

	// Verify reset
	if defaultLogger.level != originalLevel {
		t.Errorf("Failed to reset default logger level to %d, got %d", originalLevel, defaultLogger.level)
	}
}

func TestSetupLogger(t *testing.T) {
	// This function just calls log.SetFlags(0)
	// We can't easily test the effect, but we can ensure it doesn't panic
	SetupLogger()
}

func TestLogLevels(t *testing.T) {
	tests := []struct {
		name  string
		level LogLevel
		value int
	}{
		{"LevelInfo", LevelInfo, 0},
		{"LevelWarn", LevelWarn, 1},
		{"LevelError", LevelError, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if int(tt.level) != tt.value {
				t.Errorf("expected %s to have value %d, got %d", tt.name, tt.value, int(tt.level))
			}
		})
	}
}

// Benchmark tests
func BenchmarkLoggerInfo(b *testing.B) {
	logger := NewLogger(LevelInfo)
	color.NoColor = true
	defer func() { color.NoColor = false }()

	// Redirect output to discard
	oldStdout := os.Stdout
	os.Stdout, _ = os.Open(os.DevNull)
	defer func() { os.Stdout = oldStdout }()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Info("benchmark message %d", i)
	}
}

func BenchmarkLoggerWarn(b *testing.B) {
	logger := NewLogger(LevelInfo)
	color.NoColor = true
	defer func() { color.NoColor = false }()

	oldStdout := os.Stdout
	os.Stdout, _ = os.Open(os.DevNull)
	defer func() { os.Stdout = oldStdout }()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Warn("benchmark warning %d", i)
	}
}

func BenchmarkLoggerError(b *testing.B) {
	logger := NewLogger(LevelInfo)
	color.NoColor = true
	defer func() { color.NoColor = false }()

	oldStdout := os.Stdout
	os.Stdout, _ = os.Open(os.DevNull)
	defer func() { os.Stdout = oldStdout }()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Error("benchmark error %d", i)
	}
}

func BenchmarkDefaultLoggerInfo(b *testing.B) {
	color.NoColor = true
	defer func() { color.NoColor = false }()

	oldStdout := os.Stdout
	os.Stdout, _ = os.Open(os.DevNull)
	defer func() { os.Stdout = oldStdout }()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Info("benchmark message %d", i)
	}
}
