package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Config represents the application configuration
type Config struct {
	Network  NetworkConfig  `yaml:"network" json:"network"`
	Logger   LoggerConfig   `yaml:"logger" json:"logger"`
	UI       UIConfig       `yaml:"ui" json:"ui"`
	Security SecurityConfig `yaml:"security" json:"security"`
}

// NetworkConfig holds network-related configuration
type NetworkConfig struct {
	DefaultTimeout time.Duration `yaml:"default_timeout" json:"default_timeout"`
	KeepAlive      time.Duration `yaml:"keep_alive" json:"keep_alive"`
	MaxConnections int           `yaml:"max_connections" json:"max_connections"`
	BufferSize     int           `yaml:"buffer_size" json:"buffer_size"`
	RetryAttempts  int           `yaml:"retry_attempts" json:"retry_attempts"`
	RetryDelay     time.Duration `yaml:"retry_delay" json:"retry_delay"`
	BindAddress    string        `yaml:"bind_address" json:"bind_address"`
	BindPort       int           `yaml:"bind_port" json:"bind_port"`
	ReuseAddr      bool          `yaml:"reuse_addr" json:"reuse_addr"`
	ReusePort      bool          `yaml:"reuse_port" json:"reuse_port"`
	NoDelay        bool          `yaml:"no_delay" json:"no_delay"`
	IPv6           bool          `yaml:"ipv6" json:"ipv6"`
	IPv4           bool          `yaml:"ipv4" json:"ipv4"`
}

// LoggerConfig holds logging configuration
type LoggerConfig struct {
	Level      string `yaml:"level" json:"level"`
	Format     string `yaml:"format" json:"format"` // "json" or "text"
	Output     string `yaml:"output" json:"output"` // "stdout", "stderr", or file path
	ShowCaller bool   `yaml:"show_caller" json:"show_caller"`
	Colorize   bool   `yaml:"colorize" json:"colorize"`
}

// UIConfig holds UI-related configuration
type UIConfig struct {
	Theme       string `yaml:"theme" json:"theme"`
	ColorScheme string `yaml:"color_scheme" json:"color_scheme"`
	Animations  bool   `yaml:"animations" json:"animations"`
	RefreshRate int    `yaml:"refresh_rate" json:"refresh_rate"` // milliseconds
}

// SecurityConfig holds security-related configuration
type SecurityConfig struct {
	MaxHostnameLength int             `yaml:"max_hostname_length" json:"max_hostname_length"`
	AllowedProtocols  []string        `yaml:"allowed_protocols" json:"allowed_protocols"`
	RateLimit         RateLimitConfig `yaml:"rate_limit" json:"rate_limit"`
	TLSConfig         TLSConfig       `yaml:"tls" json:"tls"`
}

// RateLimitConfig holds rate limiting configuration
type RateLimitConfig struct {
	Enabled     bool          `yaml:"enabled" json:"enabled"`
	MaxRequests int           `yaml:"max_requests" json:"max_requests"`
	Window      time.Duration `yaml:"window" json:"window"`
}

// TLSConfig holds TLS configuration
type TLSConfig struct {
	Enabled            bool     `yaml:"enabled" json:"enabled"`
	CertFile           string   `yaml:"cert_file" json:"cert_file"`
	KeyFile            string   `yaml:"key_file" json:"key_file"`
	CAFile             string   `yaml:"ca_file" json:"ca_file"`
	InsecureSkipVerify bool     `yaml:"insecure_skip_verify" json:"insecure_skip_verify"`
	MinVersion         string   `yaml:"min_version" json:"min_version"`
	CipherSuites       []string `yaml:"cipher_suites" json:"cipher_suites"`
}

// Validate validates the TLS configuration
func (t *TLSConfig) Validate() error {
	if t.Enabled {
		if t.CertFile == "" {
			return fmt.Errorf("TLS is enabled but cert_file is not provided")
		}
		if t.KeyFile == "" {
			return fmt.Errorf("TLS is enabled but key_file is not provided")
		}
	}
	return nil
}

