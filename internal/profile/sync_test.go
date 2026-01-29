// ABOUTME: Unit tests for profile sync functionality
// ABOUTME: Tests sync from explicit profile name using ApplyAllScopes
package profile

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Helper to create a test profile and return profilesDir
func setupTestProfile(t *testing.T, projectDir string, plugins []string, marketplaces []Marketplace) string {
	profilesDir := filepath.Join(projectDir, "profiles")
	if err := os.MkdirAll(profilesDir, 0755); err != nil {
		t.Fatalf("failed to create profiles dir: %v", err)
	}

	testProfile := &Profile{
		Name:         "test-profile",
		Plugins:      plugins,
		Marketplaces: marketplaces,
	}
	if err := Save(profilesDir, testProfile); err != nil {
		t.Fatalf("failed to save test profile: %v", err)
	}

	return profilesDir
}

// Helper to set up Claude directories
func setupClaudeDir(t *testing.T) (claudeDir, claudeJSONPath string) {
	claudeDir = filepath.Join(t.TempDir(), ".claude")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatalf("failed to create claude dir: %v", err)
	}
	claudeJSONPath = filepath.Join(claudeDir, ".claude.json")
	return claudeDir, claudeJSONPath
}

func TestSync_NoProfileName(t *testing.T) {
	projectDir := t.TempDir()
	claudeDir, claudeJSONPath := setupClaudeDir(t)

	profilesDir := filepath.Join(projectDir, "profiles")
	os.MkdirAll(profilesDir, 0755)

	_, err := Sync(profilesDir, projectDir, claudeDir, claudeJSONPath, "", SyncOptions{})
	if err == nil {
		t.Error("expected error for missing profile name")
	}
	if !strings.Contains(err.Error(), "profile name not specified") {
		t.Errorf("error should mention profile name: %v", err)
	}
}

func TestSync_ProfileNotFound(t *testing.T) {
	projectDir := t.TempDir()
	claudeDir, claudeJSONPath := setupClaudeDir(t)
	profilesDir := filepath.Join(projectDir, "profiles")
	os.MkdirAll(profilesDir, 0755)

	_, err := Sync(profilesDir, projectDir, claudeDir, claudeJSONPath, "nonexistent-profile", SyncOptions{})
	if err == nil {
		t.Error("expected error for missing profile")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error should mention not found: %v", err)
	}
}

func TestSync_DryRun(t *testing.T) {
	projectDir := t.TempDir()
	claudeDir, claudeJSONPath := setupClaudeDir(t)

	// Create test profile with plugins
	profilesDir := setupTestProfile(t, projectDir,
		[]string{"plugin-a@test", "plugin-b@test"},
		[]Marketplace{{Source: "github", Repo: "test/plugins"}})

	result, err := Sync(profilesDir, projectDir, claudeDir, claudeJSONPath, "test-profile", SyncOptions{DryRun: true})
	if err != nil {
		t.Fatalf("Sync dry run failed: %v", err)
	}

	// Dry run should report what would be installed
	if result.PluginsInstalled != 2 {
		t.Errorf("PluginsInstalled = %d, want 2", result.PluginsInstalled)
	}

	// Settings file should NOT exist (dry run doesn't write)
	settingsPath := filepath.Join(claudeDir, "settings.json")
	if _, err := os.Stat(settingsPath); err == nil {
		t.Error("settings.json should not exist in dry run mode")
	}
}

