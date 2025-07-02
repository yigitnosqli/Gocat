package errors

import (
	"fmt"
	"runtime"
	"strings"
	"time"
)

// ErrorType represents different types of errors
type ErrorType string

const (
	ErrorTypeNetwork    ErrorType = "network"
	ErrorTypeValidation ErrorType = "validation"
	ErrorTypeSecurity   ErrorType = "security"
	ErrorTypeFileSystem ErrorType = "filesystem"
	ErrorTypeTimeout    ErrorType = "timeout"
	ErrorTypePermission ErrorType = "permission"
	ErrorTypeConfig     ErrorType = "config"
	ErrorTypeInternal   ErrorType = "internal"
	ErrorTypeUser       ErrorType = "user"
)

// Severity represents error severity levels
type Severity string

const (
	SeverityLow      Severity = "low"
	SeverityMedium   Severity = "medium"
	SeverityHigh     Severity = "high"
	SeverityCritical Severity = "critical"
)

// GoCatError represents a structured error with additional context
type GoCatError struct {
	Type         ErrorType              `json:"type"`
	Severity     Severity               `json:"severity"`
	Message      string                 `json:"message"`
	Code         string                 `json:"code"`
	Cause        error                  `json:"cause,omitempty"`
	Context      map[string]interface{} `json:"context,omitempty"`
	Timestamp    time.Time              `json:"timestamp"`
	StackTrace   []string               `json:"stack_trace,omitempty"`
	Suggestion   string                 `json:"suggestion,omitempty"`
	Retryable    bool                   `json:"retryable"`
	UserFriendly string                 `json:"user_friendly,omitempty"`
}

// Error implements the error interface
func (e *GoCatError) Error() string {
	if e.UserFriendly != "" {
		return e.UserFriendly
	}
	return e.Message
}

// Unwrap returns the underlying error
func (e *GoCatError) Unwrap() error {
	return e.Cause
}

// Is checks if the error matches the target
func (e *GoCatError) Is(target error) bool {
	if t, ok := target.(*GoCatError); ok {
		return e.Type == t.Type && e.Code == t.Code
	}
	return false
}

// NewError creates a new GoCatError
func NewError(errorType ErrorType, severity Severity, code, message string) *GoCatError {
	return &GoCatError{
		Type:       errorType,
		Severity:   severity,
		Message:    message,
		Code:       code,
		Timestamp:  time.Now(),
		Context:    make(map[string]interface{}),
		StackTrace: captureStackTrace(),
		Retryable:  false,
	}
}

// WithCause adds a cause to the error
func (e *GoCatError) WithCause(cause error) *GoCatError {
	e.Cause = cause
	return e
}

// WithContext adds context information to the error
func (e *GoCatError) WithContext(key string, value interface{}) *GoCatError {
	if e.Context == nil {
		e.Context = make(map[string]interface{})
	}
	e.Context[key] = value
	return e
}

// WithSuggestion adds a suggestion for fixing the error
func (e *GoCatError) WithSuggestion(suggestion string) *GoCatError {
	e.Suggestion = suggestion
	return e
}

// WithUserFriendly sets a user-friendly error message
func (e *GoCatError) WithUserFriendly(message string) *GoCatError {
	e.UserFriendly = message
	return e
}

// SetRetryable marks the error as retryable
func (e *GoCatError) SetRetryable(retryable bool) *GoCatError {
	e.Retryable = retryable
	return e
}

// captureStackTrace captures the current stack trace
func captureStackTrace() []string {
	var stack []string
	for i := 2; i < 10; i++ { // Skip this function and NewError
		_, file, line, ok := runtime.Caller(i)
		if !ok {
			break
		}
		// Shorten file path
		if idx := strings.LastIndex(file, "/"); idx != -1 {
			file = file[idx+1:]
		}
		stack = append(stack, fmt.Sprintf("%s:%d", file, line))
	}
	return stack
}

// Common error constructors

// NetworkError creates a network-related error
func NetworkError(code, message string) *GoCatError {
	return NewError(ErrorTypeNetwork, SeverityHigh, code, message).SetRetryable(true)
}

// ValidationError creates a validation error
func ValidationError(code, message string) *GoCatError {
	return NewError(ErrorTypeValidation, SeverityMedium, code, message)
}

// SecurityError creates a security-related error
func SecurityError(code, message string) *GoCatError {
	return NewError(ErrorTypeSecurity, SeverityCritical, code, message)
}

