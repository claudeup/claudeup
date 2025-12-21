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

func TestSaveSettingsPreservesAllFields(t *testing.T) {
	// Create temp directory
	tempDir, err := os.MkdirTemp("", "claudeup-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create settings.json with multiple fields (like real Claude settings.json)
	originalSettings := map[string]interface{}{
		"enabledPlugins": map[string]bool{
			"plugin1@marketplace": true,
		},
		"model": "claude-sonnet-4-5",
		"includeCoAuthoredBy": true,
		"permissions": map[string]interface{}{
			"bash": map[string]interface{}{
				"enabled": true,
			},
		},
		"hooks": map[string]interface{}{
			"beforeMessage": "echo 'test'",
		},
		"statusLine": map[string]interface{}{
			"enabled": true,
			"format":  "custom",
		},
	}

	data, _ := json.Marshal(originalSettings)
	settingsPath := filepath.Join(tempDir, "settings.json")
	if err := os.WriteFile(settingsPath, data, 0644); err != nil {
		t.Fatal(err)
	}

	// Load settings
	settings, err := LoadSettings(tempDir)
	if err != nil {
		t.Fatal(err)
	}

	// Modify enabledPlugins (typical use case that triggers the bug)
	settings.EnablePlugin("plugin2@marketplace")

	// Save settings back
	if err := SaveSettings(tempDir, settings); err != nil {
		t.Fatal(err)
	}

	// Read the file and verify ALL fields are preserved
	savedData, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatal(err)
	}

	var savedSettings map[string]interface{}
	if err := json.Unmarshal(savedData, &savedSettings); err != nil {
		t.Fatal(err)
	}

	// Verify enabledPlugins was updated
	enabledPlugins := savedSettings["enabledPlugins"].(map[string]interface{})
	if len(enabledPlugins) != 2 {
		t.Errorf("Expected 2 plugins in enabledPlugins, got %d", len(enabledPlugins))
	}

	// CRITICAL: Verify all other fields are preserved
	if savedSettings["model"] == nil {
		t.Error("model field was lost during save")
	}
	if savedSettings["model"] != "claude-sonnet-4-5" {
		t.Errorf("model field changed: expected 'claude-sonnet-4-5', got %v", savedSettings["model"])
	}

	if savedSettings["includeCoAuthoredBy"] == nil {
		t.Error("includeCoAuthoredBy field was lost during save")
	}

	if savedSettings["permissions"] == nil {
		t.Error("permissions field was lost during save")
	}

	if savedSettings["hooks"] == nil {
		t.Error("hooks field was lost during save")
	}

	if savedSettings["statusLine"] == nil {
		t.Error("statusLine field was lost during save")
	}
}

func TestSettingsPathForScope(t *testing.T) {
	claudeDir := "/home/user/.claude"
	projectDir := "/home/user/project"

	tests := []struct {
		name        string
		scope       string
		projectDir  string
		expected    string
		shouldError bool
	}{
		{
			name:        "user scope",
			scope:       "user",
			projectDir:  projectDir,
			expected:    filepath.Join(claudeDir, "settings.json"),
			shouldError: false,
		},
		{
			name:        "empty scope defaults to user",
			scope:       "",
			projectDir:  projectDir,
			expected:    filepath.Join(claudeDir, "settings.json"),
			shouldError: false,
		},
		{
			name:        "project scope",
			scope:       "project",
			projectDir:  projectDir,
			expected:    filepath.Join(projectDir, ".claude", "settings.json"),
			shouldError: false,
		},
		{
			name:        "local scope",
			scope:       "local",
			projectDir:  projectDir,
			expected:    filepath.Join(projectDir, ".claude", "settings.local.json"),
			shouldError: false,
		},
		{
			name:        "project scope without projectDir",
			scope:       "project",
			projectDir:  "",
			expected:    "",
			shouldError: true,
		},
		{
			name:        "invalid scope",
			scope:       "invalid",
			projectDir:  projectDir,
			expected:    "",
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, err := SettingsPathForScope(tt.scope, claudeDir, tt.projectDir)

			if tt.shouldError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if path != tt.expected {
					t.Errorf("Expected path %q, got %q", tt.expected, path)
				}
			}
		})
	}
}

