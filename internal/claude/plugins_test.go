// ABOUTME: Unit tests for plugin registry management
// ABOUTME: Tests loading, saving, and plugin operations
package claude

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestPluginPathExists(t *testing.T) {
	// Create temp directory
	tempDir, err := os.MkdirTemp("", "claudeup-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create test plugin directory
	pluginPath := filepath.Join(tempDir, "test-plugin")
	if err := os.MkdirAll(pluginPath, 0755); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name     string
		metadata PluginMetadata
		want     bool
	}{
		{
			name: "existing path",
			metadata: PluginMetadata{
				InstallPath: pluginPath,
			},
			want: true,
		},
		{
			name: "non-existent path",
			metadata: PluginMetadata{
				InstallPath: filepath.Join(tempDir, "non-existent"),
			},
			want: false,
		},
		{
			name: "empty path",
			metadata: PluginMetadata{
				InstallPath: "",
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.metadata.PathExists(); got != tt.want {
				t.Errorf("PathExists() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDisablePlugin(t *testing.T) {
	registry := &PluginRegistry{
		Version: 1,
		Plugins: map[string]PluginMetadata{
			"test-plugin": {
				Version: "1.0.0",
			},
		},
	}

	// Disable existing plugin
	if !registry.DisablePlugin("test-plugin") {
		t.Error("DisablePlugin should return true for existing plugin")
	}

	// Verify plugin was removed
	if _, exists := registry.Plugins["test-plugin"]; exists {
		t.Error("Plugin should be removed from registry after disable")
	}

	// Disable non-existent plugin
	if registry.DisablePlugin("non-existent") {
		t.Error("DisablePlugin should return false for non-existent plugin")
	}
}

func TestEnablePlugin(t *testing.T) {
	registry := &PluginRegistry{
		Version: 1,
		Plugins: make(map[string]PluginMetadata),
	}

	metadata := PluginMetadata{
		Version:     "1.0.0",
		InstallPath: "/test/path",
	}

	// Enable plugin
	registry.EnablePlugin("test-plugin", metadata)

	// Verify plugin was added
	plugin, exists := registry.Plugins["test-plugin"]
	if !exists {
		t.Error("Plugin should exist after enable")
	}

	if plugin.Version != "1.0.0" {
		t.Errorf("Expected version 1.0.0, got %s", plugin.Version)
	}

	if plugin.InstallPath != "/test/path" {
		t.Errorf("Expected path /test/path, got %s", plugin.InstallPath)
	}
}

func TestPluginExists(t *testing.T) {
	registry := &PluginRegistry{
		Version: 1,
		Plugins: map[string]PluginMetadata{
			"existing-plugin": {
				Version: "1.0.0",
			},
		},
	}

	if !registry.PluginExists("existing-plugin") {
		t.Error("PluginExists should return true for existing plugin")
	}

	if registry.PluginExists("non-existent") {
		t.Error("PluginExists should return false for non-existent plugin")
	}
}

func TestLoadAndSavePlugins(t *testing.T) {
	// Create temp directory
	tempDir, err := os.MkdirTemp("", "claudeup-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create plugins directory
	pluginsDir := filepath.Join(tempDir, "plugins")
	if err := os.MkdirAll(pluginsDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create test registry
	registry := &PluginRegistry{
		Version: 1,
		Plugins: map[string]PluginMetadata{
			"test-plugin@test-marketplace": {
				Version:      "1.0.0",
				InstallPath:  "/test/path",
				GitCommitSha: "abc123",
				IsLocal:      true,
			},
		},
	}

	// Save registry
	if err := SavePlugins(tempDir, registry); err != nil {
		t.Fatal(err)
	}

	// Verify file exists
	pluginsFile := filepath.Join(tempDir, "plugins", "installed_plugins.json")
	if _, err := os.Stat(pluginsFile); os.IsNotExist(err) {
		t.Error("installed_plugins.json should exist after save")
	}

	// Load registry
	loaded, err := LoadPlugins(tempDir)
	if err != nil {
		t.Fatal(err)
	}

	// Verify loaded data
	if loaded.Version != 1 {
		t.Errorf("Expected version 1, got %d", loaded.Version)
	}

	plugin, exists := loaded.Plugins["test-plugin@test-marketplace"]
	if !exists {
		t.Error("Plugin should exist in loaded registry")
	}

	if plugin.Version != "1.0.0" {
		t.Errorf("Expected version 1.0.0, got %s", plugin.Version)
	}

	if plugin.GitCommitSha != "abc123" {
		t.Errorf("Expected commit abc123, got %s", plugin.GitCommitSha)
	}
}

func TestLoadPluginsNonExistent(t *testing.T) {
	// Try to load from non-existent directory
	_, err := LoadPlugins("/non/existent/path")
	if err == nil {
		t.Error("LoadPlugins should return error for non-existent path")
	}
}

func TestSavePluginsInvalidPath(t *testing.T) {
	registry := &PluginRegistry{
		Version: 1,
		Plugins: make(map[string]PluginMetadata),
	}

	// Try to save to invalid path
	err := SavePlugins("/invalid/path/that/does/not/exist", registry)
	if err == nil {
		t.Error("SavePlugins should return error for invalid path")
	}
}

func TestPluginRegistryJSONMarshaling(t *testing.T) {
	registry := &PluginRegistry{
		Version: 1,
		Plugins: map[string]PluginMetadata{
			"test-plugin": {
				Version:      "1.0.0",
				InstallPath:  "/test/path",
				GitCommitSha: "abc123",
				IsLocal:      false,
			},
		},
	}

	// Marshal to JSON
	data, err := json.MarshalIndent(registry, "", "  ")
	if err != nil {
		t.Fatal(err)
	}

	// Unmarshal from JSON
	var loaded PluginRegistry
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatal(err)
	}

	// Verify data integrity
	if loaded.Version != registry.Version {
		t.Error("Version mismatch after JSON round-trip")
	}

	if len(loaded.Plugins) != len(registry.Plugins) {
		t.Error("Plugin count mismatch after JSON round-trip")
	}

	plugin := loaded.Plugins["test-plugin"]
	if plugin.Version != "1.0.0" {
		t.Error("Plugin version mismatch after JSON round-trip")
	}
}
