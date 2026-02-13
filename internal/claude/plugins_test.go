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
		Version: 2,
		Plugins: map[string][]PluginMetadata{
			"test-plugin": {{
				Scope:   "user",
				Version: "1.0.0",
			}},
		},
	}

	// Disable existing plugin
	if !registry.DisablePlugin("test-plugin") {
		t.Error("DisablePlugin should return true for existing plugin")
	}

	// Verify plugin was removed
	if _, exists := registry.GetPluginAtScope("test-plugin", "user"); exists {
		t.Error("Plugin should be removed from registry after disable")
	}

	// Disable non-existent plugin
	if registry.DisablePlugin("non-existent") {
		t.Error("DisablePlugin should return false for non-existent plugin")
	}
}

func TestEnablePlugin(t *testing.T) {
	registry := &PluginRegistry{
		Version: 2,
		Plugins: make(map[string][]PluginMetadata),
	}

	metadata := PluginMetadata{
		Scope:       "user",
		Version:     "1.0.0",
		InstallPath: "/test/path",
	}

	// Enable plugin
	registry.EnablePlugin("test-plugin", metadata)

	// Verify plugin was added
	plugin, exists := registry.GetPluginAtScope("test-plugin", "user")
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
		Version: 2,
		Plugins: map[string][]PluginMetadata{
			"existing-plugin": {{
				Scope:   "user",
				Version: "1.0.0",
			}},
		},
	}

	if !registry.PluginExistsAtAnyScope("existing-plugin") {
		t.Error("PluginExistsAtAnyScope should return true for existing plugin")
	}

	if registry.PluginExistsAtAnyScope("non-existent") {
		t.Error("PluginExistsAtAnyScope should return false for non-existent plugin")
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
		Version: 2,
		Plugins: map[string][]PluginMetadata{
			"test-plugin@test-marketplace": {{
				Scope:        "user",
				Version:      "1.0.0",
				InstallPath:  "/test/path",
				GitCommitSha: "abc123",
				IsLocal:      true,
			}},
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
	if loaded.Version != 2 {
		t.Errorf("Expected version 2, got %d", loaded.Version)
	}

	plugin, exists := loaded.GetPluginAtScope("test-plugin@test-marketplace", "user")
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

func TestLoadPluginsFreshInstall(t *testing.T) {
	// Create temp directory with plugins subdirectory but no installed_plugins.json
	// This simulates a fresh Claude Code install that hasn't installed any plugins yet
	tempDir, err := os.MkdirTemp("", "claudeup-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create plugins directory (Claude creates this on install)
	pluginsDir := filepath.Join(tempDir, "plugins")
	if err := os.MkdirAll(pluginsDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Load plugins from fresh install (no installed_plugins.json yet)
	registry, err := LoadPlugins(tempDir)
	if err != nil {
		t.Fatalf("LoadPlugins should not error on fresh install, got: %v", err)
	}

	// Should return empty V2 registry
	if registry.Version != 2 {
		t.Errorf("Expected version 2, got %d", registry.Version)
	}

	if registry.Plugins == nil {
		t.Error("Plugins map should be initialized, not nil")
	}

	if len(registry.Plugins) != 0 {
		t.Errorf("Expected 0 plugins in fresh install, got %d", len(registry.Plugins))
	}
}

func TestSavePluginsInvalidPath(t *testing.T) {
	registry := &PluginRegistry{
		Version: 2,
		Plugins: make(map[string][]PluginMetadata),
	}

	// Try to save to invalid path
	err := SavePlugins("/invalid/path/that/does/not/exist", registry)
	if err == nil {
		t.Error("SavePlugins should return error for invalid path")
	}
}

func TestPluginRegistryJSONMarshaling(t *testing.T) {
	registry := &PluginRegistry{
		Version: 2,
		Plugins: map[string][]PluginMetadata{
			"test-plugin": {{
				Scope:        "user",
				Version:      "1.0.0",
				InstallPath:  "/test/path",
				GitCommitSha: "abc123",
				IsLocal:      false,
			}},
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

	plugin, exists := loaded.GetPluginAtScope("test-plugin", "user")
	if !exists || plugin.Version != "1.0.0" {
		t.Error("Plugin version mismatch after JSON round-trip")
	}
}

func TestGetPluginAtScope(t *testing.T) {
	registry := &PluginRegistry{
		Version: 2,
		Plugins: map[string][]PluginMetadata{
			"test-plugin": {
				{Scope: "user", Version: "1.0.0"},
				{Scope: "project", Version: "2.0.0"},
			},
		},
	}

	plugin, exists := registry.GetPluginAtScope("test-plugin", "user")
	if !exists {
		t.Error("should find user-scoped instance")
	}
	if plugin.Version != "1.0.0" {
		t.Errorf("expected version 1.0.0, got %s", plugin.Version)
	}

	plugin, exists = registry.GetPluginAtScope("test-plugin", "project")
	if !exists {
		t.Error("should find project-scoped instance")
	}
	if plugin.Version != "2.0.0" {
		t.Errorf("expected version 2.0.0, got %s", plugin.Version)
	}

	_, exists = registry.GetPluginAtScope("test-plugin", "local")
	if exists {
		t.Error("should not find local-scoped instance")
	}

	_, exists = registry.GetPluginAtScope("missing", "user")
	if exists {
		t.Error("should not find non-existent plugin")
	}
}

func TestGetPluginInstances(t *testing.T) {
	registry := &PluginRegistry{
		Version: 2,
		Plugins: map[string][]PluginMetadata{
			"multi-scope": {
				{Scope: "user", Version: "1.0.0"},
				{Scope: "project", Version: "2.0.0"},
			},
			"single-scope": {
				{Scope: "user", Version: "1.0.0"},
			},
		},
	}

	instances := registry.GetPluginInstances("multi-scope")
	if len(instances) != 2 {
		t.Errorf("expected 2 instances, got %d", len(instances))
	}

	instances = registry.GetPluginInstances("single-scope")
	if len(instances) != 1 {
		t.Errorf("expected 1 instance, got %d", len(instances))
	}

	instances = registry.GetPluginInstances("missing")
	if len(instances) != 0 {
		t.Errorf("expected 0 instances, got %d", len(instances))
	}
}

func TestGetPluginsAtScopes(t *testing.T) {
	registry := &PluginRegistry{
		Version: 2,
		Plugins: map[string][]PluginMetadata{
			"plugin-a": {
				{Scope: "user", Version: "1.0.0"},
				{Scope: "project", Version: "2.0.0"},
			},
			"plugin-b": {
				{Scope: "local", Version: "3.0.0"},
			},
		},
	}

	result := registry.GetPluginsAtScopes([]string{"user"})
	if len(result) != 1 {
		t.Errorf("expected 1 result for user scope, got %d", len(result))
	}
	if result[0].Name != "plugin-a" || result[0].Version != "1.0.0" {
		t.Errorf("unexpected result: %+v", result[0])
	}

	result = registry.GetPluginsAtScopes([]string{"user", "project"})
	if len(result) != 2 {
		t.Errorf("expected 2 results for user+project, got %d", len(result))
	}

	result = registry.GetPluginsAtScopes([]string{"user", "project", "local"})
	if len(result) != 3 {
		t.Errorf("expected 3 results for all scopes, got %d", len(result))
	}

	result = registry.GetPluginsAtScopes([]string{})
	if len(result) != 0 {
		t.Errorf("expected 0 results for empty scopes, got %d", len(result))
	}
}

func TestPluginExistsAtScope(t *testing.T) {
	registry := &PluginRegistry{
		Version: 2,
		Plugins: map[string][]PluginMetadata{
			"test-plugin": {
				{Scope: "user", Version: "1.0.0"},
			},
		},
	}

	if !registry.PluginExistsAtScope("test-plugin", "user") {
		t.Error("should exist at user scope")
	}
	if registry.PluginExistsAtScope("test-plugin", "project") {
		t.Error("should not exist at project scope")
	}
	if registry.PluginExistsAtScope("missing", "user") {
		t.Error("should not find non-existent plugin")
	}
}

func TestPluginExistsAtAnyScope(t *testing.T) {
	registry := &PluginRegistry{
		Version: 2,
		Plugins: map[string][]PluginMetadata{
			"test-plugin": {
				{Scope: "project", Version: "1.0.0"},
			},
		},
	}

	if !registry.PluginExistsAtAnyScope("test-plugin") {
		t.Error("should exist at some scope")
	}
	if registry.PluginExistsAtAnyScope("missing") {
		t.Error("should not find non-existent plugin")
	}
}
