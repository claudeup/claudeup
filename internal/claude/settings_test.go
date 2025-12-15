// ABOUTME: Unit tests for Claude Code settings functionality
// ABOUTME: Tests settings loading and enabled plugin checking
package claude

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadSettings(t *testing.T) {
	// Create temp directory
	tempDir, err := os.MkdirTemp("", "claudeup-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create settings.json
	settings := map[string]interface{}{
		"model": "sonnet",
		"enabledPlugins": map[string]bool{
			"plugin1@marketplace": true,
			"plugin2@marketplace": false,
			"plugin3@marketplace": true,
		},
	}

	data, _ := json.Marshal(settings)
	if err := os.WriteFile(filepath.Join(tempDir, "settings.json"), data, 0644); err != nil {
		t.Fatal(err)
	}

	// Load settings
	loadedSettings, err := LoadSettings(tempDir)
	if err != nil {
		t.Fatal(err)
	}

	// Verify enabled plugins were loaded
	if len(loadedSettings.EnabledPlugins) != 3 {
		t.Errorf("Expected 3 plugins in enabledPlugins, got %d", len(loadedSettings.EnabledPlugins))
	}

	if !loadedSettings.EnabledPlugins["plugin1@marketplace"] {
		t.Error("plugin1@marketplace should be enabled")
	}

	if loadedSettings.EnabledPlugins["plugin2@marketplace"] {
		t.Error("plugin2@marketplace should be disabled")
	}

	if !loadedSettings.EnabledPlugins["plugin3@marketplace"] {
		t.Error("plugin3@marketplace should be enabled")
	}
}

func TestLoadSettingsNoFile(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "claudeup-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Try to load settings from directory with no settings.json
	_, err = LoadSettings(tempDir)
	if err == nil {
		t.Error("Expected error when loading non-existent settings.json")
	}
}

func TestIsPluginEnabled(t *testing.T) {
	settings := &Settings{
		EnabledPlugins: map[string]bool{
			"enabled-plugin@marketplace":  true,
			"disabled-plugin@marketplace": false,
		},
	}

	// Test enabled plugin
	if !settings.IsPluginEnabled("enabled-plugin@marketplace") {
		t.Error("enabled-plugin@marketplace should return true")
	}

	// Test disabled plugin
	if settings.IsPluginEnabled("disabled-plugin@marketplace") {
		t.Error("disabled-plugin@marketplace should return false")
	}

	// Test plugin not in map (should be false)
	if settings.IsPluginEnabled("nonexistent-plugin@marketplace") {
		t.Error("nonexistent-plugin@marketplace should return false")
	}
}

func TestIsPluginEnabledEmptyMap(t *testing.T) {
	settings := &Settings{
		EnabledPlugins: map[string]bool{},
	}

	// Plugin not in map should return false
	if settings.IsPluginEnabled("any-plugin@marketplace") {
		t.Error("Plugin not in enabledPlugins map should return false")
	}
}
