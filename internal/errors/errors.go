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
	ErrorTypeValidation ErrorType = "validation"
	ErrorTypeNetwork    ErrorType = "network"
	ErrorTypeTimeout    ErrorType = "timeout"
	ErrorTypeSecurity   ErrorType = "security"
	ErrorTypeConfig     ErrorType = "config"
	ErrorTypeSystem     ErrorType = "system"
	ErrorTypeFileSystem ErrorType = "filesystem"
	ErrorTypePermission ErrorType = "permission"
	ErrorTypeInternal   ErrorType = "internal"
	ErrorTypeUser       ErrorType = "user"
)

// ErrorSeverity represents error severity levels (alias for backward compatibility)
type ErrorSeverity string

const (
	SeverityLow      ErrorSeverity = "low"
	SeverityMedium   ErrorSeverity = "medium"
	SeverityHigh     ErrorSeverity = "high"
	SeverityCritical ErrorSeverity = "critical"
)

// Severity is an alias for ErrorSeverity for backward compatibility
type Severity = ErrorSeverity

// StructuredError interface defines the contract for structured error handling
type StructuredError interface {
	error
	Type() ErrorType
	Severity() ErrorSeverity
	Code() string
	UserFriendlyMessage() string
	Suggestion() string
	Cause() error
	IsRetryable() bool
	Context() map[string]interface{}
}

// GoCatError represents a structured error with additional context
type GoCatError struct {
	ErrorType       ErrorType              `json:"type"`
	ErrorSeverity   Severity               `json:"severity"`
	Message         string                 `json:"message"`
	ErrorCode       string                 `json:"code"`
	ErrorCause      error                  `json:"cause,omitempty"`
	ErrorContext    map[string]interface{} `json:"context,omitempty"`
	Timestamp       time.Time              `json:"timestamp"`
	StackTrace      []string               `json:"stack_trace,omitempty"`
	ErrorSuggestion string                 `json:"suggestion,omitempty"`
	Retryable       bool                   `json:"retryable"`
	UserFriendly    string                 `json:"user_friendly,omitempty"`
}

// Error implements the error interface
func (e *GoCatError) Error() string {
	if e.UserFriendly != "" {
		return e.UserFriendly
	}
	return e.Message
}

// Type returns the error type
func (e *GoCatError) Type() ErrorType {
	return e.ErrorType
}

// Severity returns the error severity
func (e *GoCatError) Severity() ErrorSeverity {
	return e.ErrorSeverity
}

// Code returns the error code
func (e *GoCatError) Code() string {
	return e.ErrorCode
}

// UserFriendlyMessage returns the user-friendly error message
func (e *GoCatError) UserFriendlyMessage() string {
	if e.UserFriendly != "" {
		return e.UserFriendly
	}
	return e.Message
}

// Suggestion returns the error suggestion
func (e *GoCatError) Suggestion() string {
	return e.ErrorSuggestion
}

// Cause returns the underlying error
func (e *GoCatError) Cause() error {
	return e.ErrorCause
}

// IsRetryable returns whether the error is retryable
func (e *GoCatError) IsRetryable() bool {
	return e.Retryable
}

// Context returns the error context
func (e *GoCatError) Context() map[string]interface{} {
	if e.ErrorContext == nil {
		return make(map[string]interface{})
	}
	return e.ErrorContext
}

// Unwrap returns the underlying error
func (e *GoCatError) Unwrap() error {
	return e.ErrorCause
}

// Is checks if the error matches the target
func (e *GoCatError) Is(target error) bool {
	if t, ok := target.(*GoCatError); ok {
		return e.ErrorType == t.ErrorType && e.ErrorCode == t.ErrorCode
	}
	return false
}

// NewError creates a new GoCatError
func NewError(errorType ErrorType, severity Severity, code, message string) *GoCatError {
	return &GoCatError{
		ErrorType:     errorType,
		ErrorSeverity: severity,
		Message:       message,
		ErrorCode:     code,
		Timestamp:     time.Now(),
		ErrorContext:  make(map[string]interface{}),
		StackTrace:    captureStackTrace(),
		Retryable:     false,
	}
}

