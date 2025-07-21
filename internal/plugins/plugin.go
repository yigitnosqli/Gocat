package plugins

import (
	"context"
	"fmt"
	"plugin"
	"sync"

	"github.com/ibrahmsql/gocat/internal/logger"
)

// Plugin represents a loadable plugin interface
type Plugin interface {
	Name() string
	Version() string
	Description() string
	Init(ctx context.Context, config map[string]interface{}) error
	Execute(ctx context.Context, args []string) (interface{}, error)
	Cleanup() error
}

// PluginManager manages loaded plugins
type PluginManager struct {
	mu      sync.RWMutex
	plugins map[string]Plugin
	logger  *logger.Logger
}

// NewPluginManager creates a new plugin manager
func NewPluginManager() *PluginManager {
	return &PluginManager{
		plugins: make(map[string]Plugin),
		logger:  logger.GetDefaultLogger(),
	}
}

// LoadPlugin loads a plugin from a shared library file
func (pm *PluginManager) LoadPlugin(path string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// Load the plugin
	p, err := plugin.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open plugin %s: %w", path, err)
	}

	// Look for the NewPlugin symbol
	symbol, err := p.Lookup("NewPlugin")
	if err != nil {
		return fmt.Errorf("plugin %s does not export NewPlugin function: %w", path, err)
	}

	// Assert that it's a function that returns a Plugin
	newPluginFunc, ok := symbol.(func() Plugin)
	if !ok {
		return fmt.Errorf("plugin %s NewPlugin function has wrong signature", path)
	}

	// Create the plugin instance
	pluginInstance := newPluginFunc()

	// Check if plugin with same name already exists
	name := pluginInstance.Name()
	if _, exists := pm.plugins[name]; exists {
		return fmt.Errorf("plugin %s already loaded", name)
	}

	// Store the plugin
	pm.plugins[name] = pluginInstance

	pm.logger.InfoWithFields("Plugin loaded successfully", map[string]interface{}{
		"name":        name,
		"version":     pluginInstance.Version(),
		"description": pluginInstance.Description(),
		"path":        path,
	})

	return nil
}

// GetPlugin returns a plugin by name
func (pm *PluginManager) GetPlugin(name string) (Plugin, bool) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	plugin, exists := pm.plugins[name]
	return plugin, exists
}

// ListPlugins returns all loaded plugins
func (pm *PluginManager) ListPlugins() map[string]Plugin {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	result := make(map[string]Plugin)
	for name, plugin := range pm.plugins {
		result[name] = plugin
	}
	return result
}

// UnloadPlugin unloads a plugin
func (pm *PluginManager) UnloadPlugin(name string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	plugin, exists := pm.plugins[name]
	if !exists {
		return fmt.Errorf("plugin %s not found", name)
	}

	// Cleanup the plugin
	if err := plugin.Cleanup(); err != nil {
		pm.logger.ErrorWithFields("Plugin cleanup failed", map[string]interface{}{
			"name":  name,
			"error": err.Error(),
		})
	}

	// Remove from map
	delete(pm.plugins, name)

	pm.logger.InfoWithFields("Plugin unloaded", map[string]interface{}{
		"name": name,
	})

	return nil
}

// ExecutePlugin executes a plugin with given arguments
func (pm *PluginManager) ExecutePlugin(ctx context.Context, name string, args []string) (interface{}, error) {
	plugin, exists := pm.GetPlugin(name)
	if !exists {
		return nil, fmt.Errorf("plugin %s not found", name)
	}

	return plugin.Execute(ctx, args)
}

// InitializePlugin initializes a plugin with configuration
func (pm *PluginManager) InitializePlugin(ctx context.Context, name string, config map[string]interface{}) error {
	plugin, exists := pm.GetPlugin(name)
	if !exists {
		return fmt.Errorf("plugin %s not found", name)
	}

	return plugin.Init(ctx, config)
}

// Shutdown gracefully shuts down all plugins
func (pm *PluginManager) Shutdown() error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	var errors []error
	for name, plugin := range pm.plugins {
		if err := plugin.Cleanup(); err != nil {
			errors = append(errors, fmt.Errorf("failed to cleanup plugin %s: %w", name, err))
		}
	}

	// Clear all plugins
	pm.plugins = make(map[string]Plugin)

	if len(errors) > 0 {
		return fmt.Errorf("plugin shutdown errors: %v", errors)
	}

	return nil
}

// PluginInfo holds information about a plugin
type PluginInfo struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Description string `json:"description"`
	Loaded      bool   `json:"loaded"`
}

// GetPluginInfo returns information about all plugins
func (pm *PluginManager) GetPluginInfo() []PluginInfo {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	var info []PluginInfo
	for _, plugin := range pm.plugins {
		info = append(info, PluginInfo{
			Name:        plugin.Name(),
			Version:     plugin.Version(),
			Description: plugin.Description(),
			Loaded:      true,
		})
	}

	return info
}
