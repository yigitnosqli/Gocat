package plugin

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"plugin"
	"sync"

	"github.com/fatih/color"
	"github.com/ibrahmsql/gocat/internal/logger"
)

// Plugin interface that all plugins must implement
type Plugin interface {
	// Metadata
	Name() string
	Version() string
	Description() string
	Author() string

	// Lifecycle
	Init(config map[string]interface{}) error
	Start() error
	Stop() error
	Cleanup() error

	// Hooks
	OnConnect(host string, port int) error
	OnDisconnect(host string, port int) error
	OnDataReceived(data []byte) ([]byte, error)
	OnDataSent(data []byte) ([]byte, error)
	OnError(err error)

	// Commands
	GetCommands() []Command
	ExecuteCommand(name string, args []string) (interface{}, error)
}

// Command represents a plugin command
type Command struct {
	Name        string
	Description string
	Usage       string
	Handler     func(args []string) (interface{}, error)
}

// PluginManager manages all loaded plugins
type PluginManager struct {
	plugins      map[string]Plugin
	pluginPath   string
	mu           sync.RWMutex
	hooks        map[string][]Plugin
	enabled      map[string]bool
}

// NewPluginManager creates a new plugin manager
func NewPluginManager(pluginPath string) *PluginManager {
	return &PluginManager{
		plugins:    make(map[string]Plugin),
		pluginPath: pluginPath,
		hooks:      make(map[string][]Plugin),
		enabled:    make(map[string]bool),
	}
}

// LoadPlugin loads a plugin from a .so file
func (pm *PluginManager) LoadPlugin(filename string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	fullPath := filepath.Join(pm.pluginPath, filename)
	
	// Open the plugin
	p, err := plugin.Open(fullPath)
	if err != nil {
		return fmt.Errorf("failed to open plugin %s: %v", filename, err)
	}

	// Look for the exported Plugin symbol
	symPlugin, err := p.Lookup("Plugin")
	if err != nil {
		return fmt.Errorf("plugin %s does not export 'Plugin': %v", filename, err)
	}

	// Assert the symbol is a Plugin
	var plug Plugin
	plug, ok := symPlugin.(Plugin)
	if !ok {
		return fmt.Errorf("plugin %s: exported 'Plugin' is not of type Plugin", filename)
	}

	// Initialize the plugin
	if err := plug.Init(nil); err != nil {
		return fmt.Errorf("failed to initialize plugin %s: %v", plug.Name(), err)
	}

	// Register the plugin
	pm.plugins[plug.Name()] = plug
	pm.enabled[plug.Name()] = true

	logger.Info("Loaded plugin: %s v%s by %s", plug.Name(), plug.Version(), plug.Author())
	return nil
}

// LoadAllPlugins loads all plugins from the plugin directory
func (pm *PluginManager) LoadAllPlugins() error {
	// Create plugin directory if it doesn't exist
	if err := os.MkdirAll(pm.pluginPath, 0755); err != nil {
		return fmt.Errorf("failed to create plugin directory: %v", err)
	}

	// List all .so files
	files, err := ioutil.ReadDir(pm.pluginPath)
	if err != nil {
		return fmt.Errorf("failed to read plugin directory: %v", err)
	}

	loadedCount := 0
	for _, file := range files {
		if filepath.Ext(file.Name()) == ".so" {
			if err := pm.LoadPlugin(file.Name()); err != nil {
				logger.Error("Failed to load plugin %s: %v", file.Name(), err)
			} else {
				loadedCount++
			}
		}
	}

	logger.Info("Loaded %d plugins", loadedCount)
	return nil
}

// GetPlugin returns a plugin by name
func (pm *PluginManager) GetPlugin(name string) (Plugin, bool) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	
	plugin, exists := pm.plugins[name]
	return plugin, exists
}

// ListPlugins returns a list of all loaded plugins
func (pm *PluginManager) ListPlugins() []Plugin {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	plugins := make([]Plugin, 0, len(pm.plugins))
	for _, p := range pm.plugins {
		plugins = append(plugins, p)
	}
	return plugins
}

// EnablePlugin enables a plugin
func (pm *PluginManager) EnablePlugin(name string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	plugin, exists := pm.plugins[name]
	if !exists {
		return fmt.Errorf("plugin %s not found", name)
	}

	if pm.enabled[name] {
		return fmt.Errorf("plugin %s is already enabled", name)
	}

	if err := plugin.Start(); err != nil {
		return fmt.Errorf("failed to start plugin %s: %v", name, err)
	}

	pm.enabled[name] = true
	logger.Info("Enabled plugin: %s", name)
	return nil
}

// DisablePlugin disables a plugin
func (pm *PluginManager) DisablePlugin(name string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	plugin, exists := pm.plugins[name]
	if !exists {
		return fmt.Errorf("plugin %s not found", name)
	}

	if !pm.enabled[name] {
		return fmt.Errorf("plugin %s is already disabled", name)
	}

	if err := plugin.Stop(); err != nil {
		return fmt.Errorf("failed to stop plugin %s: %v", name, err)
	}

	pm.enabled[name] = false
	logger.Info("Disabled plugin: %s", name)
	return nil
}

