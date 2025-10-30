package mcp

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/ibrahmsql/gocat/internal/logger"
)

// MCPClient represents an MCP client configuration
type MCPClient struct {
	Name       string
	ConfigPath string
	Detected   bool
	Installed  bool
}

// MCPServerConfig represents the server configuration in client config
type MCPServerConfig struct {
	Command string   `json:"command"`
	Args    []string `json:"args"`
}

// ClientConfig represents the full client configuration file
type ClientConfig struct {
	MCPServers map[string]MCPServerConfig `json:"mcpServers"`
}

// DetectMCPClients detects installed MCP clients
func DetectMCPClients() []MCPClient {
	clients := []MCPClient{}

	// Claude Desktop
	claudePath := getClaudeDesktopConfigPath()
	if claudePath != "" {
		clients = append(clients, MCPClient{
			Name:       "Claude Desktop",
			ConfigPath: claudePath,
			Detected:   true,
			Installed:  fileExists(filepath.Dir(claudePath)),
		})
	}

	// Cursor
	cursorPath := getCursorConfigPath()
	if cursorPath != "" {
		clients = append(clients, MCPClient{
			Name:       "Cursor",
			ConfigPath: cursorPath,
			Detected:   true,
			Installed:  fileExists(filepath.Dir(cursorPath)),
		})
	}

	// Continue (another AI IDE)
	continuePath := getContinueConfigPath()
	if continuePath != "" {
		clients = append(clients, MCPClient{
			Name:       "Continue",
			ConfigPath: continuePath,
			Detected:   true,
			Installed:  fileExists(filepath.Dir(continuePath)),
		})
	}

	// Zed Editor
	zedPath := getZedConfigPath()
	if zedPath != "" {
		clients = append(clients, MCPClient{
			Name:       "Zed Editor",
			ConfigPath: zedPath,
			Detected:   true,
			Installed:  fileExists(filepath.Dir(zedPath)),
		})
	}

	// Windsurf
	windsurfPath := getWindsurfConfigPath()
	if windsurfPath != "" {
		clients = append(clients, MCPClient{
			Name:       "Windsurf",
			ConfigPath: windsurfPath,
			Detected:   true,
			Installed:  fileExists(filepath.Dir(windsurfPath)),
		})
	}

	return clients
}

// getClaudeDesktopConfigPath returns Claude Desktop config path
func getClaudeDesktopConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(home, "Library", "Application Support", "Claude", "claude_desktop_config.json")
	case "linux":
		return filepath.Join(home, ".config", "Claude", "claude_desktop_config.json")
	case "windows":
		return filepath.Join(home, "AppData", "Roaming", "Claude", "claude_desktop_config.json")
	}
	return ""
}

// getCursorConfigPath returns Cursor config path
func getCursorConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(home, "Library", "Application Support", "Cursor", "User", "globalStorage", "rooveterinaryinc.roo-cline", "settings", "cline_mcp_settings.json")
	case "linux":
		return filepath.Join(home, ".config", "Cursor", "User", "globalStorage", "rooveterinaryinc.roo-cline", "settings", "cline_mcp_settings.json")
	case "windows":
		return filepath.Join(home, "AppData", "Roaming", "Cursor", "User", "globalStorage", "rooveterinaryinc.roo-cline", "settings", "cline_mcp_settings.json")
	}
	return ""
}

// getContinueConfigPath returns Continue config path
func getContinueConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	return filepath.Join(home, ".continue", "config.json")
}

// getZedConfigPath returns Zed editor config path
func getZedConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(home, ".config", "zed", "settings.json")
	case "linux":
		return filepath.Join(home, ".config", "zed", "settings.json")
	case "windows":
		return filepath.Join(home, "AppData", "Roaming", "Zed", "settings.json")
	}
	return ""
}

// getWindsurfConfigPath returns Windsurf config path
func getWindsurfConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	// Windsurf uses Kiro backend
	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(home, ".config", "Kiro", "User", "settings.json")
	case "linux":
		return filepath.Join(home, ".config", "Kiro", "User", "settings.json")
	case "windows":
		return filepath.Join(home, "AppData", "Roaming", "Kiro", "User", "settings.json")
	}
	return ""
}

