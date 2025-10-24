package logger

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/fatih/color"
	"gopkg.in/yaml.v3"
)

// LogLevel represents different log levels
type LogLevel int

const (
	LevelDebug LogLevel = iota
	LevelInfo
	LevelWarn
	LevelError
	LevelFatal
)

// String returns the string representation of the log level
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

// ThemeConfig represents color theme configuration
type ThemeConfig struct {
	Colors struct {
		Success   string `yaml:"success"`
		Error     string `yaml:"error"`
		Warning   string `yaml:"warning"`
		Info      string `yaml:"info"`
		Debug     string `yaml:"debug"`
		Highlight string `yaml:"highlight"`
	} `yaml:"colors"`
}

// ColorTheme holds the current color theme
type ColorTheme struct {
	Success   *color.Color
	Error     *color.Color
	Warning   *color.Color
	Info      *color.Color
	Debug     *color.Color
	Highlight *color.Color
}

// Default color theme
var defaultTheme = &ColorTheme{
	Success:   color.New(color.FgGreen),
	Error:     color.New(color.FgRed),
	Warning:   color.New(color.FgYellow),
	Info:      color.New(color.FgBlue),
	Debug:     color.New(color.FgWhite),
	Highlight: color.New(color.FgCyan),
}

// Current active theme
var currentTheme = defaultTheme

