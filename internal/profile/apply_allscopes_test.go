// ABOUTME: Unit tests for ApplyAllScopes function
// ABOUTME: Tests applying multi-scope profiles to correct scope locations
package profile

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/claudeup/claudeup/v4/internal/claude"
)

func TestApplyAllScopesMultiScope(t *testing.T) {
	// Create temp directories for testing
	tempDir := t.TempDir()
	claudeDir := filepath.Join(tempDir, ".claude")
	projectDir := filepath.Join(tempDir, "project")
	claudeJSONPath := filepath.Join(claudeDir, ".claude.json")

	// Create directory structure
	mustMkdir(t, claudeDir)
	mustMkdir(t, filepath.Join(claudeDir, "plugins"))
	mustMkdir(t, projectDir)
	mustMkdir(t, filepath.Join(projectDir, ".claude"))

	// Initialize with empty settings
	mustWriteJSON(t, filepath.Join(claudeDir, "settings.json"), map[string]any{
		"enabledPlugins": map[string]bool{},
	})
	mustWriteJSON(t, filepath.Join(claudeDir, "plugins", "known_marketplaces.json"), map[string]any{})
	mustWriteJSON(t, claudeJSONPath, map[string]any{"mcpServers": map[string]any{}})

	// Create a multi-scope profile
	profile := &Profile{
		Name: "test-multi",
		PerScope: &PerScopeSettings{
			User: &ScopeSettings{
				Plugins: []string{"user-plugin@marketplace"},
			},
			Project: &ScopeSettings{
				Plugins: []string{"project-plugin@marketplace"},
			},
			Local: &ScopeSettings{
				Plugins: []string{"local-plugin@marketplace"},
			},
		},
	}

	// Apply the profile (nil opts = additive user scope by default)
	result, err := ApplyAllScopes(profile, claudeDir, claudeJSONPath, projectDir, claudeDir, nil, nil)
	if err != nil {
		t.Fatalf("ApplyAllScopes failed: %v", err)
	}

	// Verify result is non-nil
	if result == nil {
		t.Fatal("expected non-nil result")
	}

	// Verify user-scope settings were written
	userSettings, err := claude.LoadSettings(claudeDir)
	if err != nil {
		t.Fatalf("failed to load user settings: %v", err)
	}
	if !userSettings.IsPluginEnabled("user-plugin@marketplace") {
		t.Error("expected user-plugin@marketplace to be enabled at user scope")
	}
	// User scope should NOT have project/local plugins
	if userSettings.IsPluginEnabled("project-plugin@marketplace") {
		t.Error("project-plugin should not be in user scope settings")
	}
	if userSettings.IsPluginEnabled("local-plugin@marketplace") {
		t.Error("local-plugin should not be in user scope settings")
	}

	// Verify project-scope settings were written
	projectSettings, err := claude.LoadSettingsForScope("project", claudeDir, projectDir)
	if err != nil {
		t.Fatalf("failed to load project settings: %v", err)
	}
	if !projectSettings.IsPluginEnabled("project-plugin@marketplace") {
		t.Error("expected project-plugin@marketplace to be enabled at project scope")
	}

	// Verify local-scope settings were written
	localSettings, err := claude.LoadSettingsForScope("local", claudeDir, projectDir)
	if err != nil {
		t.Fatalf("failed to load local settings: %v", err)
	}
	if !localSettings.IsPluginEnabled("local-plugin@marketplace") {
		t.Error("expected local-plugin@marketplace to be enabled at local scope")
	}
}

