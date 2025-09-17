package logging

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"
)

// LogLevel represents the severity level of a log entry
type LogLevel int

const (
	LevelDebug LogLevel = iota
	LevelInfo
	LevelWarn
	LevelError
	LevelFatal
)

// String returns the string representation of LogLevel
func (l LogLevel) String() string {
	switch l {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	case LevelFatal:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

// ParseLogLevel parses a string into a LogLevel
func ParseLogLevel(level string) (LogLevel, error) {
	switch strings.ToUpper(level) {
	case "DEBUG":
		return LevelDebug, nil
	case "INFO":
		return LevelInfo, nil
	case "WARN", "WARNING":
		return LevelWarn, nil
	case "ERROR":
		return LevelError, nil
	case "FATAL":
		return LevelFatal, nil
	default:
		return LevelInfo, fmt.Errorf("unknown log level: %s", level)
	}
}

// LogEntry represents a single log entry
type LogEntry struct {
	Timestamp time.Time              `json:"timestamp"`
	Level     LogLevel               `json:"level"`
	Message   string                 `json:"message"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
	Caller    string                 `json:"caller,omitempty"`
	TraceID   string                 `json:"trace_id,omitempty"`
	Component string                 `json:"component,omitempty"`
	Error     string                 `json:"error,omitempty"`
}

// Logger defines the logging interface
type Logger interface {
	// Level methods
	Debug(msg string, fields ...Field)
	Info(msg string, fields ...Field)
	Warn(msg string, fields ...Field)
	Error(msg string, fields ...Field)
	Fatal(msg string, fields ...Field)

	// Formatted methods
	Debugf(format string, args ...interface{})
	Infof(format string, args ...interface{})
	Warnf(format string, args ...interface{})
	Errorf(format string, args ...interface{})
	Fatalf(format string, args ...interface{})

	// Context methods
	WithContext(ctx context.Context) Logger
	WithFields(fields ...Field) Logger
	WithComponent(component string) Logger
	WithError(err error) Logger

	// Configuration
	SetLevel(level LogLevel)
	GetLevel() LogLevel
	SetOutput(w io.Writer)
}

// Field represents a key-value pair for structured logging
type Field struct {
	Key   string
	Value interface{}
}

// String creates a string field
func String(key, value string) Field {
	return Field{Key: key, Value: value}
}

// Int creates an integer field
func Int(key string, value int) Field {
	return Field{Key: key, Value: value}
}

// Int64 creates an int64 field
func Int64(key string, value int64) Field {
	return Field{Key: key, Value: value}
}

// Float64 creates a float64 field
func Float64(key string, value float64) Field {
	return Field{Key: key, Value: value}
}

// Bool creates a boolean field
func Bool(key string, value bool) Field {
	return Field{Key: key, Value: value}
}

// Duration creates a duration field
func Duration(key string, value time.Duration) Field {
	return Field{Key: key, Value: value.String()}
}

// Time creates a time field
func Time(key string, value time.Time) Field {
	return Field{Key: key, Value: value.Format(time.RFC3339)}
}

// Any creates a field with any value
func Any(key string, value interface{}) Field {
	return Field{Key: key, Value: value}
}

// Error creates an error field
func Error(err error) Field {
	if err == nil {
		return Field{Key: "error", Value: nil}
	}
	return Field{Key: "error", Value: err.Error()}
}

// StructuredLogger is the main logger implementation
type StructuredLogger struct {
	level     LogLevel
	output    io.Writer
	formatter Formatter
	mu        sync.RWMutex

	// Context fields
	fields    map[string]interface{}
	component string
	traceID   string

	// Configuration
	enableCaller bool
	callerSkip   int
}

// Formatter defines how log entries are formatted
type Formatter interface {
	Format(entry LogEntry) ([]byte, error)
}

// JSONFormatter formats log entries as JSON
type JSONFormatter struct {
	PrettyPrint bool
}

// Format formats a log entry as JSON
func (jf *JSONFormatter) Format(entry LogEntry) ([]byte, error) {
	if jf.PrettyPrint {
		return json.MarshalIndent(entry, "", "  ")
	}
	return json.Marshal(entry)
}

// TextFormatter formats log entries as human-readable text
type TextFormatter struct {
	DisableColors bool
	FullTimestamp bool
}

// Format formats a log entry as text
func (tf *TextFormatter) Format(entry LogEntry) ([]byte, error) {
	var buf strings.Builder

	// Timestamp
	if tf.FullTimestamp {
		buf.WriteString(entry.Timestamp.Format("2006-01-02 15:04:05.000"))
	} else {
		buf.WriteString(entry.Timestamp.Format("15:04:05"))
	}

	// Level with colors
	levelStr := entry.Level.String()
	if !tf.DisableColors {
		switch entry.Level {
		case LevelDebug:
			levelStr = fmt.Sprintf("\033[36m%s\033[0m", levelStr) // Cyan
		case LevelInfo:
			levelStr = fmt.Sprintf("\033[32m%s\033[0m", levelStr) // Green
		case LevelWarn:
			levelStr = fmt.Sprintf("\033[33m%s\033[0m", levelStr) // Yellow
		case LevelError:
			levelStr = fmt.Sprintf("\033[31m%s\033[0m", levelStr) // Red
		case LevelFatal:
			levelStr = fmt.Sprintf("\033[35m%s\033[0m", levelStr) // Magenta
		}
	}

	buf.WriteString(fmt.Sprintf(" [%s]", levelStr))

	// Component
	if entry.Component != "" {
		buf.WriteString(fmt.Sprintf(" [%s]", entry.Component))
	}

	// Caller
	if entry.Caller != "" {
		buf.WriteString(fmt.Sprintf(" %s", entry.Caller))
	}

	// Message
	buf.WriteString(fmt.Sprintf(" %s", entry.Message))

	// Fields
	if len(entry.Fields) > 0 {
		buf.WriteString(" |")
		for k, v := range entry.Fields {
			buf.WriteString(fmt.Sprintf(" %s=%v", k, v))
		}
	}

	// Error
	if entry.Error != "" {
		buf.WriteString(fmt.Sprintf(" error=%s", entry.Error))
	}

	// Trace ID
	if entry.TraceID != "" {
		buf.WriteString(fmt.Sprintf(" trace_id=%s", entry.TraceID))
	}

	buf.WriteString("\n")
	return []byte(buf.String()), nil
}

// NewStructuredLogger creates a new structured logger
func NewStructuredLogger(level LogLevel, output io.Writer, formatter Formatter) *StructuredLogger {
	if output == nil {
		output = os.Stdout
	}

	if formatter == nil {
		formatter = &JSONFormatter{}
	}

	return &StructuredLogger{
		level:        level,
		output:       output,
		formatter:    formatter,
		fields:       make(map[string]interface{}),
		enableCaller: true,
		callerSkip:   2,
	}
}

// NewJSONLogger creates a new JSON logger
func NewJSONLogger(level LogLevel, output io.Writer) *StructuredLogger {
	return NewStructuredLogger(level, output, &JSONFormatter{})
}

// NewTextLogger creates a new text logger
func NewTextLogger(level LogLevel, output io.Writer) *StructuredLogger {
	return NewStructuredLogger(level, output, &TextFormatter{})
}

// log is the internal logging method
func (sl *StructuredLogger) log(level LogLevel, msg string, fields ...Field) {
	sl.mu.RLock()
	currentLevel := sl.level
	sl.mu.RUnlock()

	if level < currentLevel {
		return
	}

	entry := LogEntry{
		Timestamp: time.Now(),
		Level:     level,
		Message:   msg,
		Fields:    make(map[string]interface{}),
		Component: sl.component,
		TraceID:   sl.traceID,
	}

	// Copy existing fields
	for k, v := range sl.fields {
		entry.Fields[k] = v
	}

	// Add new fields
	for _, field := range fields {
		if field.Key == "error" && field.Value != nil {
			entry.Error = fmt.Sprintf("%v", field.Value)
		} else {
			entry.Fields[field.Key] = field.Value
		}
	}

	// Add caller information
	if sl.enableCaller {
		if pc, file, line, ok := runtime.Caller(sl.callerSkip); ok {
			funcName := runtime.FuncForPC(pc).Name()
			// Shorten file path
			if idx := strings.LastIndex(file, "/"); idx != -1 {
				file = file[idx+1:]
			}
			entry.Caller = fmt.Sprintf("%s:%d %s", file, line, funcName)
		}
	}

	// Format and write
	formatted, err := sl.formatter.Format(entry)
	if err != nil {
		// Fallback to simple format if formatting fails
		formatted = []byte(fmt.Sprintf("%s [%s] %s\n",
			entry.Timestamp.Format(time.RFC3339),
			entry.Level.String(),
			entry.Message))
	}

	sl.mu.RLock()
	sl.output.Write(formatted)
	sl.mu.RUnlock()

	// Exit on fatal
	if level == LevelFatal {
		os.Exit(1)
	}
}

// Debug logs a debug message
func (sl *StructuredLogger) Debug(msg string, fields ...Field) {
	sl.log(LevelDebug, msg, fields...)
}

// Info logs an info message
func (sl *StructuredLogger) Info(msg string, fields ...Field) {
	sl.log(LevelInfo, msg, fields...)
}

// Warn logs a warning message
func (sl *StructuredLogger) Warn(msg string, fields ...Field) {
	sl.log(LevelWarn, msg, fields...)
}

// Error logs an error message
func (sl *StructuredLogger) Error(msg string, fields ...Field) {
	sl.log(LevelError, msg, fields...)
}

// Fatal logs a fatal message and exits
func (sl *StructuredLogger) Fatal(msg string, fields ...Field) {
	sl.log(LevelFatal, msg, fields...)
}

// Debugf logs a formatted debug message
func (sl *StructuredLogger) Debugf(format string, args ...interface{}) {
	sl.log(LevelDebug, fmt.Sprintf(format, args...))
}

// Infof logs a formatted info message
func (sl *StructuredLogger) Infof(format string, args ...interface{}) {
	sl.log(LevelInfo, fmt.Sprintf(format, args...))
}

// Warnf logs a formatted warning message
func (sl *StructuredLogger) Warnf(format string, args ...interface{}) {
	sl.log(LevelWarn, fmt.Sprintf(format, args...))
}

// Errorf logs a formatted error message
func (sl *StructuredLogger) Errorf(format string, args ...interface{}) {
	sl.log(LevelError, fmt.Sprintf(format, args...))
}

// Fatalf logs a formatted fatal message and exits
func (sl *StructuredLogger) Fatalf(format string, args ...interface{}) {
	sl.log(LevelFatal, fmt.Sprintf(format, args...))
}

// WithContext returns a logger with context information
func (sl *StructuredLogger) WithContext(ctx context.Context) Logger {
	newLogger := sl.clone()

	// Extract trace ID from context if available
	if traceID := ctx.Value("trace_id"); traceID != nil {
		if id, ok := traceID.(string); ok {
			newLogger.traceID = id
		}
	}

	return newLogger
}

// WithFields returns a logger with additional fields
func (sl *StructuredLogger) WithFields(fields ...Field) Logger {
	newLogger := sl.clone()

	for _, field := range fields {
		newLogger.fields[field.Key] = field.Value
	}

	return newLogger
}

// WithComponent returns a logger with a component name
func (sl *StructuredLogger) WithComponent(component string) Logger {
	newLogger := sl.clone()
	newLogger.component = component
	return newLogger
}

// WithError returns a logger with an error field
func (sl *StructuredLogger) WithError(err error) Logger {
	if err == nil {
		return sl
	}

	return sl.WithFields(Error(err))
}

// SetLevel sets the logging level
func (sl *StructuredLogger) SetLevel(level LogLevel) {
	sl.mu.Lock()
	defer sl.mu.Unlock()
	sl.level = level
}

// GetLevel returns the current logging level
func (sl *StructuredLogger) GetLevel() LogLevel {
	sl.mu.RLock()
	defer sl.mu.RUnlock()
	return sl.level
}

// SetOutput sets the output writer
func (sl *StructuredLogger) SetOutput(w io.Writer) {
	sl.mu.Lock()
	defer sl.mu.Unlock()
	sl.output = w
}

// EnableCaller enables/disables caller information
func (sl *StructuredLogger) EnableCaller(enable bool) {
	sl.mu.Lock()
	defer sl.mu.Unlock()
	sl.enableCaller = enable
}

// SetCallerSkip sets the number of stack frames to skip when determining caller
func (sl *StructuredLogger) SetCallerSkip(skip int) {
	sl.mu.Lock()
	defer sl.mu.Unlock()
	sl.callerSkip = skip
}

// clone creates a copy of the logger
func (sl *StructuredLogger) clone() *StructuredLogger {
	sl.mu.RLock()
	defer sl.mu.RUnlock()

	fields := make(map[string]interface{})
	for k, v := range sl.fields {
		fields[k] = v
	}

	return &StructuredLogger{
		level:        sl.level,
		output:       sl.output,
		formatter:    sl.formatter,
		fields:       fields,
		component:    sl.component,
		traceID:      sl.traceID,
		enableCaller: sl.enableCaller,
		callerSkip:   sl.callerSkip,
	}
}

// LogRotator handles log rotation
type LogRotator struct {
	filename   string
	maxSize    int64 // Maximum size in bytes
	maxAge     int   // Maximum age in days
	maxBackups int   // Maximum number of backup files
	compress   bool  // Whether to compress rotated files

	mu          sync.Mutex
	currentFile *os.File
	currentSize int64
}

// NewLogRotator creates a new log rotator
func NewLogRotator(filename string, maxSize int64, maxAge, maxBackups int, compress bool) *LogRotator {
	return &LogRotator{
		filename:   filename,
		maxSize:    maxSize,
		maxAge:     maxAge,
		maxBackups: maxBackups,
		compress:   compress,
	}
}

// Write implements io.Writer
func (lr *LogRotator) Write(p []byte) (n int, err error) {
	lr.mu.Lock()
	defer lr.mu.Unlock()

	// Open file if not already open
	if lr.currentFile == nil {
		if err := lr.openFile(); err != nil {
			return 0, err
		}
	}

	// Check if rotation is needed
	if lr.currentSize+int64(len(p)) > lr.maxSize {
		if err := lr.rotate(); err != nil {
			return 0, err
		}
	}

	// Write to current file
	n, err = lr.currentFile.Write(p)
	lr.currentSize += int64(n)

	return n, err
}

// openFile opens the log file
func (lr *LogRotator) openFile() error {
	file, err := os.OpenFile(lr.filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}

	// Get current file size
	info, err := file.Stat()
	if err != nil {
		file.Close()
		return err
	}

	lr.currentFile = file
	lr.currentSize = info.Size()

	return nil
}

// rotate rotates the log file
func (lr *LogRotator) rotate() error {
	// Close current file
	if lr.currentFile != nil {
		lr.currentFile.Close()
		lr.currentFile = nil
	}

	// Rotate existing files
	for i := lr.maxBackups; i > 0; i-- {
		oldName := fmt.Sprintf("%s.%d", lr.filename, i)
		newName := fmt.Sprintf("%s.%d", lr.filename, i+1)

		if i == lr.maxBackups {
			os.Remove(oldName) // Remove oldest file
		} else {
			os.Rename(oldName, newName)
		}
	}

	// Move current file to .1
	backupName := fmt.Sprintf("%s.1", lr.filename)
	os.Rename(lr.filename, backupName)

	// Compress if enabled
	if lr.compress {
		go lr.compressFile(backupName)
	}

	// Clean up old files
	go lr.cleanup()

	// Open new file
	return lr.openFile()
}

// compressFile compresses a log file (simplified implementation)
func (lr *LogRotator) compressFile(filename string) {
	// This is a placeholder - in a real implementation,
	// you would use gzip or another compression library
}

// cleanup removes old log files based on age
func (lr *LogRotator) cleanup() {
	// This is a placeholder - in a real implementation,
	// you would check file ages and remove old files
}

// Close closes the log rotator
func (lr *LogRotator) Close() error {
	lr.mu.Lock()
	defer lr.mu.Unlock()

	if lr.currentFile != nil {
		return lr.currentFile.Close()
	}

	return nil
}

// Global logger instance
var globalLogger Logger = NewJSONLogger(LevelInfo, os.Stdout)

// Global functions

// SetGlobalLogger sets the global logger
func SetGlobalLogger(logger Logger) {
	globalLogger = logger
}

// GetGlobalLogger returns the global logger
func GetGlobalLogger() Logger {
	return globalLogger
}

// Debug logs a debug message using the global logger
func Debug(msg string, fields ...Field) {
	globalLogger.Debug(msg, fields...)
}

// Info logs an info message using the global logger
func Info(msg string, fields ...Field) {
	globalLogger.Info(msg, fields...)
}

// Warn logs a warning message using the global logger
func Warn(msg string, fields ...Field) {
	globalLogger.Warn(msg, fields...)
}

// Error logs an error message using the global logger
func ErrorLog(msg string, fields ...Field) {
	globalLogger.Error(msg, fields...)
}

// Fatal logs a fatal message using the global logger
func Fatal(msg string, fields ...Field) {
	globalLogger.Fatal(msg, fields...)
}

// Debugf logs a formatted debug message using the global logger
func Debugf(format string, args ...interface{}) {
	globalLogger.Debugf(format, args...)
}

// Infof logs a formatted info message using the global logger
func Infof(format string, args ...interface{}) {
	globalLogger.Infof(format, args...)
}

// Warnf logs a formatted warning message using the global logger
func Warnf(format string, args ...interface{}) {
	globalLogger.Warnf(format, args...)
}

// Errorf logs a formatted error message using the global logger
func Errorf(format string, args ...interface{}) {
	globalLogger.Errorf(format, args...)
}

// Fatalf logs a formatted fatal message using the global logger
func Fatalf(format string, args ...interface{}) {
	globalLogger.Fatalf(format, args...)
}

// WithComponent returns a logger with a component name using the global logger
func WithComponent(component string) Logger {
	return globalLogger.WithComponent(component)
}

// WithFields returns a logger with additional fields using the global logger
func WithFields(fields ...Field) Logger {
	return globalLogger.WithFields(fields...)
}

// WithError returns a logger with an error field using the global logger
func WithError(err error) Logger {
	return globalLogger.WithError(err)
}
