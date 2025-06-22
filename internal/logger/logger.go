package logger

import (
	"log"
	"os"

	"github.com/fatih/color"
)

// LogLevel represents different log levels
type LogLevel int

const (
	LevelInfo LogLevel = iota
	LevelWarn
	LevelError
)

// Logger represents a colored logger
type Logger struct {
	level LogLevel
}

// NewLogger creates a new logger with the specified level
func NewLogger(level LogLevel) *Logger {
	return &Logger{level: level}
}

// Info logs an info message
func (l *Logger) Info(format string, args ...interface{}) {
	if l.level <= LevelInfo {
		color.Green("info: "+format, args...)
	}
}

// Warn logs a warning message
func (l *Logger) Warn(format string, args ...interface{}) {
	if l.level <= LevelWarn {
		color.Yellow("warn: "+format, args...)
	}
}

// Error logs an error message
func (l *Logger) Error(format string, args ...interface{}) {
	if l.level <= LevelError {
		color.Red("error: "+format, args...)
	}
}

// Fatal logs an error message and exits
func (l *Logger) Fatal(format string, args ...interface{}) {
	color.Red("error: "+format, args...)
	os.Exit(1)
}

// Default logger instance
var defaultLogger = NewLogger(LevelInfo)

// Info logs an info message using the default logger
func Info(format string, args ...interface{}) {
	defaultLogger.Info(format, args...)
}

// Warn logs a warning message using the default logger
func Warn(format string, args ...interface{}) {
	defaultLogger.Warn(format, args...)
}

// Error logs an error message using the default logger
func Error(format string, args ...interface{}) {
	defaultLogger.Error(format, args...)
}

// Fatal logs an error message and exits using the default logger
func Fatal(format string, args ...interface{}) {
	defaultLogger.Fatal(format, args...)
}

// SetLevel sets the log level for the default logger
func SetLevel(level LogLevel) {
	defaultLogger.level = level
}

// SetupLogger configures the standard log package
func SetupLogger() {
	// Disable standard log prefixes since we're using colored output
	log.SetFlags(0)
}