// LogEntry represents a structured log entry
type LogEntry struct {
	Timestamp time.Time              `json:"timestamp"`
	Level     string                 `json:"level"`
	Message   string                 `json:"message"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
	Caller    string                 `json:"caller,omitempty"`
}

// Logger represents a structured logger with color support
type Logger struct {
	level       LogLevel
	theme       *ColorTheme
	output      io.Writer
	errorOutput io.Writer
	mu          sync.RWMutex
	structured  bool
	showCaller  bool
}

// NewLogger creates a new logger with the specified level
func NewLogger(level LogLevel) *Logger {
	return &Logger{
		level:       level,
		theme:       currentTheme,
		output:      os.Stdout,
		errorOutput: os.Stderr,
		structured:  false,
		showCaller:  false,
	}
}

// SetOutput sets the output destination for the logger
func (l *Logger) SetOutput(w io.Writer) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.output = w
}

// SetErrorOutput sets the error output destination for the logger
func (l *Logger) SetErrorOutput(w io.Writer) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.errorOutput = w
}

// SetStructured enables or disables structured JSON logging
func (l *Logger) SetStructured(structured bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.structured = structured
}

// SetShowCaller enables or disables caller information in logs
func (l *Logger) SetShowCaller(show bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.showCaller = show
}

// getCaller returns the caller information
func (l *Logger) getCaller() string {
	if !l.showCaller {
		return ""
	}
	_, file, line, ok := runtime.Caller(3)
	if !ok {
		return "unknown"
	}
	// Get just the filename, not the full path
	parts := strings.Split(file, "/")
	filename := parts[len(parts)-1]
	return fmt.Sprintf("%s:%d", filename, line)
}

// log is the internal logging method
func (l *Logger) log(level LogLevel, message string, fields map[string]interface{}) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	if level < l.level {
		return
	}

	entry := LogEntry{
		Timestamp: time.Now(),
		Level:     level.String(),
		Message:   message,
		Fields:    fields,
		Caller:    l.getCaller(),
	}

	if l.structured {
		l.writeStructured(entry)
	} else {
		l.writeColored(entry, level)
	}
}

// writeStructured writes the log entry as JSON
func (l *Logger) writeStructured(entry LogEntry) {
	data, err := json.Marshal(entry)
	if err != nil {
		fmt.Fprintf(l.errorOutput, "Logger marshal error: %v\n", err)
		return
	}

	output := l.output
	if entry.Level == "ERROR" || entry.Level == "FATAL" {
		output = l.errorOutput
	}

	fmt.Fprintln(output, string(data))
}

// writeColored writes the log entry with colors
func (l *Logger) writeColored(entry LogEntry, level LogLevel) {
	output := l.output
	if level >= LevelError {
		output = l.errorOutput
	}

	timestamp := entry.Timestamp.Format("2006-01-02 15:04:05")
	message := fmt.Sprintf("[%s] %s: %s", timestamp, entry.Level, entry.Message)

	if entry.Caller != "" {
		message += fmt.Sprintf(" (%s)", entry.Caller)
	}

	if len(entry.Fields) > 0 {
		var fieldStrs []string
		for k, v := range entry.Fields {
			fieldStrs = append(fieldStrs, fmt.Sprintf("%s=%v", k, v))
		}
		message += fmt.Sprintf(" [%s]", strings.Join(fieldStrs, ", "))
	}

	var colorFunc func(io.Writer, string, ...interface{}) (int, error)
	switch level {
	case LevelDebug:
		colorFunc = l.theme.Debug.Fprintf
	case LevelInfo:
		colorFunc = l.theme.Info.Fprintf
	case LevelWarn:
		colorFunc = l.theme.Warning.Fprintf
	case LevelError, LevelFatal:
		colorFunc = l.theme.Error.Fprintf
	default:
		colorFunc = l.theme.Info.Fprintf
	}

	if _, err := colorFunc(output, "%s\n", message); err != nil {
		fmt.Fprintf(l.errorOutput, "Logger color error: %v\n", err)
	}
}

// Debug logs a debug message
func (l *Logger) Debug(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	l.log(LevelDebug, message, nil)
}

// DebugWithFields logs a debug message with additional fields
func (l *Logger) DebugWithFields(message string, fields map[string]interface{}) {
	l.log(LevelDebug, message, fields)
}

// Info logs an info message
func (l *Logger) Info(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	l.log(LevelInfo, message, nil)
}

// InfoWithFields logs an info message with additional fields
func (l *Logger) InfoWithFields(message string, fields map[string]interface{}) {
	l.log(LevelInfo, message, fields)
}

// Warn logs a warning message
func (l *Logger) Warn(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	l.log(LevelWarn, message, nil)
}

// WarnWithFields logs a warning message with additional fields
func (l *Logger) WarnWithFields(message string, fields map[string]interface{}) {
	l.log(LevelWarn, message, fields)
}

// Error logs an error message
func (l *Logger) Error(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	l.log(LevelError, message, nil)
}

// ErrorWithFields logs an error message with additional fields
func (l *Logger) ErrorWithFields(message string, fields map[string]interface{}) {
	l.log(LevelError, message, fields)
}

// Fatal logs an error message and exits
func (l *Logger) Fatal(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	l.log(LevelFatal, message, nil)
	os.Exit(1)
}

// FatalWithFields logs an error message with additional fields and exits
func (l *Logger) FatalWithFields(message string, fields map[string]interface{}) {
	l.log(LevelFatal, message, fields)
	os.Exit(1)
}

// Default logger instance
var defaultLogger = NewLogger(LevelInfo)

// Debug logs a debug message using the default logger
func Debug(format string, args ...interface{}) {
	defaultLogger.Debug(format, args...)
}

// DebugWithFields logs a debug message with fields using the default logger
func DebugWithFields(message string, fields map[string]interface{}) {
	defaultLogger.DebugWithFields(message, fields)
}

// Info logs an info message using the default logger
func Info(format string, args ...interface{}) {
	defaultLogger.Info(format, args...)
}

// InfoWithFields logs an info message with fields using the default logger
func InfoWithFields(message string, fields map[string]interface{}) {
	defaultLogger.InfoWithFields(message, fields)
}

// Warn logs a warning message using the default logger
func Warn(format string, args ...interface{}) {
	defaultLogger.Warn(format, args...)
}

// WarnWithFields logs a warning message with fields using the default logger
func WarnWithFields(message string, fields map[string]interface{}) {
	defaultLogger.WarnWithFields(message, fields)
}

// Error logs an error message using the default logger
func Error(format string, args ...interface{}) {
	defaultLogger.Error(format, args...)
}

// ErrorWithFields logs an error message with fields using the default logger
func ErrorWithFields(message string, fields map[string]interface{}) {
	defaultLogger.ErrorWithFields(message, fields)
}

// Fatal logs an error message and exits using the default logger
func Fatal(format string, args ...interface{}) {
	defaultLogger.Fatal(format, args...)
}

// FatalWithFields logs an error message with fields and exits using the default logger
func FatalWithFields(message string, fields map[string]interface{}) {
	defaultLogger.FatalWithFields(message, fields)
}

// SetLevel sets the log level for the default logger
func SetLevel(level LogLevel) {
	defaultLogger.mu.Lock()
	defer defaultLogger.mu.Unlock()
	defaultLogger.level = level
}

// SetStructured enables structured logging for the default logger
func SetStructured(structured bool) {
	defaultLogger.SetStructured(structured)
}

// SetShowCaller enables caller information for the default logger
func SetShowCaller(show bool) {
	defaultLogger.SetShowCaller(show)
}

// GetDefaultLogger returns the default logger instance
func GetDefaultLogger() *Logger {
	return defaultLogger
}

// SetupLogger configures the standard log package
func SetupLogger() {
	// Disable standard log prefixes since we're using colored output
	log.SetFlags(0)
}

// LoadTheme loads color theme from file
func LoadTheme(themePath string) error {
	if themePath == "" {
		// Try default locations
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil // Silently fail, use default theme
		}
		themePath = filepath.Join(homeDir, ".gocat-theme.yml")
	}

	// Check if file exists
	if _, err := os.Stat(themePath); os.IsNotExist(err) {
		return nil // File doesn't exist, use default theme
	}

	data, err := os.ReadFile(themePath)
	if err != nil {
		return fmt.Errorf("failed to read theme file: %w", err)
	}

	var config ThemeConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to parse theme file: %w", err)
	}

	// Create new theme from config
	newTheme := &ColorTheme{
		Success:   parseColor(config.Colors.Success, defaultTheme.Success),
		Error:     parseColor(config.Colors.Error, defaultTheme.Error),
		Warning:   parseColor(config.Colors.Warning, defaultTheme.Warning),
		Info:      parseColor(config.Colors.Info, defaultTheme.Info),
		Debug:     parseColor(config.Colors.Debug, defaultTheme.Debug),
		Highlight: parseColor(config.Colors.Highlight, defaultTheme.Highlight),
	}

	// Update current theme
	currentTheme = newTheme
	defaultLogger.theme = currentTheme

	return nil
}

// parseColor converts string color name to color.Color
func parseColor(colorName string, fallback *color.Color) *color.Color {
	if colorName == "" {
		return fallback
	}

	switch colorName {
	case "black":
		return color.New(color.FgBlack)
	case "red":
		return color.New(color.FgRed)
	case "green":
		return color.New(color.FgGreen)
	case "yellow":
		return color.New(color.FgYellow)
	case "blue":
		return color.New(color.FgBlue)
	case "magenta":
		return color.New(color.FgMagenta)
	case "cyan":
		return color.New(color.FgCyan)
	case "white":
		return color.New(color.FgWhite)
	case "gray", "grey":
		return color.New(color.FgHiBlack)
	case "bright_red":
		return color.New(color.FgHiRed)
	case "bright_green":
		return color.New(color.FgHiGreen)
	case "bright_yellow":
		return color.New(color.FgHiYellow)
	case "bright_blue":
		return color.New(color.FgHiBlue)
	case "bright_magenta":
		return color.New(color.FgHiMagenta)
	case "bright_cyan":
		return color.New(color.FgHiCyan)
	case "bright_white":
		return color.New(color.FgHiWhite)
	default:
		return fallback
	}
}

// SetTheme sets a custom theme
func SetTheme(theme *ColorTheme) {
	currentTheme = theme
	defaultLogger.theme = currentTheme
}

// GetCurrentTheme returns the current active theme
func GetCurrentTheme() *ColorTheme {
	return currentTheme
}

// SetOutput sets the output destination for the default logger
func SetOutput(w io.Writer) {
	defaultLogger.SetOutput(w)
}