// DefaultConfig returns a configuration with default values
func DefaultConfig() *Config {
	return &Config{
		Network: NetworkConfig{
			DefaultTimeout: 30 * time.Second,
			KeepAlive:      30 * time.Second,
			MaxConnections: 100,
			BufferSize:     4096,
			RetryAttempts:  3,
			RetryDelay:     1 * time.Second,
			BindAddress:    "",
			BindPort:       0,
			ReuseAddr:      true,
			ReusePort:      false,
			NoDelay:        true,
			IPv6:           true,
			IPv4:           true,
		},
		Logger: LoggerConfig{
			Level:      "info",
			Format:     "text",
			Output:     "stdout",
			ShowCaller: false,
			Colorize:   true,
		},
		UI: UIConfig{
			Theme:       "default",
			ColorScheme: "auto",
			Animations:  true,
			RefreshRate: 100,
		},
		Security: SecurityConfig{
			MaxHostnameLength: 253,
			AllowedProtocols:  []string{"tcp", "udp", "tls"},
			RateLimit: RateLimitConfig{
				Enabled:     true,
				MaxRequests: 100,
				Window:      1 * time.Minute,
			},
			TLSConfig: TLSConfig{
				Enabled:            false,
				InsecureSkipVerify: false,
				MinVersion:         "1.2",
			},
		},
	}
}

// LoadConfig loads configuration from file and environment variables
func LoadConfig(configPath string) (*Config, error) {
	config := DefaultConfig()

	// Load from file if provided
	if configPath != "" {
		if err := loadFromFile(config, configPath); err != nil {
			return nil, fmt.Errorf("failed to load config from file: %w", err)
		}
	}

	// Override with environment variables
	loadFromEnv(config)

	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return config, nil
}

// loadFromFile loads configuration from a YAML file
func loadFromFile(config *Config, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	return yaml.Unmarshal(data, config)
}

// loadFromEnv loads configuration from environment variables
func loadFromEnv(config *Config) {
	// Network configuration
	if val := os.Getenv("GOCAT_NETWORK_TIMEOUT"); val != "" {
		if duration, err := time.ParseDuration(val); err == nil {
			config.Network.DefaultTimeout = duration
		}
	}
	if val := os.Getenv("GOCAT_NETWORK_KEEPALIVE"); val != "" {
		if duration, err := time.ParseDuration(val); err == nil {
			config.Network.KeepAlive = duration
		}
	}
	if val := os.Getenv("GOCAT_NETWORK_MAX_CONNECTIONS"); val != "" {
		if num, err := strconv.Atoi(val); err == nil {
			config.Network.MaxConnections = num
		}
	}
	if val := os.Getenv("GOCAT_NETWORK_BUFFER_SIZE"); val != "" {
		if num, err := strconv.Atoi(val); err == nil {
			config.Network.BufferSize = num
		}
	}
	if val := os.Getenv("GOCAT_NETWORK_BIND_ADDRESS"); val != "" {
		config.Network.BindAddress = val
	}
	if val := os.Getenv("GOCAT_NETWORK_BIND_PORT"); val != "" {
		if num, err := strconv.Atoi(val); err == nil {
			config.Network.BindPort = num
		}
	}

	// Logger configuration
	if val := os.Getenv("GOCAT_LOG_LEVEL"); val != "" {
		config.Logger.Level = strings.ToLower(val)
	}
	if val := os.Getenv("GOCAT_LOG_FORMAT"); val != "" {
		config.Logger.Format = strings.ToLower(val)
	}
	if val := os.Getenv("GOCAT_LOG_OUTPUT"); val != "" {
		config.Logger.Output = val
	}
	if val := os.Getenv("GOCAT_LOG_SHOW_CALLER"); val != "" {
		config.Logger.ShowCaller = strings.ToLower(val) == "true"
	}
	if val := os.Getenv("GOCAT_LOG_COLORIZE"); val != "" {
		config.Logger.Colorize = strings.ToLower(val) == "true"
	}

	// UI configuration
	if val := os.Getenv("GOCAT_UI_THEME"); val != "" {
		config.UI.Theme = val
	}
	if val := os.Getenv("GOCAT_UI_COLOR_SCHEME"); val != "" {
		config.UI.ColorScheme = val
	}
	if val := os.Getenv("GOCAT_UI_ANIMATIONS"); val != "" {
		config.UI.Animations = strings.ToLower(val) == "true"
	}

	// Security configuration
	if val := os.Getenv("GOCAT_SECURITY_RATE_LIMIT_ENABLED"); val != "" {
		config.Security.RateLimit.Enabled = strings.ToLower(val) == "true"
	}
	if val := os.Getenv("GOCAT_SECURITY_RATE_LIMIT_MAX_REQUESTS"); val != "" {
		if num, err := strconv.Atoi(val); err == nil {
			config.Security.RateLimit.MaxRequests = num
		}
	}
	if val := os.Getenv("GOCAT_TLS_ENABLED"); val != "" {
		config.Security.TLSConfig.Enabled = strings.ToLower(val) == "true"
	}
	if val := os.Getenv("GOCAT_TLS_CERT_FILE"); val != "" {
		config.Security.TLSConfig.CertFile = val
	}
	if val := os.Getenv("GOCAT_TLS_KEY_FILE"); val != "" {
		config.Security.TLSConfig.KeyFile = val
	}
}