// AddToClient adds GoCat MCP server to a client's configuration
func AddToClient(client MCPClient, gocatPath string) error {
	// Ensure directory exists
	configDir := filepath.Dir(client.ConfigPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Read existing config
	var config ClientConfig
	if fileExists(client.ConfigPath) {
		data, err := os.ReadFile(client.ConfigPath)
		if err != nil {
			return fmt.Errorf("failed to read config: %w", err)
		}

		if err := json.Unmarshal(data, &config); err != nil {
			// If parsing fails, create new config
			config = ClientConfig{
				MCPServers: make(map[string]MCPServerConfig),
			}
		}
	} else {
		// Create new config
		config = ClientConfig{
			MCPServers: make(map[string]MCPServerConfig),
		}
	}

	// Ensure MCPServers map exists
	if config.MCPServers == nil {
		config.MCPServers = make(map[string]MCPServerConfig)
	}

	// Add or update GoCat server
	config.MCPServers["gocat"] = MCPServerConfig{
		Command: gocatPath,
		Args:    []string{"mcp"},
	}

	// Write config back
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(client.ConfigPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	logger.Info("✓ Added GoCat MCP server to %s config", client.Name)
	logger.Info("  Config: %s", client.ConfigPath)
	logger.Info("  Command: %s mcp", gocatPath)

	return nil
}

// RemoveFromClient removes GoCat MCP server from a client's configuration
func RemoveFromClient(client MCPClient) error {
	if !fileExists(client.ConfigPath) {
		return fmt.Errorf("config file not found: %s", client.ConfigPath)
	}

	// Read existing config
	data, err := os.ReadFile(client.ConfigPath)
	if err != nil {
		return fmt.Errorf("failed to read config: %w", err)
	}

	var config ClientConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	// Remove GoCat server
	if config.MCPServers != nil {
		delete(config.MCPServers, "gocat")
	}

	// Write config back
	data, err = json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(client.ConfigPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	logger.Info("✓ Removed GoCat MCP server from %s config", client.Name)

	return nil
}

// GetGoCatPath returns the absolute path to the gocat executable
func GetGoCatPath() (string, error) {
	// Try to get from executable
	exePath, err := os.Executable()
	if err == nil {
		absPath, err := filepath.Abs(exePath)
		if err == nil {
			return absPath, nil
		}
	}

	// Try to find in PATH
	path, err := findInPath("gocat")
	if err == nil {
		return path, nil
	}

	// Fallback to relative path
	return "gocat", nil
}

// findInPath searches for an executable in PATH
func findInPath(name string) (string, error) {
	path := os.Getenv("PATH")
	if path == "" {
		return "", fmt.Errorf("PATH not set")
	}

	if runtime.GOOS == "windows" {
		name += ".exe"
	}

	for _, dir := range filepath.SplitList(path) {
		fullPath := filepath.Join(dir, name)
		if fileExists(fullPath) {
			return fullPath, nil
		}
	}

	return "", fmt.Errorf("not found in PATH")
}

// FileExists checks if a file exists
func FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// fileExists is an alias for FileExists (for internal use)
func fileExists(path string) bool {
	return FileExists(path)
}

// ShowClientStatus shows the status of MCP client configurations
func ShowClientStatus() {
	clients := DetectMCPClients()
	
	fmt.Println()
	fmt.Println("🔍 Detected MCP Clients:")
	fmt.Println()

	if len(clients) == 0 {
		fmt.Println("  No MCP clients detected.")
		fmt.Println()
		return
	}

	for i, client := range clients {
		status := "❌ Not Configured"
		
		if fileExists(client.ConfigPath) {
			// Check if gocat is in config
			data, err := os.ReadFile(client.ConfigPath)
			if err == nil {
				var config ClientConfig
				if json.Unmarshal(data, &config) == nil {
					if _, exists := config.MCPServers["gocat"]; exists {
						status = "✅ Configured"
					}
				}
			}
		}

		fmt.Printf("  %d. %s\n", i+1, client.Name)
		fmt.Printf("     Status: %s\n", status)
		fmt.Printf("     Config: %s\n", client.ConfigPath)
		
		if !client.Installed {
			fmt.Printf("     ⚠️  Application not detected\n")
		}
		
		fmt.Println()
	}
}

// GetConfiguredClients returns list of clients where GoCat is configured
func GetConfiguredClients() []MCPClient {
	clients := DetectMCPClients()
	configured := []MCPClient{}

	for _, client := range clients {
		if !fileExists(client.ConfigPath) {
			continue
		}

		data, err := os.ReadFile(client.ConfigPath)
		if err != nil {
			continue
		}

		var config ClientConfig
		if json.Unmarshal(data, &config) != nil {
			continue
		}

		if _, exists := config.MCPServers["gocat"]; exists {
			configured = append(configured, client)
		}
	}

	return configured
}