// FileSystemError creates a filesystem-related error
func FileSystemError(code, message string) *GoCatError {
	return NewError(ErrorTypeFileSystem, SeverityMedium, code, message)
}

// TimeoutError creates a timeout error
func TimeoutError(code, message string) *GoCatError {
	return NewError(ErrorTypeTimeout, SeverityHigh, code, message).SetRetryable(true)
}

// PermissionError creates a permission error
func PermissionError(code, message string) *GoCatError {
	return NewError(ErrorTypePermission, SeverityHigh, code, message)
}

// ConfigError creates a configuration error
func ConfigError(code, message string) *GoCatError {
	return NewError(ErrorTypeConfig, SeverityMedium, code, message)
}

// InternalError creates an internal error
func InternalError(code, message string) *GoCatError {
	return NewError(ErrorTypeInternal, SeverityCritical, code, message)
}

// UserError creates a user error
func UserError(code, message string) *GoCatError {
	return NewError(ErrorTypeUser, SeverityLow, code, message)
}

// Predefined error codes and messages
var (
	// Network errors
	ErrConnectionFailed    = NetworkError("NET001", "Failed to establish connection")
	ErrConnectionTimeout   = TimeoutError("NET002", "Connection timeout")
	ErrConnectionRefused   = NetworkError("NET003", "Connection refused")
	ErrHostUnreachable     = NetworkError("NET004", "Host unreachable")
	ErrNetworkUnreachable  = NetworkError("NET005", "Network unreachable")
	ErrDNSResolutionFailed = NetworkError("NET006", "DNS resolution failed")

	// Validation errors
	ErrInvalidHostname = ValidationError("VAL001", "Invalid hostname")
	ErrInvalidPort     = ValidationError("VAL002", "Invalid port number")
	ErrInvalidProtocol = ValidationError("VAL003", "Invalid protocol")
	ErrInvalidAddress  = ValidationError("VAL004", "Invalid address format")
	ErrInvalidInput    = ValidationError("VAL005", "Invalid input")

	// Security errors
	ErrUnauthorized         = SecurityError("SEC001", "Unauthorized access")
	ErrForbidden            = SecurityError("SEC002", "Forbidden operation")
	ErrInsecureConnection   = SecurityError("SEC003", "Insecure connection")
	ErrCertificateInvalid   = SecurityError("SEC004", "Invalid certificate")
	ErrAuthenticationFailed = SecurityError("SEC005", "Authentication failed")

	// File system errors
	ErrFileNotFound      = FileSystemError("FS001", "File not found")
	ErrFilePermission    = PermissionError("FS002", "File permission denied")
	ErrDirectoryNotFound = FileSystemError("FS003", "Directory not found")
	ErrDiskFull          = FileSystemError("FS004", "Disk full")
	ErrFileCorrupted     = FileSystemError("FS005", "File corrupted")

	// Configuration errors
	ErrConfigNotFound   = ConfigError("CFG001", "Configuration file not found")
	ErrConfigInvalid    = ConfigError("CFG002", "Invalid configuration")
	ErrConfigPermission = PermissionError("CFG003", "Configuration file permission denied")

	// Internal errors
	ErrInternalFailure   = InternalError("INT001", "Internal failure")
	ErrMemoryAllocation  = InternalError("INT002", "Memory allocation failed")
	ErrResourceExhausted = InternalError("INT003", "Resource exhausted")
)

// WrapError wraps an existing error with additional context
func WrapError(err error, errorType ErrorType, severity Severity, code, message string) *GoCatError {
	gcErr := NewError(errorType, severity, code, message)
	gcErr.Cause = err
	return gcErr
}

// IsRetryable checks if an error is retryable
func IsRetryable(err error) bool {
	if gcErr, ok := err.(*GoCatError); ok {
		return gcErr.Retryable
	}
	return false
}

// GetErrorType returns the error type if it's a GoCatError
func GetErrorType(err error) ErrorType {
	if gcErr, ok := err.(*GoCatError); ok {
		return gcErr.Type
	}
	return ErrorTypeInternal
}

// GetSeverity returns the error severity if it's a GoCatError
func GetSeverity(err error) Severity {
	if gcErr, ok := err.(*GoCatError); ok {
		return gcErr.Severity
	}
	return SeverityMedium
}
