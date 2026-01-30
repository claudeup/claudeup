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
		"model":               "claude-sonnet-4-5",
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

func TestSaveSettingsCanonicalKeyOrder(t *testing.T) {
	// Create temp directory
	tempDir, err := os.MkdirTemp("", "claudeup-order-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create settings.json with Claude Code's fields in a different order
	// (simulating what happens when we read a file that Claude Code wrote)
	originalSettings := map[string]interface{}{
		"$schema":             "https://json.schemastore.org/claude-code-settings.json",
		"enabledPlugins":      map[string]bool{"plugin1@marketplace": true},
		"hooks":               map[string]interface{}{"PreToolUse": []interface{}{}},
		"permissions":         map[string]interface{}{"allow": []interface{}{}},
		"statusLine":          map[string]interface{}{"type": "command"},
		"includeCoAuthoredBy": false,
	}

	data, _ := json.Marshal(originalSettings)
	settingsPath := filepath.Join(tempDir, "settings.json")
	if err := os.WriteFile(settingsPath, data, 0644); err != nil {
		t.Fatal(err)
	}

	// Load and save settings
	settings, err := LoadSettings(tempDir)
	if err != nil {
		t.Fatal(err)
	}

	// Add a plugin to trigger a save
	settings.EnablePlugin("plugin2@marketplace")

	if err := SaveSettings(tempDir, settings); err != nil {
		t.Fatal(err)
	}

	// Read the saved file and check key order
	savedData, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatal(err)
	}

	// Claude Code's canonical order is:
	// $schema, includeCoAuthoredBy, permissions, hooks, statusLine, enabledPlugins
	//
	// We verify this by checking that keys appear in that order in the output
	content := string(savedData)

	schemaIdx := indexOf(content, `"$schema"`)
	includeCoAuthoredByIdx := indexOf(content, `"includeCoAuthoredBy"`)
	permissionsIdx := indexOf(content, `"permissions"`)
	hooksIdx := indexOf(content, `"hooks"`)
	statusLineIdx := indexOf(content, `"statusLine"`)
	enabledPluginsIdx := indexOf(content, `"enabledPlugins"`)

	// Verify order (each key should appear after the previous one)
	if schemaIdx == -1 {
		t.Error("$schema not found in output")
	}
	if includeCoAuthoredByIdx == -1 || includeCoAuthoredByIdx < schemaIdx {
		t.Error("includeCoAuthoredBy should appear after $schema")
	}
	if permissionsIdx == -1 || permissionsIdx < includeCoAuthoredByIdx {
		t.Error("permissions should appear after includeCoAuthoredBy")
	}
	if hooksIdx == -1 || hooksIdx < permissionsIdx {
		t.Error("hooks should appear after permissions")
	}
	if statusLineIdx == -1 || statusLineIdx < hooksIdx {
		t.Error("statusLine should appear after hooks")
	}
	if enabledPluginsIdx == -1 || enabledPluginsIdx < statusLineIdx {
		t.Error("enabledPlugins should appear after statusLine")
	}
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

func TestSaveSettingsPreservesUnknownFields(t *testing.T) {
	// Create temp directory
	tempDir, err := os.MkdirTemp("", "claudeup-unknown-fields-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create settings.json with unknown fields (future Claude Code additions)
	originalSettings := map[string]interface{}{
		"$schema":         "https://json.schemastore.org/claude-code-settings.json",
		"enabledPlugins":  map[string]bool{"plugin1@marketplace": true},
		"unknownField":    "some value",
		"anotherNewField": map[string]interface{}{"nested": true},
		"futureFeature":   123,
	}

	data, _ := json.Marshal(originalSettings)
	settingsPath := filepath.Join(tempDir, "settings.json")
	if err := os.WriteFile(settingsPath, data, 0644); err != nil {
		t.Fatal(err)
	}

	// Load and save settings
	settings, err := LoadSettings(tempDir)
	if err != nil {
		t.Fatal(err)
	}

	settings.EnablePlugin("plugin2@marketplace")

	if err := SaveSettings(tempDir, settings); err != nil {
		t.Fatal(err)
	}

	// Verify unknown fields are preserved (at the end, alphabetically)
	savedData, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatal(err)
	}

	var savedSettings map[string]interface{}
	if err := json.Unmarshal(savedData, &savedSettings); err != nil {
		t.Fatal(err)
	}

	if savedSettings["unknownField"] != "some value" {
		t.Error("unknownField should be preserved")
	}
	if savedSettings["anotherNewField"] == nil {
		t.Error("anotherNewField should be preserved")
	}
	if savedSettings["futureFeature"] == nil {
		t.Error("futureFeature should be preserved")
	}
}

func TestSaveSettingsForScopeCanonicalKeyOrder(t *testing.T) {
	// Create temp directory
	tempDir, err := os.MkdirTemp("", "claudeup-scope-order-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	claudeDir := filepath.Join(tempDir, ".claude")
	projectDir := filepath.Join(tempDir, "project")

	// Create project scope settings with Claude Code's fields
	os.MkdirAll(filepath.Join(projectDir, ".claude"), 0755)
	originalSettings := map[string]interface{}{
		"enabledPlugins": map[string]bool{"plugin1@marketplace": true},
		"hooks":          map[string]interface{}{},
		"permissions":    map[string]interface{}{},
	}

	data, _ := json.Marshal(originalSettings)
	settingsPath := filepath.Join(projectDir, ".claude", "settings.json")
	if err := os.WriteFile(settingsPath, data, 0644); err != nil {
		t.Fatal(err)
	}

	// Load and save settings
	settings, err := LoadSettingsForScope("project", claudeDir, projectDir)
	if err != nil {
		t.Fatal(err)
	}

	settings.EnablePlugin("plugin2@marketplace")

	if err := SaveSettingsForScope("project", claudeDir, projectDir, settings); err != nil {
		t.Fatal(err)
	}

	// Read the saved file and check key order
	savedData, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatal(err)
	}

	content := string(savedData)

	// When only some fields are present, they should still be in canonical order
	permissionsIdx := indexOf(content, `"permissions"`)
	hooksIdx := indexOf(content, `"hooks"`)
	enabledPluginsIdx := indexOf(content, `"enabledPlugins"`)

	// Verify order for the fields that are present
	if permissionsIdx == -1 {
		t.Error("permissions not found in output")
	}
	if hooksIdx == -1 || hooksIdx < permissionsIdx {
		t.Error("hooks should appear after permissions")
	}
	if enabledPluginsIdx == -1 || enabledPluginsIdx < hooksIdx {
		t.Error("enabledPlugins should appear after hooks")
	}
}

func TestMergeHooks(t *testing.T) {
	// Create settings with existing hooks
	settings := &Settings{
		raw: map[string]interface{}{
			"hooks": map[string]interface{}{
				"PostToolUse": []interface{}{
					map[string]interface{}{
						"matcher": "Edit|Write",
						"hooks": []interface{}{
							map[string]interface{}{
								"type":    "command",
								"command": "~/.claude/hooks/format-on-save.sh",
							},
						},
					},
				},
			},
		},
	}

	// Merge new hooks
	newHooks := map[string][]map[string]interface{}{
		"SessionStart": {
			{"type": "command", "command": "node ~/.claude/hooks/gsd-check-update.js"},
		},
	}

	err := settings.MergeHooks(newHooks)
	if err != nil {
		t.Fatalf("MergeHooks() error = %v", err)
	}

	// Verify PostToolUse still exists
	hooks := settings.raw["hooks"].(map[string]interface{})
	if hooks["PostToolUse"] == nil {
		t.Error("PostToolUse hooks were removed")
	}

	// Verify SessionStart was added
	if hooks["SessionStart"] == nil {
		t.Error("SessionStart hooks were not added")
	}

	sessionStart := hooks["SessionStart"].([]interface{})
	if len(sessionStart) != 1 {
		t.Errorf("Expected 1 SessionStart entry, got %d", len(sessionStart))
	}
}

func TestMergeHooksDeduplicate(t *testing.T) {
	settings := &Settings{
		raw: map[string]interface{}{
			"hooks": map[string]interface{}{
				"SessionStart": []interface{}{
					map[string]interface{}{
						"hooks": []interface{}{
							map[string]interface{}{
								"type":    "command",
								"command": "node ~/.claude/hooks/existing.js",
							},
						},
					},
				},
			},
		},
	}

	// Try to add a duplicate and a new hook
	newHooks := map[string][]map[string]interface{}{
		"SessionStart": {
			{"type": "command", "command": "node ~/.claude/hooks/existing.js"}, // duplicate
			{"type": "command", "command": "node ~/.claude/hooks/new.js"},      // new
		},
	}

	err := settings.MergeHooks(newHooks)
	if err != nil {
		t.Fatalf("MergeHooks() error = %v", err)
	}

	// Count unique commands
	hooks := settings.raw["hooks"].(map[string]interface{})
	sessionStart := hooks["SessionStart"].([]interface{})

	commandSet := make(map[string]bool)
	for _, entry := range sessionStart {
		entryMap := entry.(map[string]interface{})
		hooksList := entryMap["hooks"].([]interface{})
		for _, hook := range hooksList {
			hookMap := hook.(map[string]interface{})
			if cmd, ok := hookMap["command"].(string); ok {
				commandSet[cmd] = true
			}
		}
	}

	// Should have 2 unique commands (existing + new, no duplicate)
	if len(commandSet) != 2 {
		t.Errorf("Expected 2 unique commands after dedup, got %d", len(commandSet))
	}

	if !commandSet["node ~/.claude/hooks/existing.js"] {
		t.Error("existing.js should be present")
	}
	if !commandSet["node ~/.claude/hooks/new.js"] {
		t.Error("new.js should be present")
	}
}

func TestMergeHooksEmptySettings(t *testing.T) {
	settings := &Settings{
		raw: nil, // No existing settings
	}

	newHooks := map[string][]map[string]interface{}{
		"SessionStart": {
			{"type": "command", "command": "node ~/.claude/hooks/test.js"},
		},
	}

	err := settings.MergeHooks(newHooks)
	if err != nil {
		t.Fatalf("MergeHooks() error = %v", err)
	}

	// Verify hooks were created
	if settings.raw == nil {
		t.Fatal("raw map should be initialized")
	}

	hooks := settings.raw["hooks"].(map[string]interface{})
	if hooks["SessionStart"] == nil {
		t.Error("SessionStart hooks were not added")
	}
}
