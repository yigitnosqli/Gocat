package errors

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestNewError(t *testing.T) {
	err := NewError(ErrorTypeNetwork, SeverityHigh, "NET001", "Connection failed")

	if err.Type() != ErrorTypeNetwork {
		t.Errorf("Expected type %v, got %v", ErrorTypeNetwork, err.Type())
	}

	if err.Severity() != SeverityHigh {
		t.Errorf("Expected severity %v, got %v", SeverityHigh, err.Severity())
	}

	if err.Code() != "NET001" {
		t.Errorf("Expected code NET001, got %v", err.Code())
	}

	if err.Message != "Connection failed" {
		t.Errorf("Expected message 'Connection failed', got %v", err.Message)
	}

	if err.Context() == nil {
		t.Error("Expected context to be initialized")
	}

	if len(err.StackTrace) == 0 {
		t.Error("Expected stack trace to be captured")
	}

	if err.Timestamp.IsZero() {
		t.Error("Expected timestamp to be set")
	}
}

func TestGoCatError_Error(t *testing.T) {
	tests := []struct {
		name         string
		message      string
		userFriendly string
		expected     string
	}{
		{
			name:     "without user friendly message",
			message:  "Internal error occurred",
			expected: "Internal error occurred",
		},
		{
			name:         "with user friendly message",
			message:      "Internal error occurred",
			userFriendly: "Something went wrong. Please try again.",
			expected:     "Something went wrong. Please try again.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewError(ErrorTypeInternal, SeverityHigh, "INT001", tt.message)
			if tt.userFriendly != "" {
				err.WithUserFriendly(tt.userFriendly)
			}

			if got := err.Error(); got != tt.expected {
				t.Errorf("Error() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestGoCatError_WithCause(t *testing.T) {
	cause := errors.New("original error")
	err := NewError(ErrorTypeNetwork, SeverityHigh, "NET001", "Connection failed").WithCause(cause)

	if err.Cause() != cause {
		t.Errorf("Expected cause to be %v, got %v", cause, err.Cause())
	}

	if err.Unwrap() != cause {
		t.Errorf("Expected Unwrap() to return %v, got %v", cause, err.Unwrap())
	}
}

func TestGoCatError_WithContext(t *testing.T) {
	err := NewError(ErrorTypeNetwork, SeverityHigh, "NET001", "Connection failed")
	err.WithContext("host", "example.com")
	err.WithContext("port", 80)

	context := err.Context()
	if context["host"] != "example.com" {
		t.Errorf("Expected host context to be 'example.com', got %v", context["host"])
	}

	if context["port"] != 80 {
		t.Errorf("Expected port context to be 80, got %v", context["port"])
	}
}

func TestGoCatError_WithSuggestion(t *testing.T) {
	suggestion := "Check your network connection"
	err := NewError(ErrorTypeNetwork, SeverityHigh, "NET001", "Connection failed").WithSuggestion(suggestion)

	if err.Suggestion() != suggestion {
		t.Errorf("Expected suggestion to be %v, got %v", suggestion, err.Suggestion())
	}
}

func TestGoCatError_SetRetryable(t *testing.T) {
	err := NewError(ErrorTypeNetwork, SeverityHigh, "NET001", "Connection failed")

	// Initially should not be retryable
	if err.Retryable {
		t.Error("Expected error to not be retryable initially")
	}

	// Set as retryable
	err.SetRetryable(true)
	if !err.Retryable {
		t.Error("Expected error to be retryable after setting")
	}
}

func TestGoCatError_Is(t *testing.T) {
	err1 := NewError(ErrorTypeNetwork, SeverityHigh, "NET001", "Connection failed")
	err2 := NewError(ErrorTypeNetwork, SeverityHigh, "NET001", "Different message")
	err3 := NewError(ErrorTypeNetwork, SeverityHigh, "NET002", "Connection failed")
	err4 := NewError(ErrorTypeValidation, SeverityHigh, "NET001", "Connection failed")
	stdErr := errors.New("standard error")

	if !err1.Is(err2) {
		t.Error("Expected errors with same type and code to be equal")
	}

	if err1.Is(err3) {
		t.Error("Expected errors with different codes to not be equal")
	}

	if err1.Is(err4) {
		t.Error("Expected errors with different types to not be equal")
	}

	if err1.Is(stdErr) {
		t.Error("Expected GoCatError to not be equal to standard error")
	}
}

func TestErrorConstructors(t *testing.T) {
	tests := []struct {
		name         string
		constructor  func(string, string) *GoCatError
		expectedType ErrorType
		expectedSev  Severity
		retryable    bool
	}{
		{"NetworkError", NetworkError, ErrorTypeNetwork, SeverityHigh, true},
		{"ValidationError", ValidationError, ErrorTypeValidation, SeverityMedium, false},
		{"SecurityError", SecurityError, ErrorTypeSecurity, SeverityCritical, false},
		{"FileSystemError", FileSystemError, ErrorTypeFileSystem, SeverityMedium, false},
		{"TimeoutError", TimeoutError, ErrorTypeTimeout, SeverityHigh, true},
		{"PermissionError", PermissionError, ErrorTypePermission, SeverityHigh, false},
		{"ConfigError", ConfigError, ErrorTypeConfig, SeverityMedium, false},
		{"InternalError", InternalError, ErrorTypeInternal, SeverityCritical, false},
		{"UserError", UserError, ErrorTypeUser, SeverityLow, false},
		{"SystemError", SystemError, ErrorTypeSystem, SeverityHigh, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.constructor("TEST001", "Test message")

			if err.Type() != tt.expectedType {
				t.Errorf("Expected type %v, got %v", tt.expectedType, err.Type())
			}

			if err.Severity() != tt.expectedSev {
				t.Errorf("Expected severity %v, got %v", tt.expectedSev, err.Severity())
			}

			if err.IsRetryable() != tt.retryable {
				t.Errorf("Expected retryable %v, got %v", tt.retryable, err.IsRetryable())
			}

			if err.Code() != "TEST001" {
				t.Errorf("Expected code TEST001, got %v", err.Code())
			}

			if err.Message != "Test message" {
				t.Errorf("Expected message 'Test message', got %v", err.Message)
			}
		})
	}
}

func TestWrapError(t *testing.T) {
	original := errors.New("original error")
	wrapped := WrapError(original, ErrorTypeNetwork, SeverityHigh, "NET001", "Wrapped error")

	if wrapped.Cause() != original {
		t.Errorf("Expected cause to be %v, got %v", original, wrapped.Cause())
	}

	if wrapped.Type() != ErrorTypeNetwork {
		t.Errorf("Expected type %v, got %v", ErrorTypeNetwork, wrapped.Type())
	}

	if wrapped.Severity() != SeverityHigh {
		t.Errorf("Expected severity %v, got %v", SeverityHigh, wrapped.Severity())
	}

	if wrapped.Code() != "NET001" {
		t.Errorf("Expected code NET001, got %v", wrapped.Code())
	}

	if wrapped.Message != "Wrapped error" {
		t.Errorf("Expected message 'Wrapped error', got %v", wrapped.Message)
	}
}

func TestIsRetryable(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "retryable GoCatError",
			err:      NetworkError("NET001", "Connection failed"),
			expected: true,
		},
		{
			name:     "non-retryable GoCatError",
			err:      ValidationError("VAL001", "Invalid input"),
			expected: false,
		},
		{
			name:     "standard error",
			err:      errors.New("standard error"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsRetryable(tt.err); got != tt.expected {
				t.Errorf("IsRetryable() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestGetErrorType(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected ErrorType
	}{
		{
			name:     "GoCatError",
			err:      NetworkError("NET001", "Connection failed"),
			expected: ErrorTypeNetwork,
		},
		{
			name:     "standard error",
			err:      errors.New("standard error"),
			expected: ErrorTypeInternal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetErrorType(tt.err); got != tt.expected {
				t.Errorf("GetErrorType() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestGetSeverity(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected Severity
	}{
		{
			name:     "GoCatError",
			err:      SecurityError("SEC001", "Unauthorized"),
			expected: SeverityCritical,
		},
		{
			name:     "standard error",
			err:      errors.New("standard error"),
			expected: SeverityMedium,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetSeverity(tt.err); got != tt.expected {
				t.Errorf("GetSeverity() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestPredefinedErrors(t *testing.T) {
	// Test that predefined errors have correct properties
	if ErrConnectionFailed.Type() != ErrorTypeNetwork {
		t.Error("ErrConnectionFailed should be network error")
	}

	if ErrInvalidHostname.Type() != ErrorTypeValidation {
		t.Error("ErrInvalidHostname should be validation error")
	}

	if ErrUnauthorized.Type() != ErrorTypeSecurity {
		t.Error("ErrUnauthorized should be security error")
	}

	if ErrFileNotFound.Type() != ErrorTypeFileSystem {
		t.Error("ErrFileNotFound should be filesystem error")
	}

	if ErrConnectionTimeout.Type() != ErrorTypeTimeout {
		t.Error("ErrConnectionTimeout should be timeout error")
	}

	if ErrSystemFailure.Type() != ErrorTypeSystem {
		t.Error("ErrSystemFailure should be system error")
	}
}

func TestStackTraceCapture(t *testing.T) {
	err := NewError(ErrorTypeInternal, SeverityHigh, "INT001", "Test error")

	if len(err.StackTrace) == 0 {
		t.Error("Expected stack trace to be captured")
	}

	// Check that stack trace contains this test function
	found := false
	for _, frame := range err.StackTrace {
		if strings.Contains(frame, "errors_test.go") {
			found = true
			break
		}
	}

	if !found {
		t.Error("Expected stack trace to contain test file reference")
	}

	// Verify stack trace format (file:line)
	for _, frame := range err.StackTrace {
		if !strings.Contains(frame, ":") {
			t.Errorf("Invalid stack trace frame format: %s", frame)
		}
	}
}

func TestTimestamp(t *testing.T) {
	before := time.Now()
	err := NewError(ErrorTypeInternal, SeverityHigh, "INT001", "Test error")
	after := time.Now()

	if err.Timestamp.Before(before) || err.Timestamp.After(after) {
		t.Error("Error timestamp should be between before and after times")
	}
}
func TestStructuredErrorInterface(t *testing.T) {
	err := NewError(ErrorTypeNetwork, SeverityHigh, "NET001", "Connection failed")
	err.WithUserFriendly("Network connection failed")
	err.WithSuggestion("Check your internet connection")
	err.WithContext("host", "example.com")
	err.SetRetryable(true)

	// Test that GoCatError implements StructuredError interface
	var structuredErr StructuredError = err

	if structuredErr.Type() != ErrorTypeNetwork {
		t.Errorf("Expected type %v, got %v", ErrorTypeNetwork, structuredErr.Type())
	}

	if structuredErr.Severity() != SeverityHigh {
		t.Errorf("Expected severity %v, got %v", SeverityHigh, structuredErr.Severity())
	}

	if structuredErr.Code() != "NET001" {
		t.Errorf("Expected code NET001, got %v", structuredErr.Code())
	}

	if structuredErr.UserFriendlyMessage() != "Network connection failed" {
		t.Errorf("Expected user friendly message 'Network connection failed', got %v", structuredErr.UserFriendlyMessage())
	}

	if structuredErr.Suggestion() != "Check your internet connection" {
		t.Errorf("Expected suggestion 'Check your internet connection', got %v", structuredErr.Suggestion())
	}

	if !structuredErr.IsRetryable() {
		t.Error("Expected error to be retryable")
	}

	context := structuredErr.Context()
	if context["host"] != "example.com" {
		t.Errorf("Expected host context to be 'example.com', got %v", context["host"])
	}

	if structuredErr.Cause() != nil {
		t.Errorf("Expected no cause, got %v", structuredErr.Cause())
	}
}

func TestStructuredErrorWithCause(t *testing.T) {
	originalErr := errors.New("original error")
	err := NewError(ErrorTypeNetwork, SeverityHigh, "NET001", "Connection failed")
	err.WithCause(originalErr)

	var structuredErr StructuredError = err

	if structuredErr.Cause() != originalErr {
		t.Errorf("Expected cause to be %v, got %v", originalErr, structuredErr.Cause())
	}
}

func TestNewPanicRecovery(t *testing.T) {
	var loggedMessages []string
	var recordedMetrics []map[string]interface{}

	config := PanicRecoveryConfig{
		Logger: func(format string, args ...interface{}) {
			loggedMessages = append(loggedMessages, fmt.Sprintf(format, args...))
		},
		MetricsFunc: func(name string, metrics map[string]interface{}) {
			recordedMetrics = append(recordedMetrics, metrics)
		},
		ShutdownFunc: func() {
			// Shutdown function for testing
		},
		EnableMetrics: true,
	}

	pr := NewPanicRecovery(config)

	if pr.logger == nil {
		t.Error("Expected logger to be set")
	}

	if pr.metricsFunc == nil {
		t.Error("Expected metricsFunc to be set")
	}

	if pr.shutdownFunc == nil {
		t.Error("Expected shutdownFunc to be set")
	}

	if !pr.enableMetrics {
		t.Error("Expected enableMetrics to be true")
	}
}

func TestPanicRecovery_RecoverWithContext(t *testing.T) {
	var loggedMessages []string
	var recordedMetrics []map[string]interface{}

	config := PanicRecoveryConfig{
		Logger: func(format string, args ...interface{}) {
			loggedMessages = append(loggedMessages, fmt.Sprintf(format, args...))
		},
		MetricsFunc: func(name string, metrics map[string]interface{}) {
			recordedMetrics = append(recordedMetrics, metrics)
		},
		EnableMetrics: true,
	}

	pr := NewPanicRecovery(config)

	// Test panic recovery
	func() {
		defer pr.RecoverWithContext("test_operation", map[string]interface{}{
			"test_key": "test_value",
		})
		panic("test panic")
	}()

	// Verify logging occurred
	if len(loggedMessages) == 0 {
		t.Error("Expected panic to be logged")
	}

	// Verify metrics were recorded
	if len(recordedMetrics) == 0 {
		t.Error("Expected panic metrics to be recorded")
	}

	// Check metrics content
	metrics := recordedMetrics[0]
	if metrics["operation"] != "test_operation" {
		t.Errorf("Expected operation to be 'test_operation', got %v", metrics["operation"])
	}

	if metrics["panic_type"] != "string" {
		t.Errorf("Expected panic_type to be 'string', got %v", metrics["panic_type"])
	}
}

func TestPanicRecovery_RecoverWithCallback(t *testing.T) {
	var callbackErr *GoCatError
	callbackCalled := false

	config := PanicRecoveryConfig{
		Logger:        func(format string, args ...interface{}) {},
		EnableMetrics: false,
	}

	pr := NewPanicRecovery(config)

	// Test panic recovery with callback
	func() {
		defer pr.RecoverWithCallback("test_operation", nil, func(err *GoCatError) {
			callbackCalled = true
			callbackErr = err
		})
		panic("test panic")
	}()

	if !callbackCalled {
		t.Error("Expected callback to be called")
	}

	if callbackErr == nil {
		t.Error("Expected callback to receive error")
	}

	if callbackErr.Type() != ErrorTypeSystem {
		t.Errorf("Expected error type to be %v, got %v", ErrorTypeSystem, callbackErr.Type())
	}

	if callbackErr.Severity() != SeverityCritical {
		t.Errorf("Expected severity to be %v, got %v", SeverityCritical, callbackErr.Severity())
	}
}

func TestPanicRecovery_isCriticalPanic(t *testing.T) {
	pr := NewPanicRecovery(PanicRecoveryConfig{})

	tests := []struct {
		name       string
		panicValue interface{}
		operation  string
		expected   bool
	}{
		{
			name:       "out of memory panic",
			panicValue: "runtime error: out of memory",
			operation:  "normal_operation",
			expected:   true,
		},
		{
			name:       "fatal error panic",
			panicValue: "fatal error: something went wrong",
			operation:  "normal_operation",
			expected:   true,
		},
		{
			name:       "critical operation panic",
			panicValue: "normal panic",
			operation:  "main_initialization",
			expected:   true,
		},
		{
			name:       "server start panic",
			panicValue: "normal panic",
			operation:  "server_start",
			expected:   true,
		},
		{
			name:       "normal panic",
			panicValue: "normal panic",
			operation:  "normal_operation",
			expected:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			panicErr := pr.createPanicError(tt.panicValue, tt.operation, nil)
			result := pr.isCriticalPanic(panicErr)

			if result != tt.expected {
				t.Errorf("Expected isCriticalPanic to return %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestPanicRecovery_SafeGo(t *testing.T) {
	var loggedMessages []string
	panicRecovered := false

	config := PanicRecoveryConfig{
		Logger: func(format string, args ...interface{}) {
			loggedMessages = append(loggedMessages, fmt.Sprintf(format, args...))
			panicRecovered = true
		},
		EnableMetrics: false,
	}

	pr := NewPanicRecovery(config)

	// Test SafeGo with panic
	done := make(chan bool, 1)
	pr.SafeGo("test_goroutine", func() {
		defer func() { done <- true }()
		panic("goroutine panic")
	})

	// Wait for goroutine to complete
	<-done

	// Give some time for panic recovery to complete
	time.Sleep(10 * time.Millisecond)

	if !panicRecovered {
		t.Error("Expected panic to be recovered in goroutine")
	}
}

func TestPanicRecovery_SafeGoWithContext(t *testing.T) {
	var recordedMetrics []map[string]interface{}

	config := PanicRecoveryConfig{
		Logger: func(format string, args ...interface{}) {},
		MetricsFunc: func(name string, metrics map[string]interface{}) {
			recordedMetrics = append(recordedMetrics, metrics)
		},
		EnableMetrics: true,
	}

	pr := NewPanicRecovery(config)

	// Test SafeGoWithContext
	done := make(chan bool, 1)
	context := map[string]interface{}{
		"custom_key": "custom_value",
	}

	pr.SafeGoWithContext("test_goroutine", context, func() {
		defer func() { done <- true }()
		panic("goroutine panic with context")
	})

	// Wait for goroutine to complete
	<-done

	// Give some time for panic recovery to complete
	time.Sleep(10 * time.Millisecond)

	if len(recordedMetrics) == 0 {
		t.Error("Expected metrics to be recorded")
	}

	// Verify custom context was preserved
	metrics := recordedMetrics[0]
	if metrics["operation"] != "test_goroutine" {
		t.Errorf("Expected operation to be 'test_goroutine', got %v", metrics["operation"])
	}
}

func TestDefaultPanicRecovery(t *testing.T) {
	// Test that default panic recovery exists and works
	if DefaultPanicRecovery == nil {
		t.Error("Expected DefaultPanicRecovery to be initialized")
	}

	// Test convenience functions don't panic
	func() {
		defer Recover("test_operation")
		// No panic, should complete normally
	}()

	func() {
		defer RecoverWithContext("test_operation", map[string]interface{}{
			"test": "value",
		})
		// No panic, should complete normally
	}()
}

func TestConvenienceFunctions(t *testing.T) {
	// Test that convenience functions work with actual panics
	recovered := false

	// Temporarily replace default logger to capture output
	originalPR := DefaultPanicRecovery
	defer func() { DefaultPanicRecovery = originalPR }()

	DefaultPanicRecovery = NewPanicRecovery(PanicRecoveryConfig{
		Logger: func(format string, args ...interface{}) {
			recovered = true
		},
		EnableMetrics: false,
	})

	func() {
		defer Recover("convenience_test")
		panic("test panic")
	}()

	if !recovered {
		t.Error("Expected panic to be recovered by convenience function")
	}

	// Reset for next test
	recovered = false

	func() {
		defer RecoverWithContext("convenience_test", map[string]interface{}{
			"test": "context",
		})
		panic("test panic with context")
	}()

	if !recovered {
		t.Error("Expected panic to be recovered by convenience function with context")
	}
}

func TestPanicRecovery_CriticalPanicShutdown(t *testing.T) {
	shutdownCalled := false

	config := PanicRecoveryConfig{
		Logger: func(format string, args ...interface{}) {},
		ShutdownFunc: func() {
			shutdownCalled = true
		},
		EnableMetrics: false,
	}

	pr := NewPanicRecovery(config)

	// Test critical panic triggers shutdown
	func() {
		defer pr.RecoverWithContext("main_initialization", nil)
		panic("fatal error: critical system failure")
	}()

	if !shutdownCalled {
		t.Error("Expected shutdown to be called for critical panic")
	}
}

func TestCreatePanicError(t *testing.T) {
	pr := NewPanicRecovery(PanicRecoveryConfig{})

	panicValue := "test panic"
	operation := "test_operation"
	context := map[string]interface{}{
		"custom_key": "custom_value",
	}

	panicErr := pr.createPanicError(panicValue, operation, context)

	if panicErr.Type() != ErrorTypeSystem {
		t.Errorf("Expected error type to be %v, got %v", ErrorTypeSystem, panicErr.Type())
	}

	if panicErr.Severity() != SeverityCritical {
		t.Errorf("Expected severity to be %v, got %v", SeverityCritical, panicErr.Severity())
	}

	if panicErr.Code() != "SYS004" {
		t.Errorf("Expected code to be SYS004, got %v", panicErr.Code())
	}

	if !strings.Contains(panicErr.Message, operation) {
		t.Errorf("Expected message to contain operation '%s', got %v", operation, panicErr.Message)
	}

	// Check context
	if panicErr.Context()["panic_value"] != panicValue {
		t.Errorf("Expected panic_value to be %v, got %v", panicValue, panicErr.Context()["panic_value"])
	}

	if panicErr.Context()["operation"] != operation {
		t.Errorf("Expected operation to be %v, got %v", operation, panicErr.Context()["operation"])
	}

	if panicErr.Context()["custom_key"] != "custom_value" {
		t.Errorf("Expected custom_key to be 'custom_value', got %v", panicErr.Context()["custom_key"])
	}

	if panicErr.UserFriendlyMessage() == "" {
		t.Error("Expected user-friendly message to be set")
	}

	if panicErr.Suggestion() == "" {
		t.Error("Expected suggestion to be set")
	}
}