func TestApplyAllScopesLegacyProfile(t *testing.T) {
	// Create temp directories
	tempDir := t.TempDir()
	claudeDir := filepath.Join(tempDir, ".claude")
	projectDir := filepath.Join(tempDir, "project")
	claudeJSONPath := filepath.Join(claudeDir, ".claude.json")

	// Create directory structure
	mustMkdir(t, claudeDir)
	mustMkdir(t, filepath.Join(claudeDir, "plugins"))
	mustMkdir(t, projectDir)

	// Initialize with empty settings
	mustWriteJSON(t, filepath.Join(claudeDir, "settings.json"), map[string]any{
		"enabledPlugins": map[string]bool{},
	})
	mustWriteJSON(t, filepath.Join(claudeDir, "plugins", "known_marketplaces.json"), map[string]any{})
	mustWriteJSON(t, claudeJSONPath, map[string]any{"mcpServers": map[string]any{}})

	// Create a legacy (flat) profile
	profile := &Profile{
		Name:    "test-legacy",
		Plugins: []string{"legacy-plugin@marketplace"},
	}

	// Apply the profile (nil opts = additive user scope by default)
	result, err := ApplyAllScopes(profile, claudeDir, claudeJSONPath, projectDir, claudeDir, nil, nil)
	if err != nil {
		t.Fatalf("ApplyAllScopes failed: %v", err)
	}

	if result == nil {
		t.Fatal("expected non-nil result")
	}

	// Legacy profiles should apply to user scope only
	userSettings, err := claude.LoadSettings(claudeDir)
	if err != nil {
		t.Fatalf("failed to load user settings: %v", err)
	}
	if !userSettings.IsPluginEnabled("legacy-plugin@marketplace") {
		t.Error("expected legacy-plugin@marketplace to be enabled at user scope")
	}

	// Project scope should NOT have the plugin
	projectSettingsPath := filepath.Join(projectDir, ".claude", "settings.json")
	if _, err := os.Stat(projectSettingsPath); err == nil {
		projectSettings, _ := claude.LoadSettingsForScope("project", claudeDir, projectDir)
		if projectSettings != nil && projectSettings.IsPluginEnabled("legacy-plugin@marketplace") {
			t.Error("legacy profile should not apply to project scope")
		}
	}
}

func TestApplyAllScopesPartialScopes(t *testing.T) {
	// Test profile with only some scopes populated
	tempDir := t.TempDir()
	claudeDir := filepath.Join(tempDir, ".claude")
	projectDir := filepath.Join(tempDir, "project")
	claudeJSONPath := filepath.Join(claudeDir, ".claude.json")

	mustMkdir(t, claudeDir)
	mustMkdir(t, filepath.Join(claudeDir, "plugins"))
	mustMkdir(t, projectDir)

	mustWriteJSON(t, filepath.Join(claudeDir, "settings.json"), map[string]any{
		"enabledPlugins": map[string]bool{},
	})
	mustWriteJSON(t, filepath.Join(claudeDir, "plugins", "known_marketplaces.json"), map[string]any{})
	mustWriteJSON(t, claudeJSONPath, map[string]any{"mcpServers": map[string]any{}})

	// Profile with only user and project scope (no local)
	profile := &Profile{
		Name: "partial",
		PerScope: &PerScopeSettings{
			User: &ScopeSettings{
				Plugins: []string{"user-only@marketplace"},
			},
			Project: &ScopeSettings{
				Plugins: []string{"project-only@marketplace"},
			},
			// Local is nil
		},
	}

	result, err := ApplyAllScopes(profile, claudeDir, claudeJSONPath, projectDir, claudeDir, nil, nil)
	if err != nil {
		t.Fatalf("ApplyAllScopes failed: %v", err)
	}

	if result == nil {
		t.Fatal("expected non-nil result")
	}

	// User scope should have its plugin
	userSettings, err := claude.LoadSettings(claudeDir)
	if err != nil {
		t.Fatalf("failed to load user settings: %v", err)
	}
	if !userSettings.IsPluginEnabled("user-only@marketplace") {
		t.Error("expected user-only@marketplace to be enabled")
	}

	// Project scope should have its plugin
	projectSettings, err := claude.LoadSettingsForScope("project", claudeDir, projectDir)
	if err != nil {
		t.Fatalf("failed to load project settings: %v", err)
	}
	if !projectSettings.IsPluginEnabled("project-only@marketplace") {
		t.Error("expected project-only@marketplace to be enabled")
	}
}

