package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/ibrahmsql/gocat/internal/logger"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// Build information variables
var (
	version   = "dev"
	buildTime = "unknown"
	gitCommit = "unknown"
	gitBranch = "unknown"
	builtBy   = "unknown"
)

var rootCmd = &cobra.Command{
	Use:   "gocat",
	Short: "A modern netcat-like tool written in Go",
	Long: `Gocat is a netcat-like tool written in Go that provides network connectivity.
It can be used for port scanning, file transfers, backdoors, port redirection,
and many other networking tasks.

Basic Usage:
  gocat connect <host> <port>    # Connect to host:port
  gocat listen <port>            # Listen on port
  gocat broker <port>            # Start connection broker
  gocat chat <port>              # Start chat server

Common Flags:
  -l, --listen                   Listen mode
  -u, --udp                      Use UDP
  -v, --verbose                  Verbose output
  -k, --keep-open                Keep listening
  --ssl                          Use SSL/TLS
  --debug                        Debug mode

For complete flag documentation, please read the man page:
  man gocat

Or visit: https://github.com/ibrahmsql/gocat
`,
}

func Execute() error {
	return rootCmd.Execute()
}

func ExecuteContext(ctx context.Context) error {
	return rootCmd.ExecuteContext(ctx)
}

// SetBuildInfo sets the build information
func SetBuildInfo(v, bt, gc, gb, bb string) {
	version = v
	buildTime = bt
	gitCommit = gc
	gitBranch = gb
	builtBy = bb
}

// showVersion displays version and build information
func showVersion() {
	// Try to use TUI styling if available
	if isTerminal() {
		// Import ui package for styled output
		// This will be handled by the ui.ShowVersion function
		fmt.Printf("GoCat %s\n\n", version)
		fmt.Println("Build Information:")
		fmt.Printf("  Version:     %s\n", version)
		fmt.Printf("  Git Commit:  %s\n", gitCommit)
		fmt.Printf("  Git Branch:  %s\n", gitBranch)
		fmt.Printf("  Build Time:  %s\n", buildTime)
		fmt.Printf("  Built By:    %s\n", builtBy)
		fmt.Println()
		fmt.Println("Runtime Information:")
		fmt.Printf("  Go Version:  %s\n", runtime.Version())
		fmt.Printf("  OS/Arch:     %s/%s\n", runtime.GOOS, runtime.GOARCH)
		fmt.Printf("  CPUs:        %d\n", runtime.NumCPU())
	} else {
		// Fallback to plain text
		fmt.Printf("GoCat %s\n\n", version)
		fmt.Println("Build Information:")
		fmt.Printf("  Version:     %s\n", version)
		fmt.Printf("  Git Commit:  %s\n", gitCommit)
		fmt.Printf("  Git Branch:  %s\n", gitBranch)
		fmt.Printf("  Build Time:  %s\n", buildTime)
		fmt.Printf("  Built By:    %s\n", builtBy)
		fmt.Println()
		fmt.Println("Runtime Information:")
		fmt.Printf("  Go Version:  %s\n", runtime.Version())
		fmt.Printf("  OS/Arch:     %s/%s\n", runtime.GOOS, runtime.GOARCH)
		fmt.Printf("  CPUs:        %d\n", runtime.NumCPU())
	}
}

// isTerminal checks if stdout is a terminal
func isTerminal() bool {
	fileInfo, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return (fileInfo.Mode() & os.ModeCharDevice) != 0
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version information",
	Run: func(cmd *cobra.Command, args []string) {
		showVersion()
	},
}