func TestSync_CreatesLocalProfileCopy(t *testing.T) {
	projectDir := t.TempDir()
	claudeDir, claudeJSONPath := setupClaudeDir(t)

	// Create profile in project directory (simulating team profile)
	projectProfilesDir := filepath.Join(projectDir, ".claudeup", "profiles")
	os.MkdirAll(projectProfilesDir, 0755)

	projectProfile := &Profile{
		Name:    "team-profile",
		Plugins: []string{"team-plugin@marketplace"},
	}
	if err := Save(projectProfilesDir, projectProfile); err != nil {
		t.Fatal(err)
	}

	// User profiles directory (where local copy will be saved)
	userProfilesDir := filepath.Join(t.TempDir(), "user-profiles")
	os.MkdirAll(userProfilesDir, 0755)

	result, err := Sync(userProfilesDir, projectDir, claudeDir, claudeJSONPath, "team-profile", SyncOptions{})
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}

	// Check local profile copy was created
	if !result.ProfileCreated {
		t.Error("ProfileCreated should be true")
	}

	// Verify the profile exists in user profiles directory
	localProfilePath := filepath.Join(userProfilesDir, "team-profile.json")
	if _, err := os.Stat(localProfilePath); os.IsNotExist(err) {
		t.Errorf("Local profile copy not created at %s", localProfilePath)
	}
}

func TestSync_AppliesPluginsToSettings(t *testing.T) {
	projectDir := t.TempDir()
	claudeDir, claudeJSONPath := setupClaudeDir(t)

	// Create test profile with plugins
	profilesDir := setupTestProfile(t, projectDir,
		[]string{"plugin-a@test", "plugin-b@test"},
		nil)

	result, err := Sync(profilesDir, projectDir, claudeDir, claudeJSONPath, "test-profile", SyncOptions{})
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}

	if result.ProfileName != "test-profile" {
		t.Errorf("ProfileName = %s, want test-profile", result.ProfileName)
	}

	// Check user settings file was created with plugins
	userSettingsPath := filepath.Join(claudeDir, "settings.json")
	data, err := os.ReadFile(userSettingsPath)
	if err != nil {
		t.Fatalf("Failed to read user settings: %v", err)
	}

	var settings map[string]interface{}
	if err := json.Unmarshal(data, &settings); err != nil {
		t.Fatalf("Failed to parse settings: %v", err)
	}

	enabledPlugins, ok := settings["enabledPlugins"].(map[string]interface{})
	if !ok {
		t.Fatal("enabledPlugins not found in settings")
	}

	if _, ok := enabledPlugins["plugin-a@test"]; !ok {
		t.Error("plugin-a@test not found in settings")
	}
	if _, ok := enabledPlugins["plugin-b@test"]; !ok {
		t.Error("plugin-b@test not found in settings")
	}
}

func TestSync_MultiScopeProfile(t *testing.T) {
	projectDir := t.TempDir()
	claudeDir, claudeJSONPath := setupClaudeDir(t)

	// Create a multi-scope profile
	profilesDir := filepath.Join(projectDir, "profiles")
	os.MkdirAll(profilesDir, 0755)

	multiProfile := &Profile{
		Name: "multi-scope-test",
		PerScope: &PerScopeSettings{
			User: &ScopeSettings{
				Plugins: []string{"user-plugin@test"},
			},
			Project: &ScopeSettings{
				Plugins: []string{"project-plugin@test"},
			},
		},
	}
	if err := Save(profilesDir, multiProfile); err != nil {
		t.Fatal(err)
	}

	// Create project .claude directory
	projectClaudeDir := filepath.Join(projectDir, ".claude")
	os.MkdirAll(projectClaudeDir, 0755)

	result, err := Sync(profilesDir, projectDir, claudeDir, claudeJSONPath, "multi-scope-test", SyncOptions{})
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}

	// In unit tests without the claude CLI, plugins won't actually install
	// (they'll result in errors), but settings should still be written.
	// The important thing is that sync attempted to process all plugins.
	totalAttempted := result.PluginsInstalled + result.PluginsSkipped + len(result.Errors)
	if totalAttempted < 2 {
		t.Errorf("Total plugins attempted = %d (installed=%d, skipped=%d, errors=%d), want at least 2",
			totalAttempted, result.PluginsInstalled, result.PluginsSkipped, len(result.Errors))
	}

	// Check user settings
	userSettingsPath := filepath.Join(claudeDir, "settings.json")
	userData, err := os.ReadFile(userSettingsPath)
	if err != nil {
		t.Fatalf("Failed to read user settings: %v", err)
	}
	var userSettings map[string]interface{}
	json.Unmarshal(userData, &userSettings)
	userPlugins := userSettings["enabledPlugins"].(map[string]interface{})
	if _, ok := userPlugins["user-plugin@test"]; !ok {
		t.Error("user-plugin@test not found in user settings")
	}

	// Check project settings
	projectSettingsPath := filepath.Join(projectDir, ".claude", "settings.json")
	projectData, err := os.ReadFile(projectSettingsPath)
	if err != nil {
		t.Fatalf("Failed to read project settings: %v", err)
	}
	var projectSettings map[string]interface{}
	json.Unmarshal(projectData, &projectSettings)
	projectPlugins := projectSettings["enabledPlugins"].(map[string]interface{})
	if _, ok := projectPlugins["project-plugin@test"]; !ok {
		t.Error("project-plugin@test not found in project settings")
	}
}

