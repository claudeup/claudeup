// ABOUTME: Data structures and functions for managing Claude Code plugins
// ABOUTME: Handles reading and writing installed_plugins.json
package claude

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/claudeup/claudeup/internal/events"
)

// PluginRegistry represents the installed_plugins.json file structure
// Version 2 format uses arrays to support multiple scopes per plugin
type PluginRegistry struct {
	Version int                         `json:"version"`
	Plugins map[string][]PluginMetadata `json:"plugins"`
}

// PluginMetadata represents metadata for an installed plugin
type PluginMetadata struct {
	Scope        string `json:"scope"`        // "user" or "project"
	Version      string `json:"version"`
	InstalledAt  string `json:"installedAt"`
	LastUpdated  string `json:"lastUpdated"`
	InstallPath  string `json:"installPath"`
	GitCommitSha string `json:"gitCommitSha"`
	IsLocal      bool   `json:"isLocal"`
}

// LoadPlugins reads and parses the installed_plugins.json file
// Supports both V1 (single objects) and V2 (arrays with scopes) formats
func LoadPlugins(claudeDir string) (*PluginRegistry, error) {
	// Check if Claude directory exists
	if _, err := os.Stat(claudeDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("Claude CLI not found (directory %s does not exist)", claudeDir)
	}

	pluginsPath := filepath.Join(claudeDir, "plugins", "installed_plugins.json")
	data, err := os.ReadFile(pluginsPath)
	if os.IsNotExist(err) {
		// Fresh Claude install - no plugins installed yet
		return &PluginRegistry{
			Version: 2,
			Plugins: make(map[string][]PluginMetadata),
		}, nil
	}
	if err != nil {
		return nil, err
	}

	// Try V2 format (arrays) first
	var registry PluginRegistry
	err = json.Unmarshal(data, &registry)
	if err == nil && registry.Version == 2 {
		// Validate V2 format
		if err := validatePluginRegistry(&registry); err != nil {
			return nil, err
		}
		return &registry, nil
	}

	// Fall back to V1 format (single objects) and convert to V2
	type PluginMetadataV1 struct {
		Version      string `json:"version"`
		InstalledAt  string `json:"installedAt"`
		LastUpdated  string `json:"lastUpdated"`
		InstallPath  string `json:"installPath"`
		GitCommitSha string `json:"gitCommitSha"`
		IsLocal      bool   `json:"isLocal"`
	}
	type PluginRegistryV1 struct {
		Version int                         `json:"version"`
		Plugins map[string]PluginMetadataV1 `json:"plugins"`
	}

	var registryV1 PluginRegistryV1
	if err := json.Unmarshal(data, &registryV1); err != nil {
		return nil, fmt.Errorf("failed to parse installed_plugins.json as V1 or V2 format: %w (file may be corrupted)", err)
	}

	// Convert V1 to V2 format
	registry = PluginRegistry{
		Version: 2, // Upgrade to V2
		Plugins: make(map[string][]PluginMetadata),
	}
	for name, metaV1 := range registryV1.Plugins {
		registry.Plugins[name] = []PluginMetadata{{
			Scope:        "user", // V1 didn't have scopes, default to user
			Version:      metaV1.Version,
			InstalledAt:  metaV1.InstalledAt,
			LastUpdated:  metaV1.LastUpdated,
			InstallPath:  metaV1.InstallPath,
			GitCommitSha: metaV1.GitCommitSha,
			IsLocal:      metaV1.IsLocal,
		}}
	}

	// Validate converted registry
	if err := validatePluginRegistry(&registry); err != nil {
		return nil, err
	}

	return &registry, nil
}

// SavePlugins writes the plugin registry back to installed_plugins.json
func SavePlugins(claudeDir string, registry *PluginRegistry) error {
	pluginsPath := filepath.Join(claudeDir, "plugins", "installed_plugins.json")

	data, err := json.MarshalIndent(registry, "", "  ")
	if err != nil {
		return err
	}

	// Wrap file write with event tracking
	// Note: Operation name is generic. Phase 2 will add context to distinguish
	// between "profile apply", "plugin cleanup", and "plugin update" operations.
	return events.GlobalTracker().RecordFileWrite(
		"plugin update",
		pluginsPath,
		"user",
		func() error {
			return os.WriteFile(pluginsPath, data, 0644)
		},
	)
}

// PathExists checks if a plugin's install path actually exists
func (p *PluginMetadata) PathExists() bool {
	if p.InstallPath == "" {
		return false
	}
	_, err := os.Stat(p.InstallPath)
	return err == nil
}

// GetPlugin retrieves a plugin by name, defaulting to "user" scope
// Returns (metadata, exists) where exists is false if plugin not found
func (r *PluginRegistry) GetPlugin(pluginName string) (PluginMetadata, bool) {
	instances, exists := r.Plugins[pluginName]
	if !exists || len(instances) == 0 {
		return PluginMetadata{}, false
	}
	// Return first instance with "user" scope, or first instance if no user scope
	for _, inst := range instances {
		if inst.Scope == "user" || inst.Scope == "" {
			return inst, true
		}
	}
	return instances[0], true
}

// SetPlugin sets or updates a plugin's metadata for the "user" scope
func (r *PluginRegistry) SetPlugin(pluginName string, metadata PluginMetadata) {
	// Ensure scope is set
	if metadata.Scope == "" {
		metadata.Scope = "user"
	}

	instances, exists := r.Plugins[pluginName]
	if !exists {
		// New plugin, create array with single entry
		r.Plugins[pluginName] = []PluginMetadata{metadata}
		return
	}

	// Update existing user-scoped instance or append
	for i, inst := range instances {
		if inst.Scope == metadata.Scope {
			instances[i] = metadata
			r.Plugins[pluginName] = instances
			return
		}
	}

	// No matching scope, append
	r.Plugins[pluginName] = append(instances, metadata)
}

// GetAllPlugins returns a map of plugin names to their user-scoped metadata
// This simplifies iteration for code that doesn't care about scopes
func (r *PluginRegistry) GetAllPlugins() map[string]PluginMetadata {
	result := make(map[string]PluginMetadata)
	for name := range r.Plugins {
		if meta, exists := r.GetPlugin(name); exists {
			result[name] = meta
		}
	}
	return result
}

// DisablePlugin removes a plugin from the registry
func (r *PluginRegistry) DisablePlugin(pluginName string) bool {
	if _, exists := r.Plugins[pluginName]; !exists {
		return false // Plugin not found
	}
	delete(r.Plugins, pluginName)
	return true
}

// EnablePlugin adds a plugin back to the registry
// Note: This requires having the plugin metadata available
func (r *PluginRegistry) EnablePlugin(pluginName string, metadata PluginMetadata) {
	r.SetPlugin(pluginName, metadata)
}

// PluginExists checks if a plugin is in the registry
func (r *PluginRegistry) PluginExists(pluginName string) bool {
	_, exists := r.GetPlugin(pluginName)
	return exists
}

// IsPluginInstalled checks if a plugin is installed (alias for PluginExists)
func (r *PluginRegistry) IsPluginInstalled(pluginName string) bool {
	return r.PluginExists(pluginName)
}

// RemovePlugin removes a plugin from the registry entirely
func (r *PluginRegistry) RemovePlugin(pluginName string) bool {
	if _, exists := r.Plugins[pluginName]; !exists {
		return false
	}
	delete(r.Plugins, pluginName)
	return true
}