func TestLoadSettingsForScope(t *testing.T) {
	// Create temp directory structure
	tempDir, err := os.MkdirTemp("", "claudeup-scope-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	claudeDir := filepath.Join(tempDir, ".claude")
	projectDir := filepath.Join(tempDir, "project")

	// Create user scope settings
	userSettings := map[string]interface{}{
		"enabledPlugins": map[string]bool{
			"user-plugin@marketplace": true,
		},
	}
	os.MkdirAll(claudeDir, 0755)
	data, _ := json.Marshal(userSettings)
	os.WriteFile(filepath.Join(claudeDir, "settings.json"), data, 0644)

	// Create project scope settings
	projectSettings := map[string]interface{}{
		"enabledPlugins": map[string]bool{
			"project-plugin@marketplace": true,
		},
	}
	os.MkdirAll(filepath.Join(projectDir, ".claude"), 0755)
	data, _ = json.Marshal(projectSettings)
	os.WriteFile(filepath.Join(projectDir, ".claude", "settings.json"), data, 0644)

	// Test loading user scope
	userLoaded, err := LoadSettingsForScope("user", claudeDir, projectDir)
	if err != nil {
		t.Fatalf("Failed to load user scope: %v", err)
	}
	if !userLoaded.IsPluginEnabled("user-plugin@marketplace") {
		t.Error("user-plugin@marketplace should be enabled in user scope")
	}

	// Test loading project scope
	projectLoaded, err := LoadSettingsForScope("project", claudeDir, projectDir)
	if err != nil {
		t.Fatalf("Failed to load project scope: %v", err)
	}
	if !projectLoaded.IsPluginEnabled("project-plugin@marketplace") {
		t.Error("project-plugin@marketplace should be enabled in project scope")
	}

	// Test loading local scope (doesn't exist, should return empty)
	localLoaded, err := LoadSettingsForScope("local", claudeDir, projectDir)
	if err != nil {
		t.Fatalf("Failed to load local scope: %v", err)
	}
	if len(localLoaded.EnabledPlugins) != 0 {
		t.Error("local scope should be empty when file doesn't exist")
	}
}

func TestSaveSettingsForScope(t *testing.T) {
	// Create temp directory
	tempDir, err := os.MkdirTemp("", "claudeup-scope-save-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	claudeDir := filepath.Join(tempDir, ".claude")
	projectDir := filepath.Join(tempDir, "project")

	// Create settings to save
	settings := &Settings{
		EnabledPlugins: map[string]bool{
			"test-plugin@marketplace": true,
		},
	}

	// Save to project scope (directory doesn't exist yet)
	err = SaveSettingsForScope("project", claudeDir, projectDir, settings)
	if err != nil {
		t.Fatalf("Failed to save to project scope: %v", err)
	}

	// Verify directory was created
	projectSettingsPath := filepath.Join(projectDir, ".claude", "settings.json")
	if _, err := os.Stat(projectSettingsPath); os.IsNotExist(err) {
		t.Error("Project settings file should have been created")
	}

	// Verify content
	loaded, err := LoadSettingsForScope("project", claudeDir, projectDir)
	if err != nil {
		t.Fatalf("Failed to load saved settings: %v", err)
	}
	if !loaded.IsPluginEnabled("test-plugin@marketplace") {
		t.Error("test-plugin@marketplace should be enabled in saved settings")
	}
}

func TestLoadMergedSettings(t *testing.T) {
	// Create temp directory structure
	tempDir, err := os.MkdirTemp("", "claudeup-merge-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	claudeDir := filepath.Join(tempDir, ".claude")
	projectDir := filepath.Join(tempDir, "project")

	// Create user scope settings
	userSettings := map[string]interface{}{
		"enabledPlugins": map[string]bool{
			"user-plugin@marketplace":   true,
			"shared-plugin@marketplace": false, // Will be overridden by project
		},
	}
	os.MkdirAll(claudeDir, 0755)
	data, _ := json.Marshal(userSettings)
	os.WriteFile(filepath.Join(claudeDir, "settings.json"), data, 0644)

	// Create project scope settings
	projectSettings := map[string]interface{}{
		"enabledPlugins": map[string]bool{
			"project-plugin@marketplace": true,
			"shared-plugin@marketplace":  true, // Overrides user scope
		},
	}
	os.MkdirAll(filepath.Join(projectDir, ".claude"), 0755)
	data, _ = json.Marshal(projectSettings)
	os.WriteFile(filepath.Join(projectDir, ".claude", "settings.json"), data, 0644)

	// Create local scope settings
	localSettings := map[string]interface{}{
		"enabledPlugins": map[string]bool{
			"local-plugin@marketplace": true,
		},
	}
	data, _ = json.Marshal(localSettings)
	os.WriteFile(filepath.Join(projectDir, ".claude", "settings.local.json"), data, 0644)

	// Load merged settings
	merged, err := LoadMergedSettings(claudeDir, projectDir)
	if err != nil {
		t.Fatalf("Failed to load merged settings: %v", err)
	}

	// Verify all plugins from all scopes are present
	if !merged.IsPluginEnabled("user-plugin@marketplace") {
		t.Error("user-plugin@marketplace should be enabled")
	}
	if !merged.IsPluginEnabled("project-plugin@marketplace") {
		t.Error("project-plugin@marketplace should be enabled")
	}
	if !merged.IsPluginEnabled("local-plugin@marketplace") {
		t.Error("local-plugin@marketplace should be enabled")
	}

	// Verify precedence: project scope overrides user scope
	if !merged.IsPluginEnabled("shared-plugin@marketplace") {
		t.Error("shared-plugin@marketplace should be true (project overrides user)")
	}

	// Verify total count (user + project + local plugins)
	if len(merged.EnabledPlugins) != 4 {
		t.Errorf("Expected 4 plugins in merged settings, got %d", len(merged.EnabledPlugins))
	}
}