func init() {
	rootCmd.CompletionOptions.DisableDefaultCmd = true

	// Add version command
	rootCmd.AddCommand(versionCmd)

	// Network and Protocol flags
	rootCmd.PersistentFlags().BoolP("ipv4", "4", false, "Use IPv4 only")
	rootCmd.PersistentFlags().BoolP("ipv6", "6", false, "Use IPv6 only")
	rootCmd.PersistentFlags().BoolP("unixsock", "U", false, "Use Unix domain sockets only")
	rootCmd.PersistentFlags().BoolP("udp", "u", false, "Use UDP instead of default TCP")
	rootCmd.PersistentFlags().Bool("sctp", false, "Use SCTP instead of default TCP")

	// Hide advanced network flags
	rootCmd.PersistentFlags().MarkHidden("ipv4")
	rootCmd.PersistentFlags().MarkHidden("ipv6")
	rootCmd.PersistentFlags().MarkHidden("unixsock")
	rootCmd.PersistentFlags().MarkHidden("sctp")

	// Connection and Behavior flags
	rootCmd.PersistentFlags().BoolP("listen", "l", false, "Bind and listen for incoming connections")
	rootCmd.PersistentFlags().BoolP("keep-open", "k", false, "Accept multiple connections in listen mode")
	rootCmd.PersistentFlags().BoolP("nodns", "n", false, "Do not resolve hostnames via DNS")
	rootCmd.PersistentFlags().BoolP("telnet", "t", false, "Answer Telnet negotiations")
	rootCmd.PersistentFlags().Bool("zero-io", false, "Zero-I/O mode, report connection status only")
	rootCmd.PersistentFlags().BoolP("crlf", "C", false, "Use CRLF for EOL sequence")

	// Port Scanning flags
	rootCmd.PersistentFlags().BoolP("scan", "z", false, "Zero-I/O mode (used for scanning)")
	rootCmd.PersistentFlags().Duration("scan-delay", time.Second, "Delay between port scans")
	rootCmd.PersistentFlags().Duration("scan-timeout", 3*time.Second, "Timeout for port scans")
	rootCmd.PersistentFlags().String("port-range", "", "Port range to scan (e.g., 1-1000 or 80,443,8080)")
	rootCmd.PersistentFlags().Bool("randomize-hosts", false, "Randomize target host order")
	rootCmd.PersistentFlags().Bool("randomize-ports", false, "Randomize target port order")

	// Hide advanced connection flags
	rootCmd.PersistentFlags().MarkHidden("nodns")
	rootCmd.PersistentFlags().MarkHidden("telnet")
	rootCmd.PersistentFlags().MarkHidden("zero-io")
	rootCmd.PersistentFlags().MarkHidden("crlf")

	// Hide advanced scanning flags
	rootCmd.PersistentFlags().MarkHidden("scan-delay")
	rootCmd.PersistentFlags().MarkHidden("scan-timeout")
	rootCmd.PersistentFlags().MarkHidden("randomize-hosts")
	rootCmd.PersistentFlags().MarkHidden("randomize-ports")

	// Timing and Connection Control
	rootCmd.PersistentFlags().DurationP("wait", "w", 0, "Connect timeout")
	rootCmd.PersistentFlags().DurationP("delay", "d", 0, "Wait between read/writes")
	rootCmd.PersistentFlags().DurationP("idle-timeout", "i", 0, "Idle read/write timeout")
	rootCmd.PersistentFlags().Duration("quit-timeout", 0, "After EOF on stdin, wait then quit")

	// Buffer and Performance flags
	rootCmd.PersistentFlags().Int("buffer-size", 8192, "Buffer size for network operations")
	rootCmd.PersistentFlags().Int("send-buffer", 0, "Socket send buffer size")
	rootCmd.PersistentFlags().Int("recv-buffer", 0, "Socket receive buffer size")
	rootCmd.PersistentFlags().String("rate-limit", "", "Rate limit for data transfer (e.g., 1MB/s)")
	rootCmd.PersistentFlags().Int("max-rate", 0, "Maximum transfer rate in bytes per second")
	rootCmd.PersistentFlags().Bool("nodelay", false, "Disable Nagle's algorithm (TCP_NODELAY)")
	rootCmd.PersistentFlags().Bool("keepalive", false, "Enable TCP keepalive")

	// Hide advanced timing flags
	rootCmd.PersistentFlags().MarkHidden("delay")
	rootCmd.PersistentFlags().MarkHidden("idle-timeout")
	rootCmd.PersistentFlags().MarkHidden("quit-timeout")

	// Hide advanced buffer and performance flags
	rootCmd.PersistentFlags().MarkHidden("buffer-size")
	rootCmd.PersistentFlags().MarkHidden("send-buffer")
	rootCmd.PersistentFlags().MarkHidden("recv-buffer")
	rootCmd.PersistentFlags().MarkHidden("rate-limit")
	rootCmd.PersistentFlags().MarkHidden("max-rate")
	rootCmd.PersistentFlags().MarkHidden("nodelay")
	rootCmd.PersistentFlags().MarkHidden("keepalive")

	// Source and Routing
	rootCmd.PersistentFlags().IntP("source-port", "p", 0, "Specify source port to use")
	rootCmd.PersistentFlags().StringP("source", "s", "", "Specify source address to use (doesn't affect -l)")
	rootCmd.PersistentFlags().String("loose-routing", "", "Loose source routing hop points (8 max)")
	rootCmd.PersistentFlags().Int("loose-pointer", 0, "Loose source routing hop pointer (4, 8, 12, ...)")

	// Hide all source and routing flags
	rootCmd.PersistentFlags().MarkHidden("source-port")
	rootCmd.PersistentFlags().MarkHidden("source")
	rootCmd.PersistentFlags().MarkHidden("loose-routing")
	rootCmd.PersistentFlags().MarkHidden("loose-pointer")

	// Execution and Command flags
	rootCmd.PersistentFlags().StringP("sh-exec", "c", "", "Executes the given command via /bin/sh")
	rootCmd.PersistentFlags().StringP("exec", "e", "", "Executes the given command")
	rootCmd.PersistentFlags().String("lua-exec", "", "Executes the given Lua script")

	// Hide execution flags
	rootCmd.PersistentFlags().MarkHidden("sh-exec")
	rootCmd.PersistentFlags().MarkHidden("exec")
	rootCmd.PersistentFlags().MarkHidden("lua-exec")

	// Connection Limits and Management
	rootCmd.PersistentFlags().IntP("max-conns", "m", 0, "Maximum simultaneous connections")

	// Hide connection limits
	rootCmd.PersistentFlags().MarkHidden("max-conns")

	// Output and Logging
	rootCmd.PersistentFlags().StringP("output", "o", "", "Dump session data to a file")
	rootCmd.PersistentFlags().StringP("hex-dump", "x", "", "Dump session data as hex to a file")
	rootCmd.PersistentFlags().Bool("append-output", false, "Append rather than clobber specified output files")
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "Set verbosity level (can be used several times)")
	rootCmd.PersistentFlags().BoolP("quiet", "q", false, "Suppress output")

	// Hide advanced output flags
	rootCmd.PersistentFlags().MarkHidden("output")
	rootCmd.PersistentFlags().MarkHidden("hex-dump")
	rootCmd.PersistentFlags().MarkHidden("append-output")
	rootCmd.PersistentFlags().MarkHidden("quiet")

	// Data Flow Control
	rootCmd.PersistentFlags().Bool("send-only", false, "Only send data, ignoring received; quit on EOF")
	rootCmd.PersistentFlags().Bool("recv-only", false, "Only receive data, never send anything")
	rootCmd.PersistentFlags().Bool("no-shutdown", false, "Continue half-duplex when receiving EOF on stdin")

	// Hide data flow control flags
	rootCmd.PersistentFlags().MarkHidden("send-only")
	rootCmd.PersistentFlags().MarkHidden("recv-only")
	rootCmd.PersistentFlags().MarkHidden("no-shutdown")

	// Access Control
	rootCmd.PersistentFlags().StringSlice("allow", nil, "Allow only given hosts to connect")
	rootCmd.PersistentFlags().String("allowfile", "", "A file of hosts allowed to connect")
	rootCmd.PersistentFlags().StringSlice("deny", nil, "Deny given hosts from connecting")
	rootCmd.PersistentFlags().String("denyfile", "", "A file of hosts denied from connecting")

	// Hide access control flags
	rootCmd.PersistentFlags().MarkHidden("allow")
	rootCmd.PersistentFlags().MarkHidden("allowfile")
	rootCmd.PersistentFlags().MarkHidden("deny")
	rootCmd.PersistentFlags().MarkHidden("denyfile")

	// Special Modes
	rootCmd.PersistentFlags().Bool("broker", false, "Enable Ncat's connection brokering mode")
	rootCmd.PersistentFlags().Bool("chat", false, "Start a simple Ncat chat server")

	// Hide special mode flags (use commands instead)
	rootCmd.PersistentFlags().MarkHidden("broker")
	rootCmd.PersistentFlags().MarkHidden("chat")

	// Proxy Support
	rootCmd.PersistentFlags().String("proxy", "", "Specify address of host to proxy through")
	rootCmd.PersistentFlags().String("proxy-type", "", "Specify proxy type (\"http\", \"socks4\", \"socks5\")")
	rootCmd.PersistentFlags().String("proxy-auth", "", "Authenticate with HTTP or SOCKS proxy server")
	rootCmd.PersistentFlags().String("proxy-dns", "", "Specify where to resolve proxy destination")

	// Hide proxy flags
	rootCmd.PersistentFlags().MarkHidden("proxy")
	rootCmd.PersistentFlags().MarkHidden("proxy-type")
	rootCmd.PersistentFlags().MarkHidden("proxy-auth")
	rootCmd.PersistentFlags().MarkHidden("proxy-dns")

	// SSL/TLS Support
	rootCmd.PersistentFlags().Bool("ssl", false, "Connect or listen with SSL")
	rootCmd.PersistentFlags().String("ssl-cert", "", "Specify SSL certificate file (PEM) for listening")
	rootCmd.PersistentFlags().String("ssl-key", "", "Specify SSL private key (PEM) for listening")
	rootCmd.PersistentFlags().Bool("ssl-verify", false, "Verify trust and domain name of certificates")
	rootCmd.PersistentFlags().String("ssl-trustfile", "", "PEM file containing trusted SSL certificates")
	rootCmd.PersistentFlags().String("ssl-ciphers", "", "Cipherlist containing SSL ciphers to use")
	rootCmd.PersistentFlags().String("ssl-servername", "", "Request distinct server name (SNI)")
	rootCmd.PersistentFlags().String("ssl-alpn", "", "ALPN protocol list to use")

	// Hide advanced SSL flags
	rootCmd.PersistentFlags().MarkHidden("ssl-cert")
	rootCmd.PersistentFlags().MarkHidden("ssl-key")
	rootCmd.PersistentFlags().MarkHidden("ssl-verify")
	rootCmd.PersistentFlags().MarkHidden("ssl-trustfile")
	rootCmd.PersistentFlags().MarkHidden("ssl-ciphers")
	rootCmd.PersistentFlags().MarkHidden("ssl-servername")
	rootCmd.PersistentFlags().MarkHidden("ssl-alpn")

	// Legacy GoCat flags
	rootCmd.PersistentFlags().Bool("debug", false, "Enable debug output")
	rootCmd.PersistentFlags().String("theme", "", "Path to color theme file (default: ~/.gocat-theme.yml)")
	rootCmd.PersistentFlags().Bool("json", false, "Output logs in JSON format")
	rootCmd.PersistentFlags().Bool("no-color", false, "Disable colored output")
	rootCmd.PersistentFlags().String("log-level", "info", "Set log level (debug, info, warn, error)")
	rootCmd.PersistentFlags().String("config", "", "Path to configuration file")

	// Hide advanced legacy flags
	rootCmd.PersistentFlags().MarkHidden("theme")
	rootCmd.PersistentFlags().MarkHidden("json")
	rootCmd.PersistentFlags().MarkHidden("no-color")
	rootCmd.PersistentFlags().MarkHidden("log-level")
	rootCmd.PersistentFlags().MarkHidden("config")

	// Initialize configuration on startup
	cobra.OnInitialize(initConfig)
}