// UnloadPlugin unloads a plugin
func (pm *PluginManager) UnloadPlugin(name string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	plugin, exists := pm.plugins[name]
	if !exists {
		return fmt.Errorf("plugin %s not found", name)
	}

	// Stop if enabled
	if pm.enabled[name] {
		if err := plugin.Stop(); err != nil {
			logger.Warn("Error stopping plugin %s: %v", name, err)
		}
	}

	// Cleanup
	if err := plugin.Cleanup(); err != nil {
		logger.Warn("Error cleaning up plugin %s: %v", name, err)
	}

	// Remove from maps
	delete(pm.plugins, name)
	delete(pm.enabled, name)

	logger.Info("Unloaded plugin: %s", name)
	return nil
}

// ExecuteHook executes a hook for all enabled plugins
func (pm *PluginManager) ExecuteHook(hookName string, args ...interface{}) error {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	for name, plugin := range pm.plugins {
		if !pm.enabled[name] {
			continue
		}

		switch hookName {
		case "OnConnect":
			if len(args) >= 2 {
				if host, ok := args[0].(string); ok {
					if port, ok := args[1].(int); ok {
						if err := plugin.OnConnect(host, port); err != nil {
							logger.Warn("Plugin %s OnConnect hook failed: %v", name, err)
						}
					}
				}
			}

		case "OnDisconnect":
			if len(args) >= 2 {
				if host, ok := args[0].(string); ok {
					if port, ok := args[1].(int); ok {
						if err := plugin.OnDisconnect(host, port); err != nil {
							logger.Warn("Plugin %s OnDisconnect hook failed: %v", name, err)
						}
					}
				}
			}

		case "OnError":
			if len(args) >= 1 {
				if err, ok := args[0].(error); ok {
					plugin.OnError(err)
				}
			}
		}
	}

	return nil
}

// ProcessData processes data through all enabled plugins
func (pm *PluginManager) ProcessData(data []byte, direction string) ([]byte, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	processedData := data
	for name, plugin := range pm.plugins {
		if !pm.enabled[name] {
			continue
		}

		var err error
		switch direction {
		case "received":
			processedData, err = plugin.OnDataReceived(processedData)
		case "sent":
			processedData, err = plugin.OnDataSent(processedData)
		}

		if err != nil {
			logger.Warn("Plugin %s data processing failed: %v", name, err)
		}
	}

	return processedData, nil
}

// GetAllCommands returns all commands from all enabled plugins
func (pm *PluginManager) GetAllCommands() map[string][]Command {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	commands := make(map[string][]Command)
	for name, plugin := range pm.plugins {
		if pm.enabled[name] {
			commands[name] = plugin.GetCommands()
		}
	}
	return commands
}

// ExecutePluginCommand executes a command on a specific plugin
func (pm *PluginManager) ExecutePluginCommand(pluginName, commandName string, args []string) (interface{}, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	plugin, exists := pm.plugins[pluginName]
	if !exists {
		return nil, fmt.Errorf("plugin %s not found", pluginName)
	}

	if !pm.enabled[pluginName] {
		return nil, fmt.Errorf("plugin %s is disabled", pluginName)
	}

	return plugin.ExecuteCommand(commandName, args)
}

// PrintPluginInfo prints information about all loaded plugins
func (pm *PluginManager) PrintPluginInfo() {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	if len(pm.plugins) == 0 {
		color.Yellow("No plugins loaded")
		return
	}

	fmt.Println()
	color.New(color.FgCyan, color.Bold).Println("╔══════════════════════════════════════════════╗")
	color.New(color.FgCyan, color.Bold).Println("║              🔌 LOADED PLUGINS              ║")
	color.New(color.FgCyan, color.Bold).Println("╚══════════════════════════════════════════════╝")
	fmt.Println()

	for name, plugin := range pm.plugins {
		status := "❌ Disabled"
		statusColor := color.FgRed
		if pm.enabled[name] {
			status = "✅ Enabled"
			statusColor = color.FgGreen
		}

		color.New(color.FgWhite, color.Bold).Printf("📦 %s", name)
		color.New(statusColor).Printf(" [%s]\n", status)
		fmt.Printf("   Version: %s\n", plugin.Version())
		fmt.Printf("   Author: %s\n", plugin.Author())
		fmt.Printf("   Description: %s\n", plugin.Description())
		
		commands := plugin.GetCommands()
		if len(commands) > 0 {
			fmt.Printf("   Commands: ")
			for i, cmd := range commands {
				if i > 0 {
					fmt.Print(", ")
				}
				fmt.Print(cmd.Name)
			}
			fmt.Println()
		}
		fmt.Println()
	}
}

// Shutdown shuts down all plugins
func (pm *PluginManager) Shutdown() error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	for name, plugin := range pm.plugins {
		if pm.enabled[name] {
			if err := plugin.Stop(); err != nil {
				logger.Warn("Error stopping plugin %s: %v", name, err)
			}
		}
		if err := plugin.Cleanup(); err != nil {
			logger.Warn("Error cleaning up plugin %s: %v", name, err)
		}
	}

	pm.plugins = make(map[string]Plugin)
	pm.enabled = make(map[string]bool)
	
	return nil
}