func TestApplyAllScopesPreservesExistingSettings(t *testing.T) {
	// Test that applying doesn't wipe out other settings or existing plugins (additive behavior)
	tempDir := t.TempDir()
	claudeDir := filepath.Join(tempDir, ".claude")
	projectDir := filepath.Join(tempDir, "project")
	claudeJSONPath := filepath.Join(claudeDir, ".claude.json")

	mustMkdir(t, claudeDir)
	mustMkdir(t, filepath.Join(claudeDir, "plugins"))
	mustMkdir(t, projectDir)

	// Pre-existing settings with other fields and existing plugins
	existingSettings := map[string]any{
		"enabledPlugins": map[string]bool{
			"existing-plugin@marketplace": true,
		},
		"someOtherSetting": "should-be-preserved",
	}
	mustWriteJSON(t, filepath.Join(claudeDir, "settings.json"), existingSettings)
	mustWriteJSON(t, filepath.Join(claudeDir, "plugins", "known_marketplaces.json"), map[string]any{})
	mustWriteJSON(t, claudeJSONPath, map[string]any{"mcpServers": map[string]any{}})

	profile := &Profile{
		Name: "new-profile",
		PerScope: &PerScopeSettings{
			User: &ScopeSettings{
				Plugins: []string{"new-plugin@marketplace"},
			},
		},
	}

	// Apply with default options (additive for user scope)
	_, err := ApplyAllScopes(profile, claudeDir, claudeJSONPath, projectDir, claudeDir, nil, nil)
	if err != nil {
		t.Fatalf("ApplyAllScopes failed: %v", err)
	}

	// Load raw settings to check other fields
	data, err := os.ReadFile(filepath.Join(claudeDir, "settings.json"))
	if err != nil {
		t.Fatalf("failed to read settings: %v", err)
	}

	var settings map[string]any
	if err := json.Unmarshal(data, &settings); err != nil {
		t.Fatalf("failed to parse settings: %v", err)
	}

	// Check that other settings are preserved
	if settings["someOtherSetting"] != "should-be-preserved" {
		t.Error("expected someOtherSetting to be preserved")
	}

	// Check that existing plugins are preserved (additive behavior)
	enabledPlugins := settings["enabledPlugins"].(map[string]any)
	if enabledPlugins["existing-plugin@marketplace"] != true {
		t.Error("expected existing-plugin@marketplace to be preserved (additive behavior)")
	}
	if enabledPlugins["new-plugin@marketplace"] != true {
		t.Error("expected new-plugin@marketplace to be added")
	}
}

func TestApplyAllScopesReplaceUserScope(t *testing.T) {
	// Test that ReplaceUserScope option removes existing plugins
	tempDir := t.TempDir()
	claudeDir := filepath.Join(tempDir, ".claude")
	projectDir := filepath.Join(tempDir, "project")
	claudeJSONPath := filepath.Join(claudeDir, ".claude.json")

	mustMkdir(t, claudeDir)
	mustMkdir(t, filepath.Join(claudeDir, "plugins"))
	mustMkdir(t, projectDir)

	// Pre-existing settings with existing plugins
	existingSettings := map[string]any{
		"enabledPlugins": map[string]bool{
			"existing-plugin@marketplace":  true,
			"another-existing@marketplace": true,
		},
	}
	mustWriteJSON(t, filepath.Join(claudeDir, "settings.json"), existingSettings)
	mustWriteJSON(t, filepath.Join(claudeDir, "plugins", "known_marketplaces.json"), map[string]any{})
	mustWriteJSON(t, claudeJSONPath, map[string]any{"mcpServers": map[string]any{}})

	profile := &Profile{
		Name: "new-profile",
		PerScope: &PerScopeSettings{
			User: &ScopeSettings{
				Plugins: []string{"new-plugin@marketplace"},
			},
		},
	}

	// Apply with ReplaceUserScope = true
	opts := &ApplyAllScopesOptions{
		ReplaceUserScope: true,
	}
	_, err := ApplyAllScopes(profile, claudeDir, claudeJSONPath, projectDir, claudeDir, nil, opts)
	if err != nil {
		t.Fatalf("ApplyAllScopes failed: %v", err)
	}

	// Load settings
	userSettings, err := claude.LoadSettings(claudeDir)
	if err != nil {
		t.Fatalf("failed to load settings: %v", err)
	}

	// Existing plugins should be removed (replace behavior)
	if userSettings.IsPluginEnabled("existing-plugin@marketplace") {
		t.Error("expected existing-plugin@marketplace to be removed (replace behavior)")
	}
	if userSettings.IsPluginEnabled("another-existing@marketplace") {
		t.Error("expected another-existing@marketplace to be removed (replace behavior)")
	}

	// New plugin should be present
	if !userSettings.IsPluginEnabled("new-plugin@marketplace") {
		t.Error("expected new-plugin@marketplace to be enabled")
	}
}