func TestSync_ReplaceUserScope(t *testing.T) {
	projectDir := t.TempDir()
	claudeDir, claudeJSONPath := setupClaudeDir(t)

	// Create existing user settings with a plugin
	existingSettings := map[string]interface{}{
		"enabledPlugins": map[string]interface{}{
			"existing-plugin@test": true,
		},
	}
	existingData, _ := json.Marshal(existingSettings)
	os.WriteFile(filepath.Join(claudeDir, "settings.json"), existingData, 0644)

	// Create test profile with different plugins
	profilesDir := setupTestProfile(t, projectDir,
		[]string{"new-plugin@test"},
		nil)

	// Sync WITHOUT replace - should be additive
	_, err := Sync(profilesDir, projectDir, claudeDir, claudeJSONPath, "test-profile", SyncOptions{
		ReplaceUserScope: false,
	})
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}

	// Check both plugins exist (additive)
	data, _ := os.ReadFile(filepath.Join(claudeDir, "settings.json"))
	var settings map[string]interface{}
	json.Unmarshal(data, &settings)
	plugins := settings["enabledPlugins"].(map[string]interface{})

	if _, ok := plugins["existing-plugin@test"]; !ok {
		t.Error("existing-plugin@test should be preserved (additive mode)")
	}
	if _, ok := plugins["new-plugin@test"]; !ok {
		t.Error("new-plugin@test should be added")
	}
}

func TestSync_ReplaceUserScope_Declarative(t *testing.T) {
	projectDir := t.TempDir()
	claudeDir, claudeJSONPath := setupClaudeDir(t)

	// Create existing user settings with a plugin
	existingSettings := map[string]interface{}{
		"enabledPlugins": map[string]interface{}{
			"existing-plugin@test": true,
		},
	}
	existingData, _ := json.Marshal(existingSettings)
	os.WriteFile(filepath.Join(claudeDir, "settings.json"), existingData, 0644)

	// Create test profile with different plugins
	profilesDir := setupTestProfile(t, projectDir,
		[]string{"new-plugin@test"},
		nil)

	// Sync WITH replace - should be declarative
	_, err := Sync(profilesDir, projectDir, claudeDir, claudeJSONPath, "test-profile", SyncOptions{
		ReplaceUserScope: true,
	})
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}

	// Check only new plugin exists (replace mode)
	data, _ := os.ReadFile(filepath.Join(claudeDir, "settings.json"))
	var settings map[string]interface{}
	json.Unmarshal(data, &settings)
	plugins := settings["enabledPlugins"].(map[string]interface{})

	if _, ok := plugins["existing-plugin@test"]; ok {
		t.Error("existing-plugin@test should be removed (replace mode)")
	}
	if _, ok := plugins["new-plugin@test"]; !ok {
		t.Error("new-plugin@test should be present")
	}
}

func TestSync_EmptyProfilesDir(t *testing.T) {
	projectDir := t.TempDir()
	claudeDir, claudeJSONPath := setupClaudeDir(t)

	_, err := Sync("", projectDir, claudeDir, claudeJSONPath, "test-profile", SyncOptions{})
	if err == nil {
		t.Error("expected error for empty profiles directory")
	}
	if !strings.Contains(err.Error(), "profiles directory not specified") {
		t.Errorf("unexpected error: %v", err)
	}
}
