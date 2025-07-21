package plugins

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"plugin"
	"strings"
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
	mu               sync.RWMutex
	plugins          map[string]Plugin
	logger           *logger.Logger
	safeDirectories  []string
	trustedChecksums map[string]string
}

// NewPluginManager creates a new plugin manager
func NewPluginManager() *PluginManager {
	return &PluginManager{
		plugins:          make(map[string]Plugin),
		logger:           logger.GetDefaultLogger(),
		safeDirectories:  []string{"/usr/local/lib/gocat/plugins", "./plugins"},
		trustedChecksums: make(map[string]string),
	}
}

// LoadPlugin loads a plugin from a shared library file with security validation
func (pm *PluginManager) LoadPlugin(path string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// Security validation before loading
	if err := pm.validatePluginSecurity(path); err != nil {
		return fmt.Errorf("security validation failed for plugin %s: %w", path, err)
	}

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

// validatePluginSecurity performs comprehensive security validation
func (pm *PluginManager) validatePluginSecurity(path string) error {
	// 1. Validate file exists and is readable
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("plugin file does not exist: %s", path)
	}

	// 2. Validate plugin is in a safe directory
	if err := pm.validateSafeDirectory(path); err != nil {
		return err
	}

	// 3. Validate file permissions (should not be world-writable)
	if err := pm.validateFilePermissions(path); err != nil {
		return err
	}

	// 4. Validate checksum if available
	if err := pm.validateChecksum(path); err != nil {
		return err
	}

	pm.logger.InfoWithFields("Plugin security validation passed", map[string]interface{}{
		"path": path,
	})

	return nil
}

// validateSafeDirectory ensures plugin is loaded from a trusted directory
func (pm *PluginManager) validateSafeDirectory(path string) error {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("failed to resolve absolute path: %w", err)
	}

	for _, safeDir := range pm.safeDirectories {
		absSafeDir, err := filepath.Abs(safeDir)
		if err != nil {
			continue
		}

		// Check if plugin path is within safe directory
		relPath, err := filepath.Rel(absSafeDir, absPath)
		if err == nil && !strings.HasPrefix(relPath, "..") {
			return nil // Plugin is in safe directory
		}
	}

	return fmt.Errorf("plugin not in safe directory, allowed directories: %v", pm.safeDirectories)
}

// validateFilePermissions checks that plugin file has secure permissions
func (pm *PluginManager) validateFilePermissions(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("failed to get file info: %w", err)
	}

	mode := info.Mode()
	// Check if file is world-writable (security risk)
	if mode&0002 != 0 {
		return fmt.Errorf("plugin file is world-writable, this is a security risk")
	}

	// Check if file is group-writable by non-owner (potential risk)
	if mode&0020 != 0 {
		pm.logger.WarnWithFields("Plugin file is group-writable", map[string]interface{}{
			"path": path,
			"mode": mode.String(),
		})
	}

	return nil
}

// validateChecksum verifies plugin integrity using SHA256 checksum
func (pm *PluginManager) validateChecksum(path string) error {
	// Calculate file checksum
	checksum, err := pm.calculateSHA256(path)
	if err != nil {
		return fmt.Errorf("failed to calculate checksum: %w", err)
	}

	// Check against trusted checksums if available
	if trustedChecksum, exists := pm.trustedChecksums[path]; exists {
		if checksum != trustedChecksum {
			return fmt.Errorf("checksum mismatch: expected %s, got %s", trustedChecksum, checksum)
		}
		pm.logger.InfoWithFields("Plugin checksum verified", map[string]interface{}{
			"path":     path,
			"checksum": checksum,
		})
	} else {
		// Log checksum for future reference
		pm.logger.InfoWithFields("Plugin checksum calculated (not verified)", map[string]interface{}{
			"path":     path,
			"checksum": checksum,
		})
	}

	return nil
}

// calculateSHA256 calculates SHA256 hash of a file
func (pm *PluginManager) calculateSHA256(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

// AddTrustedChecksum adds a trusted checksum for a plugin
func (pm *PluginManager) AddTrustedChecksum(path, checksum string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.trustedChecksums[path] = checksum
}

// AddSafeDirectory adds a directory to the list of safe plugin directories
func (pm *PluginManager) AddSafeDirectory(dir string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.safeDirectories = append(pm.safeDirectories, dir)
}

// GetSafeDirectories returns the list of safe plugin directories
func (pm *PluginManager) GetSafeDirectories() []string {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return append([]string{}, pm.safeDirectories...)
}