// WithCause adds a cause to the error
func (e *GoCatError) WithCause(cause error) *GoCatError {
	e.ErrorCause = cause
	return e
}

// WithContext adds context information to the error
func (e *GoCatError) WithContext(key string, value interface{}) *GoCatError {
	if e.ErrorContext == nil {
		e.ErrorContext = make(map[string]interface{})
	}
	e.ErrorContext[key] = value
	return e
}

// WithSuggestion adds a suggestion for fixing the error
func (e *GoCatError) WithSuggestion(suggestion string) *GoCatError {
	e.ErrorSuggestion = suggestion
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

// SystemError creates a system-related error
func SystemError(code, message string) *GoCatError {
	return NewError(ErrorTypeSystem, SeverityHigh, code, message)
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

	// System errors
	ErrSystemFailure      = SystemError("SYS001", "System failure")
	ErrProcessFailed      = SystemError("SYS002", "Process execution failed")
	ErrServiceUnavailable = SystemError("SYS003", "Service unavailable")
)

// WrapError wraps an existing error with additional context
func WrapError(err error, errorType ErrorType, severity Severity, code, message string) *GoCatError {
	gcErr := NewError(errorType, severity, code, message)
	gcErr.ErrorCause = err
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
		return gcErr.ErrorType
	}
	return ErrorTypeInternal
}

// GetSeverity returns the error severity if it's a GoCatError
func GetSeverity(err error) Severity {
	if gcErr, ok := err.(*GoCatError); ok {
		return gcErr.ErrorSeverity
	}
	return SeverityMedium
}

// PanicRecovery provides context-aware panic recovery mechanisms
type PanicRecovery struct {
	logger        func(string, ...interface{})
	metricsFunc   func(string, map[string]interface{})
	shutdownFunc  func()
	enableMetrics bool
}

// PanicRecoveryConfig configures panic recovery behavior
type PanicRecoveryConfig struct {
	Logger        func(string, ...interface{})
	MetricsFunc   func(string, map[string]interface{})
	ShutdownFunc  func()
	EnableMetrics bool
}

// NewPanicRecovery creates a new panic recovery instance
func NewPanicRecovery(config PanicRecoveryConfig) *PanicRecovery {
	return &PanicRecovery{
		logger:        config.Logger,
		metricsFunc:   config.MetricsFunc,
		shutdownFunc:  config.ShutdownFunc,
		enableMetrics: config.EnableMetrics,
	}
}

// RecoverWithContext handles panic recovery with contextual information
func (pr *PanicRecovery) RecoverWithContext(operation string, context map[string]interface{}) {
	if r := recover(); r != nil {
		// Create panic error
		panicErr := pr.createPanicError(r, operation, context)

		// Log the panic with stack trace
		pr.logPanic(panicErr)

		// Record metrics if enabled
		if pr.enableMetrics && pr.metricsFunc != nil {
			pr.recordPanicMetrics(panicErr)
		}

		// Determine if this is a critical panic requiring shutdown
		if pr.isCriticalPanic(panicErr) && pr.shutdownFunc != nil {
			pr.logger("Critical panic detected, initiating graceful shutdown: %v", r)
			pr.shutdownFunc()
		}
	}
}

// RecoverWithCallback handles panic recovery and executes a callback
func (pr *PanicRecovery) RecoverWithCallback(operation string, context map[string]interface{}, callback func(*GoCatError)) {
	if r := recover(); r != nil {
		panicErr := pr.createPanicError(r, operation, context)
		pr.logPanic(panicErr)

		if pr.enableMetrics && pr.metricsFunc != nil {
			pr.recordPanicMetrics(panicErr)
		}

		if callback != nil {
			callback(panicErr)
		}

		if pr.isCriticalPanic(panicErr) && pr.shutdownFunc != nil {
			pr.logger("Critical panic detected, initiating graceful shutdown: %v", r)
			pr.shutdownFunc()
		}
	}
}