// initConfig initializes the application configuration
func initConfig() {
	// Configure logging based on flags
	if verbose, _ := rootCmd.PersistentFlags().GetBool("verbose"); verbose {
		logger.SetLevel(logger.LevelDebug)
		logger.SetShowCaller(true)
	}

	if debug, _ := rootCmd.PersistentFlags().GetBool("debug"); debug {
		logger.SetLevel(logger.LevelDebug)
		logger.SetShowCaller(true)
	}

	if quiet, _ := rootCmd.PersistentFlags().GetBool("quiet"); quiet {
		logger.SetLevel(logger.LevelError)
	}

	if jsonOutput, _ := rootCmd.PersistentFlags().GetBool("json"); jsonOutput {
		logger.SetStructured(true)
	}

	// Set log level from flag
	if logLevel, _ := rootCmd.PersistentFlags().GetString("log-level"); logLevel != "" {
		switch logLevel {
		case "debug":
			logger.SetLevel(logger.LevelDebug)
		case "info":
			logger.SetLevel(logger.LevelInfo)
		case "warn":
			logger.SetLevel(logger.LevelWarn)
		case "error":
			logger.SetLevel(logger.LevelError)
		default:
			logger.Warn("Invalid log level '%s', using 'info'", logLevel)
			logger.SetLevel(logger.LevelInfo)
		}
	}

	// Load theme if not disabled
	if noColor, _ := rootCmd.PersistentFlags().GetBool("no-color"); !noColor {
		initTheme()
	}

	// Load configuration file if specified
	if configPath, _ := rootCmd.PersistentFlags().GetString("config"); configPath != "" {
		if err := loadConfigFile(configPath); err != nil {
			logger.Warn("Failed to load config file: %v", err)
		}
	}
}

