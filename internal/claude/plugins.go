// ABOUTME: Data structures and functions for managing Claude Code plugins
// ABOUTME: Handles reading and writing installed_plugins.json
package claude

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/claudeup/claudeup/v5/internal/events"
)

// PluginRegistry represents the installed_plugins.json file structure
// Version 2 format uses arrays to support multiple scopes per plugin
type PluginRegistry struct {
	Version int                         `json:"version"`
	Plugins map[string][]PluginMetadata `json:"plugins"`
}

// PluginMetadata represents metadata for an installed plugin
type PluginMetadata struct {
	Scope        string `json:"scope"` // "user" or "project"
	Version      string `json:"version"`
	InstalledAt  string `json:"installedAt"`
	LastUpdated  string `json:"lastUpdated"`
	InstallPath  string `json:"installPath"`
	GitCommitSha string `json:"gitCommitSha"`
	IsLocal      bool   `json:"isLocal"`
}

// ScopedPlugin pairs a plugin name with its scope-specific metadata.
// Used by GetPluginsAtScopes to flatten the registry into a single slice.
type ScopedPlugin struct {
	Name string
	PluginMetadata
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

// GetPluginAtScope retrieves a plugin's metadata for a specific scope.
// Returns (metadata, true) if found, (zero, false) if not.
func (r *PluginRegistry) GetPluginAtScope(pluginName, scope string) (PluginMetadata, bool) {
	instances, exists := r.Plugins[pluginName]
	if !exists {
		return PluginMetadata{}, false
	}
	for _, inst := range instances {
		if inst.Scope == scope {
			return inst, true
		}
	}
	return PluginMetadata{}, false
}

// GetPluginInstances returns all scope instances for a plugin.
// Returns nil if the plugin is not in the registry.
func (r *PluginRegistry) GetPluginInstances(pluginName string) []PluginMetadata {
	return r.Plugins[pluginName]
}

// GetPluginsAtScopes returns all plugin instances installed at the given scopes.
// Each instance is paired with its plugin name.
func (r *PluginRegistry) GetPluginsAtScopes(scopes []string) []ScopedPlugin {
	scopeSet := make(map[string]bool, len(scopes))
	for _, s := range scopes {
		scopeSet[s] = true
	}

	var result []ScopedPlugin
	for name, instances := range r.Plugins {
		for _, inst := range instances {
			if scopeSet[inst.Scope] {
				result = append(result, ScopedPlugin{Name: name, PluginMetadata: inst})
			}
		}
	}
	return result
}

// PluginExistsAtScope checks if a plugin is installed at a specific scope
func (r *PluginRegistry) PluginExistsAtScope(pluginName, scope string) bool {
	_, exists := r.GetPluginAtScope(pluginName, scope)
	return exists
}

// PluginExistsAtAnyScope checks if a plugin is installed at any scope
func (r *PluginRegistry) PluginExistsAtAnyScope(pluginName string) bool {
	instances, exists := r.Plugins[pluginName]
	return exists && len(instances) > 0
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

// RemovePlugin removes a plugin from the registry entirely
func (r *PluginRegistry) RemovePlugin(pluginName string) bool {
	if _, exists := r.Plugins[pluginName]; !exists {
		return false
	}
	delete(r.Plugins, pluginName)
	return true
}
