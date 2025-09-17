package logging

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestLogLevel(t *testing.T) {
	testCases := []struct {
		level    LogLevel
		expected string
	}{
		{LevelDebug, "DEBUG"},
		{LevelInfo, "INFO"},
		{LevelWarn, "WARN"},
		{LevelError, "ERROR"},
		{LevelFatal, "FATAL"},
	}

	for _, tc := range testCases {
		if tc.level.String() != tc.expected {
			t.Errorf("Expected %s, got %s", tc.expected, tc.level.String())
		}
	}
}

func TestParseLogLevel(t *testing.T) {
	testCases := []struct {
		input    string
		expected LogLevel
		hasError bool
	}{
		{"DEBUG", LevelDebug, false},
		{"debug", LevelDebug, false},
		{"INFO", LevelInfo, false},
		{"info", LevelInfo, false},
		{"WARN", LevelWarn, false},
		{"WARNING", LevelWarn, false},
		{"ERROR", LevelError, false},
		{"FATAL", LevelFatal, false},
		{"INVALID", LevelInfo, true},
	}

	for _, tc := range testCases {
		level, err := ParseLogLevel(tc.input)

		if tc.hasError {
			if err == nil {
				t.Errorf("Expected error for input %s", tc.input)
			}
		} else {
			if err != nil {
				t.Errorf("Unexpected error for input %s: %v", tc.input, err)
			}
			if level != tc.expected {
				t.Errorf("For input %s, expected %s, got %s", tc.input, tc.expected.String(), level.String())
			}
		}
	}
}

func TestFieldCreation(t *testing.T) {
	// Test different field types
	stringField := String("key", "value")
	if stringField.Key != "key" || stringField.Value != "value" {
		t.Error("String field creation failed")
	}

	intField := Int("count", 42)
	if intField.Key != "count" || intField.Value != 42 {
		t.Error("Int field creation failed")
	}

	boolField := Bool("enabled", true)
	if boolField.Key != "enabled" || boolField.Value != true {
		t.Error("Bool field creation failed")
	}

	durationField := Duration("elapsed", time.Second)
	if durationField.Key != "elapsed" || durationField.Value != "1s" {
		t.Error("Duration field creation failed")
	}

	errorField := Error(errors.New("test error"))
	if errorField.Key != "error" || errorField.Value != "test error" {
		t.Error("Error field creation failed")
	}

	nilErrorField := Error(nil)
	if nilErrorField.Key != "error" || nilErrorField.Value != nil {
		t.Error("Nil error field creation failed")
	}
}

func TestJSONFormatter(t *testing.T) {
	formatter := &JSONFormatter{}

	entry := LogEntry{
		Timestamp: time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
		Level:     LevelInfo,
		Message:   "test message",
		Fields: map[string]interface{}{
			"key": "value",
		},
		Component: "test",
	}

	data, err := formatter.Format(entry)
	if err != nil {
		t.Fatalf("JSON formatting failed: %v", err)
	}

	// Parse back to verify
	var parsed LogEntry
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	if parsed.Message != "test message" {
		t.Errorf("Expected message 'test message', got '%s'", parsed.Message)
	}

	if parsed.Component != "test" {
		t.Errorf("Expected component 'test', got '%s'", parsed.Component)
	}
}

func TestTextFormatter(t *testing.T) {
	formatter := &TextFormatter{DisableColors: true}

	entry := LogEntry{
		Timestamp: time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
		Level:     LevelInfo,
		Message:   "test message",
		Fields: map[string]interface{}{
			"key": "value",
		},
		Component: "test",
	}

	data, err := formatter.Format(entry)
	if err != nil {
		t.Fatalf("Text formatting failed: %v", err)
	}

	output := string(data)

	if !strings.Contains(output, "test message") {
		t.Error("Expected output to contain message")
	}

	if !strings.Contains(output, "[INFO]") {
		t.Error("Expected output to contain log level")
	}

	if !strings.Contains(output, "[test]") {
		t.Error("Expected output to contain component")
	}

	if !strings.Contains(output, "key=value") {
		t.Error("Expected output to contain fields")
	}
}

func TestStructuredLogger(t *testing.T) {
	var buf bytes.Buffer
	logger := NewJSONLogger(LevelDebug, &buf)

	// Test basic logging
	logger.Info("test message", String("key", "value"))

	// Parse the output
	var entry LogEntry
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("Failed to parse log output: %v", err)
	}

	if entry.Level != LevelInfo {
		t.Errorf("Expected level INFO, got %s", entry.Level.String())
	}

	if entry.Message != "test message" {
		t.Errorf("Expected message 'test message', got '%s'", entry.Message)
	}

	if entry.Fields["key"] != "value" {
		t.Errorf("Expected field key=value, got %v", entry.Fields["key"])
	}
}

func TestLoggerLevels(t *testing.T) {
	var buf bytes.Buffer
	logger := NewJSONLogger(LevelWarn, &buf)

	// These should not be logged (below threshold)
	logger.Debug("debug message")
	logger.Info("info message")

	// These should be logged
	logger.Warn("warn message")
	logger.Error("error message")

	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")

	// Should only have 2 lines (warn and error)
	if len(lines) != 2 {
		t.Errorf("Expected 2 log lines, got %d", len(lines))
	}

	// Check that warn and error messages are present
	if !strings.Contains(output, "warn message") {
		t.Error("Expected warn message to be logged")
	}

	if !strings.Contains(output, "error message") {
		t.Error("Expected error message to be logged")
	}
}

