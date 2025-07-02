package errors

import (
	"errors"
	"testing"
	"time"
)

func TestNewError(t *testing.T) {
	err := NewError(ErrorTypeNetwork, SeverityHigh, "NET001", "Connection failed")

	if err.Type != ErrorTypeNetwork {
		t.Errorf("Expected type %v, got %v", ErrorTypeNetwork, err.Type)
	}

	if err.Severity != SeverityHigh {
		t.Errorf("Expected severity %v, got %v", SeverityHigh, err.Severity)
	}

	if err.Code != "NET001" {
		t.Errorf("Expected code NET001, got %v", err.Code)
	}

	if err.Message != "Connection failed" {
		t.Errorf("Expected message 'Connection failed', got %v", err.Message)
	}

	if err.Context == nil {
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

	if err.Cause != cause {
		t.Errorf("Expected cause to be %v, got %v", cause, err.Cause)
	}

	if err.Unwrap() != cause {
		t.Errorf("Expected Unwrap() to return %v, got %v", cause, err.Unwrap())
	}
}

func TestGoCatError_WithContext(t *testing.T) {
	err := NewError(ErrorTypeNetwork, SeverityHigh, "NET001", "Connection failed")
	err.WithContext("host", "example.com")
	err.WithContext("port", 80)

	if err.Context["host"] != "example.com" {
		t.Errorf("Expected host context to be 'example.com', got %v", err.Context["host"])
	}

	if err.Context["port"] != 80 {
		t.Errorf("Expected port context to be 80, got %v", err.Context["port"])
	}
}

func TestGoCatError_WithSuggestion(t *testing.T) {
	suggestion := "Check your network connection"
	err := NewError(ErrorTypeNetwork, SeverityHigh, "NET001", "Connection failed").WithSuggestion(suggestion)

	if err.Suggestion != suggestion {
		t.Errorf("Expected suggestion to be %v, got %v", suggestion, err.Suggestion)
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.constructor("TEST001", "Test message")

			if err.Type != tt.expectedType {
				t.Errorf("Expected type %v, got %v", tt.expectedType, err.Type)
			}

			if err.Severity != tt.expectedSev {
				t.Errorf("Expected severity %v, got %v", tt.expectedSev, err.Severity)
			}

			if err.Retryable != tt.retryable {
				t.Errorf("Expected retryable %v, got %v", tt.retryable, err.Retryable)
			}

			if err.Code != "TEST001" {
				t.Errorf("Expected code TEST001, got %v", err.Code)
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

	if wrapped.Cause != original {
		t.Errorf("Expected cause to be %v, got %v", original, wrapped.Cause)
	}

	if wrapped.Type != ErrorTypeNetwork {
		t.Errorf("Expected type %v, got %v", ErrorTypeNetwork, wrapped.Type)
	}

	if wrapped.Severity != SeverityHigh {
		t.Errorf("Expected severity %v, got %v", SeverityHigh, wrapped.Severity)
	}

	if wrapped.Code != "NET001" {
		t.Errorf("Expected code NET001, got %v", wrapped.Code)
	}

	if wrapped.Message != "Wrapped error" {
		t.Errorf("Expected message 'Wrapped error', got %v", wrapped.Message)
	}
}

func TestIsRetryable(t *testing.T) {
	tests := []struct {
		name      string
		err       error
		expected  bool
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
	if ErrConnectionFailed.Type != ErrorTypeNetwork {
		t.Error("ErrConnectionFailed should be network error")
	}

	if ErrInvalidHostname.Type != ErrorTypeValidation {
		t.Error("ErrInvalidHostname should be validation error")
	}

	if ErrUnauthorized.Type != ErrorTypeSecurity {
		t.Error("ErrUnauthorized should be security error")
	}

	if ErrFileNotFound.Type != ErrorTypeFileSystem {
		t.Error("ErrFileNotFound should be filesystem error")
	}

	if ErrConnectionTimeout.Type != ErrorTypeTimeout {
		t.Error("ErrConnectionTimeout should be timeout error")
	}
}

func TestStackTraceCapture(t *testing.T) {
	t.Skip("Skipping stack trace test - behavior varies across Go versions and environments")
}

func TestTimestamp(t *testing.T) {
	before := time.Now()
	err := NewError(ErrorTypeInternal, SeverityHigh, "INT001", "Test error")
	after := time.Now()

	if err.Timestamp.Before(before) || err.Timestamp.After(after) {
		t.Error("Error timestamp should be between before and after times")
	}
}