// initTheme loads the color theme
func initTheme() {
	themePath, _ := rootCmd.PersistentFlags().GetString("theme")
	if err := logger.LoadTheme(themePath); err != nil {
		logger.Debug("Theme loading info: %v", err)
	}
}

// loadConfigFile loads configuration from a file
func loadConfigFile(configPath string) error {
	logger.Debug("Loading configuration from: %s", configPath)

	// Check if file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return fmt.Errorf("configuration file not found: %s", configPath)
	}

	// Read file content
	content, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse configuration based on file extension
	var config map[string]interface{}
	switch {
	case strings.HasSuffix(configPath, ".yml") || strings.HasSuffix(configPath, ".yaml"):
		if err := yaml.Unmarshal(content, &config); err != nil {
			return fmt.Errorf("failed to parse YAML config: %w", err)
		}
	case strings.HasSuffix(configPath, ".json"):
		if err := json.Unmarshal(content, &config); err != nil {
			return fmt.Errorf("failed to parse JSON config: %w", err)
		}
	default:
		return fmt.Errorf("unsupported config file format (supported: .yml, .yaml, .json)")
	}

	// Apply configuration values to flags
	for key, value := range config {
		if flag := rootCmd.PersistentFlags().Lookup(key); flag != nil {
			if !flag.Changed { // Only set if not already set by command line
				var stringValue string

				// Handle different value types with type switch
				switch v := value.(type) {
				case []interface{}:
					// Handle slice of interfaces (e.g., StringSlice flags)
					var strSlice []string
					for _, item := range v {
						strSlice = append(strSlice, fmt.Sprintf("%v", item))
					}
					stringValue = strings.Join(strSlice, ",")
				case []string:
					// Handle slice of strings
					stringValue = strings.Join(v, ",")
				default:
					// Handle primitive types
					stringValue = fmt.Sprintf("%v", value)
				}

				if err := flag.Value.Set(stringValue); err != nil {
					logger.Warn("Failed to set config value for %s: %v", key, err)
				}
			}
		}
	}

	logger.Debug("Configuration loaded successfully from: %s", configPath)
	return nil
}