// createPanicError creates a structured error from panic information
func (pr *PanicRecovery) createPanicError(panicValue interface{}, operation string, context map[string]interface{}) *GoCatError {
	panicErr := NewError(ErrorTypeSystem, SeverityCritical, "SYS004", fmt.Sprintf("Panic in operation: %s", operation))

	// Add panic value to context
	if context == nil {
		context = make(map[string]interface{})
	}
	context["panic_value"] = panicValue
	context["operation"] = operation
	context["recovery_time"] = time.Now()

	// Set all context
	for key, value := range context {
		_ = panicErr.WithContext(key, value)
	}

	_ = panicErr.WithUserFriendly("An unexpected error occurred. The system has recovered automatically.")
	_ = panicErr.WithSuggestion("If this error persists, please check the logs and contact support.")

	return panicErr
}

// logPanic logs panic information with stack trace
func (pr *PanicRecovery) logPanic(panicErr *GoCatError) {
	if pr.logger == nil {
		return
	}

	pr.logger("PANIC RECOVERED: %s", panicErr.Message)
	pr.logger("Panic Details: %+v", panicErr.Context())
	pr.logger("Stack Trace:")
	for i, frame := range panicErr.StackTrace {
		pr.logger("  %d: %s", i, frame)
	}
}

// recordPanicMetrics records panic metrics
func (pr *PanicRecovery) recordPanicMetrics(panicErr *GoCatError) {
	metrics := map[string]interface{}{
		"panic_type":    fmt.Sprintf("%T", panicErr.Context()["panic_value"]),
		"operation":     panicErr.Context()["operation"],
		"recovery_time": panicErr.Context()["recovery_time"],
		"error_code":    panicErr.Code,
		"stack_depth":   len(panicErr.StackTrace),
	}

	pr.metricsFunc("panic_recovered", metrics)
}

// isCriticalPanic determines if a panic requires system shutdown
func (pr *PanicRecovery) isCriticalPanic(panicErr *GoCatError) bool {
	panicValue := panicErr.Context()["panic_value"]

	// Check for critical panic types
	switch v := panicValue.(type) {
	case string:
		// Critical string patterns
		criticalPatterns := []string{
			"runtime error: out of memory",
			"fatal error:",
			"runtime: out of memory",
			"cannot allocate memory",
		}
		for _, pattern := range criticalPatterns {
			if strings.Contains(strings.ToLower(v), pattern) {
				return true
			}
		}
	case runtime.Error:
		// Runtime errors are generally critical
		return true
	}

	// Check operation context for critical operations
	if operation, ok := panicErr.Context()["operation"].(string); ok {
		criticalOperations := []string{
			"main",
			"server_start",
			"listener_init",
			"config_load",
		}
		for _, criticalOp := range criticalOperations {
			if strings.Contains(strings.ToLower(operation), criticalOp) {
				return true
			}
		}
	}

	return false
}

// SafeGo runs a goroutine with panic recovery
func (pr *PanicRecovery) SafeGo(operation string, fn func()) {
	go func() {
		defer pr.RecoverWithContext(operation, map[string]interface{}{
			"goroutine":  true,
			"started_at": time.Now(),
		})
		fn()
	}()
}

// SafeGoWithContext runs a goroutine with panic recovery and custom context
func (pr *PanicRecovery) SafeGoWithContext(operation string, context map[string]interface{}, fn func()) {
	go func() {
		defer pr.RecoverWithContext(operation, context)
		fn()
	}()
}

// DefaultPanicRecovery provides a default panic recovery instance
var DefaultPanicRecovery = NewPanicRecovery(PanicRecoveryConfig{
	Logger: func(format string, args ...interface{}) {
		fmt.Printf("[PANIC RECOVERY] "+format+"\n", args...)
	},
	EnableMetrics: false,
})

// Recover is a convenience function using the default panic recovery
func Recover(operation string) {
	DefaultPanicRecovery.RecoverWithContext(operation, nil)
}

// RecoverWithContext is a convenience function using the default panic recovery
func RecoverWithContext(operation string, context map[string]interface{}) {
	DefaultPanicRecovery.RecoverWithContext(operation, context)
}

// SafeGo is a convenience function using the default panic recovery
func SafeGo(operation string, fn func()) {
	DefaultPanicRecovery.SafeGo(operation, fn)
}