func TestFormattedLogging(t *testing.T) {
	var buf bytes.Buffer
	logger := NewJSONLogger(LevelDebug, &buf)

	logger.Infof("formatted message: %s %d", "test", 42)

	var entry LogEntry
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("Failed to parse log output: %v", err)
	}

	expected := "formatted message: test 42"
	if entry.Message != expected {
		t.Errorf("Expected message '%s', got '%s'", expected, entry.Message)
	}
}

func TestLoggerWithFields(t *testing.T) {
	var buf bytes.Buffer
	logger := NewJSONLogger(LevelDebug, &buf)

	// Create logger with fields
	fieldLogger := logger.WithFields(
		String("service", "test"),
		Int("version", 1),
	)

	fieldLogger.Info("test message", String("extra", "field"))

	var entry LogEntry
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("Failed to parse log output: %v", err)
	}

	if entry.Fields["service"] != "test" {
		t.Error("Expected service field to be preserved")
	}

	if entry.Fields["version"] != float64(1) { // JSON unmarshals numbers as float64
		t.Error("Expected version field to be preserved")
	}

	if entry.Fields["extra"] != "field" {
		t.Error("Expected extra field to be added")
	}
}

func TestLoggerWithComponent(t *testing.T) {
	var buf bytes.Buffer
	logger := NewJSONLogger(LevelDebug, &buf)

	componentLogger := logger.WithComponent("database")
	componentLogger.Info("connection established")

	var entry LogEntry
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("Failed to parse log output: %v", err)
	}

	if entry.Component != "database" {
		t.Errorf("Expected component 'database', got '%s'", entry.Component)
	}
}

func TestLoggerWithError(t *testing.T) {
	var buf bytes.Buffer
	logger := NewJSONLogger(LevelDebug, &buf)

	testErr := errors.New("test error")
	errorLogger := logger.WithError(testErr)
	errorLogger.Error("operation failed")

	var entry LogEntry
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("Failed to parse log output: %v", err)
	}

	if entry.Error != "test error" {
		t.Errorf("Expected error 'test error', got '%s'", entry.Error)
	}
}

func TestLoggerWithContext(t *testing.T) {
	var buf bytes.Buffer
	logger := NewJSONLogger(LevelDebug, &buf)

	ctx := context.WithValue(context.Background(), "trace_id", "abc123")
	contextLogger := logger.WithContext(ctx)
	contextLogger.Info("request processed")

	var entry LogEntry
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("Failed to parse log output: %v", err)
	}

	if entry.TraceID != "abc123" {
		t.Errorf("Expected trace ID 'abc123', got '%s'", entry.TraceID)
	}
}

func TestLoggerSetLevel(t *testing.T) {
	var buf bytes.Buffer
	logger := NewJSONLogger(LevelInfo, &buf)

	// Debug should not be logged initially
	logger.Debug("debug message")
	if buf.Len() > 0 {
		t.Error("Debug message should not be logged at INFO level")
	}

	// Change level to DEBUG
	logger.SetLevel(LevelDebug)

	// Now debug should be logged
	logger.Debug("debug message")
	if buf.Len() == 0 {
		t.Error("Debug message should be logged at DEBUG level")
	}

	// Verify level getter
	if logger.GetLevel() != LevelDebug {
		t.Error("GetLevel should return DEBUG")
	}
}

func TestLoggerConcurrency(t *testing.T) {
	var buf bytes.Buffer
	logger := NewJSONLogger(LevelDebug, &buf)

	// Test concurrent logging
	done := make(chan bool)

	for i := 0; i < 10; i++ {
		go func(id int) {
			logger.Infof("message from goroutine %d", id)
			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")

	if len(lines) != 10 {
		t.Errorf("Expected 10 log lines, got %d", len(lines))
	}
}

func TestGlobalLogger(t *testing.T) {
	var buf bytes.Buffer
	originalLogger := GetGlobalLogger()

	// Set a test logger
	testLogger := NewJSONLogger(LevelDebug, &buf)
	SetGlobalLogger(testLogger)

	// Test global functions
	Info("global info message", String("test", "value"))
	Debugf("global debug: %s", "formatted")

	// Restore original logger
	SetGlobalLogger(originalLogger)

	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")

	if len(lines) != 2 {
		t.Errorf("Expected 2 log lines, got %d", len(lines))
	}

	if !strings.Contains(output, "global info message") {
		t.Error("Expected global info message")
	}

	if !strings.Contains(output, "global debug: formatted") {
		t.Error("Expected global debug message")
	}
}

func TestLogRotator(t *testing.T) {
	// Create a temporary file for testing
	rotator := NewLogRotator("test.log", 100, 7, 3, false)

	// Write some data
	data := []byte("test log entry\n")
	n, err := rotator.Write(data)

	if err != nil {
		t.Fatalf("Failed to write to rotator: %v", err)
	}

	if n != len(data) {
		t.Errorf("Expected to write %d bytes, wrote %d", len(data), n)
	}

	// Clean up
	rotator.Close()
}

func BenchmarkJSONLogging(b *testing.B) {
	var buf bytes.Buffer
	logger := NewJSONLogger(LevelInfo, &buf)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Info("benchmark message",
			String("key1", "value1"),
			Int("key2", i),
			Bool("key3", true))
	}
}

func BenchmarkTextLogging(b *testing.B) {
	var buf bytes.Buffer
	logger := NewTextLogger(LevelInfo, &buf)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Info("benchmark message",
			String("key1", "value1"),
			Int("key2", i),
			Bool("key3", true))
	}
}

func BenchmarkFormattedLogging(b *testing.B) {
	var buf bytes.Buffer
	logger := NewJSONLogger(LevelInfo, &buf)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Infof("benchmark message %d with %s", i, "formatting")
	}
}
