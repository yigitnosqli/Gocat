package logger

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/fatih/color"
	"gopkg.in/yaml.v3"
)

// LogLevel represents different log levels
type LogLevel int

const (
	LevelInfo LogLevel = iota
	LevelWarn
	LevelError
)

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

// Logger represents a colored logger
type Logger struct {
	level LogLevel
	theme *ColorTheme
}

// NewLogger creates a new logger with the specified level
func NewLogger(level LogLevel) *Logger {
	return &Logger{level: level, theme: currentTheme}
}

// Info logs an info message
func (l *Logger) Info(format string, args ...interface{}) {
	if l.level <= LevelInfo {
		if _, err := l.theme.Info.Printf("info: "+format+"\n", args...); err != nil {
			fmt.Fprintf(os.Stderr, "Logger error: %v\n", err)
		}
	}
}

// Warn logs a warning message
func (l *Logger) Warn(format string, args ...interface{}) {
	if l.level <= LevelWarn {
		if _, err := l.theme.Warning.Printf("warn: "+format+"\n", args...); err != nil {
			fmt.Fprintf(os.Stderr, "Logger error: %v\n", err)
		}
	}
}

// Error logs an error message
func (l *Logger) Error(format string, args ...interface{}) {
	if l.level <= LevelError {
		if _, err := l.theme.Error.Printf("error: "+format+"\n", args...); err != nil {
			fmt.Fprintf(os.Stderr, "Logger error: %v\n", err)
		}
	}
}

// Fatal logs an error message and exits
func (l *Logger) Fatal(format string, args ...interface{}) {
	if _, err := l.theme.Error.Printf("error: "+format+"\n", args...); err != nil {
		fmt.Fprintf(os.Stderr, "Logger error: %v\n", err)
	}
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
