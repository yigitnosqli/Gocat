// Package interfaces defines error handling interfaces following consistent patterns
package interfaces

import (
	"context"
	"time"
)

// Error represents an enhanced error with additional context and metadata
type Error interface {
	error

	// Type returns the error type
	Type() ErrorType

	// Severity returns the error severity level
	Severity() ErrorSeverity

	// Code returns a unique error code
	Code() string

	// UserFriendlyMessage returns a user-friendly error message
	UserFriendlyMessage() string

	// Suggestion returns a suggestion for resolving the error
	Suggestion() string

	// Cause returns the underlying cause of the error
	Cause() error

	// IsRetryable indicates if the operation can be retried
	IsRetryable() bool

	// Context returns additional context information
	Context() map[string]interface{}

	// WithContext adds context information to the error
	WithContext(key string, value interface{}) Error

	// WithSuggestion adds a suggestion to the error
	WithSuggestion(suggestion string) Error

	// WithCause sets the underlying cause
	WithCause(cause error) Error

	// SetRetryable sets whether the error is retryable
	SetRetryable(retryable bool) Error
}

// ErrorType represents different categories of errors
type ErrorType string

const (
	ErrorTypeValidation ErrorType = "validation"
	ErrorTypeNetwork    ErrorType = "network"
	ErrorTypeTimeout    ErrorType = "timeout"
	ErrorTypeSecurity   ErrorType = "security"
	ErrorTypeConfig     ErrorType = "config"
	ErrorTypeSystem     ErrorType = "system"
	ErrorTypeAuth       ErrorType = "authentication"
	ErrorTypePermission ErrorType = "permission"
)

// ErrorSeverity represents the severity level of errors
type ErrorSeverity string

const (
	SeverityLow      ErrorSeverity = "low"
	SeverityMedium   ErrorSeverity = "medium"
	SeverityHigh     ErrorSeverity = "high"
	SeverityCritical ErrorSeverity = "critical"
)

// ErrorFactory creates errors with consistent formatting and metadata
type ErrorFactory interface {
	// NewValidationError creates a validation error
	NewValidationError(code, message string) Error

	// NewNetworkError creates a network error
	NewNetworkError(code, message string) Error

	// NewTimeoutError creates a timeout error
	NewTimeoutError(code, message string) Error

	// NewSecurityError creates a security error
	NewSecurityError(code, message string) Error

	// NewConfigError creates a configuration error
	NewConfigError(code, message string) Error

	// NewSystemError creates a system error
	NewSystemError(code, message string) Error

	// WrapError wraps an existing error with additional context
	WrapError(err error, errorType ErrorType, severity ErrorSeverity, code, message string) Error
}

// PanicRecovery handles panic recovery with proper logging and metrics
type PanicRecovery interface {
	// RecoverWithContext recovers from a panic with context information
	RecoverWithContext(ctx context.Context, operation string) error

	// RecoverWithCallback recovers from a panic and calls a callback
	RecoverWithCallback(operation string, callback func(recovered interface{})) error

	// SetLogger sets the logger for panic recovery
	SetLogger(logger Logger)

	// SetMetrics sets the metrics collector for panic recovery
	SetMetrics(metrics MetricsCollector)
}

// Logger defines the interface for structured logging
type Logger interface {
	// Debug logs a debug message
	Debug(msg string, fields ...interface{})

	// Info logs an info message
	Info(msg string, fields ...interface{})

	// Warn logs a warning message
	Warn(msg string, fields ...interface{})

	// Error logs an error message
	Error(msg string, fields ...interface{})

	// Fatal logs a fatal message and exits
	Fatal(msg string, fields ...interface{})

	// WithFields returns a logger with additional fields
	WithFields(fields map[string]interface{}) Logger

	// WithContext returns a logger with context
	WithContext(ctx context.Context) Logger
}

// HealthChecker provides health checking capabilities
type HealthChecker interface {
	// CheckHealth performs a health check
	CheckHealth(ctx context.Context) HealthStatus

	// RegisterCheck registers a health check
	RegisterCheck(name string, check HealthCheck)

	// GetStatus returns the current health status
	GetStatus() map[string]HealthStatus
}

// HealthCheck represents a single health check
type HealthCheck interface {
	// Name returns the name of the health check
	Name() string

	// Check performs the health check
	Check(ctx context.Context) HealthStatus
}

// HealthStatus represents the result of a health check
type HealthStatus struct {
	Name      string                 `json:"name"`
	Status    string                 `json:"status"` // "healthy", "unhealthy", "unknown"
	Message   string                 `json:"message,omitempty"`
	Details   map[string]interface{} `json:"details,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
	Duration  time.Duration          `json:"duration"`
}

// ConfigManager handles application configuration
type ConfigManager interface {
	// Load loads configuration from various sources
	Load(sources ...ConfigSource) error

	// Get retrieves a configuration value
	Get(key string) interface{}

	// GetString retrieves a string configuration value
	GetString(key string) string

	// GetInt retrieves an integer configuration value
	GetInt(key string) int

	// GetBool retrieves a boolean configuration value
	GetBool(key string) bool

	// GetDuration retrieves a duration configuration value
	GetDuration(key string) time.Duration

	// Set sets a configuration value
	Set(key string, value interface{})

	// Validate validates the current configuration
	Validate() error

	// Watch watches for configuration changes
	Watch(callback func(key string, oldValue, newValue interface{}))
}

// ConfigSource represents a source of configuration data
type ConfigSource interface {
	// Name returns the name of the configuration source
	Name() string

	// Load loads configuration data
	Load() (map[string]interface{}, error)

	// Watch watches for changes in the configuration source
	Watch(callback func(map[string]interface{})) error
}