// Validate validates the configuration
func (c *Config) Validate() error {
	// Validate network configuration
	if c.Network.DefaultTimeout <= 0 {
		return fmt.Errorf("network.default_timeout must be positive")
	}
	if c.Network.MaxConnections <= 0 {
		return fmt.Errorf("network.max_connections must be positive")
	}
	if c.Network.BufferSize <= 0 {
		return fmt.Errorf("network.buffer_size must be positive")
	}
	if c.Network.RetryAttempts < 0 {
		return fmt.Errorf("network.retry_attempts must be non-negative")
	}
	if c.Network.BindPort < 0 || c.Network.BindPort > 65535 {
		return fmt.Errorf("network.bind_port must be between 0 and 65535")
	}

	// Validate logger configuration
	validLogLevels := []string{"debug", "info", "warn", "error", "fatal"}
	if !contains(validLogLevels, c.Logger.Level) {
		return fmt.Errorf("logger.level must be one of: %v", validLogLevels)
	}
	validFormats := []string{"json", "text"}
	if !contains(validFormats, c.Logger.Format) {
		return fmt.Errorf("logger.format must be one of: %v", validFormats)
	}

	// Validate UI configuration
	if c.UI.RefreshRate <= 0 {
		return fmt.Errorf("ui.refresh_rate must be positive")
	}

	// Validate security configuration
	if c.Security.MaxHostnameLength <= 0 {
		return fmt.Errorf("security.max_hostname_length must be positive")
	}
	if len(c.Security.AllowedProtocols) == 0 {
		return fmt.Errorf("security.allowed_protocols cannot be empty")
	}
	if c.Security.RateLimit.Enabled {
		if c.Security.RateLimit.MaxRequests <= 0 {
			return fmt.Errorf("security.rate_limit.max_requests must be positive when enabled")
		}
		if c.Security.RateLimit.Window <= 0 {
			return fmt.Errorf("security.rate_limit.window must be positive when enabled")
		}
	}

	// Validate TLS configuration
	if err := c.Security.TLSConfig.Validate(); err != nil {
		return err
	}

	if c.Security.TLSConfig.Enabled {
		// Both CertFile and KeyFile are required when TLS is enabled (already checked by Validate())
		// Now verify that the files actually exist
		if _, err := os.Stat(c.Security.TLSConfig.CertFile); os.IsNotExist(err) {
			return fmt.Errorf("tls.cert_file does not exist: %s", c.Security.TLSConfig.CertFile)
		}
		if _, err := os.Stat(c.Security.TLSConfig.KeyFile); os.IsNotExist(err) {
			return fmt.Errorf("tls.key_file does not exist: %s", c.Security.TLSConfig.KeyFile)
		}
	}

	return nil
}

// SaveToFile saves the configuration to a YAML file
func (c *Config) SaveToFile(path string) error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	return os.WriteFile(path, data, 0600)
}

// GetConfigPath returns the default configuration file path
func GetConfigPath() string {
	if configDir := os.Getenv("XDG_CONFIG_HOME"); configDir != "" {
		return filepath.Join(configDir, "gocat", "config.yaml")
	}

	if homeDir := os.Getenv("HOME"); homeDir != "" {
		return filepath.Join(homeDir, ".config", "gocat", "config.yaml")
	}

	return "config.yaml"
}

// contains checks if a slice contains a string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
