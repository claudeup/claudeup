// ABOUTME: Unit tests for scope helper functions (clearScope and RenderPluginsByScope)
// ABOUTME: Covers absent settings.json, non-existent file removal, and error propagation
package commands

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ---- clearScope tests ----

func TestClearScope_UserScope_AbsentSettingsJSON(t *testing.T) {
	// claudeDir exists but settings.json is absent: should succeed and create the file
	claudeDir := t.TempDir()
	settingsPath := filepath.Join(claudeDir, "settings.json")

	if err := clearScope("user", settingsPath, claudeDir); err != nil {
		t.Fatalf("clearScope(user) with absent settings.json: unexpected error: %v", err)
	}

	// settings.json must now exist
	if _, err := os.Stat(settingsPath); os.IsNotExist(err) {
		t.Error("expected settings.json to be created after clearScope(user)")
	}
}

func TestClearScope_UserScope_ClearsExistingPlugins(t *testing.T) {
	claudeDir := t.TempDir()
	settingsPath := filepath.Join(claudeDir, "settings.json")

	// Write a settings file that has two plugins enabled
	initial := `{"enabledPlugins":{"plugin-a":true,"plugin-b":true}}`
	if err := os.WriteFile(settingsPath, []byte(initial), 0644); err != nil {
		t.Fatal(err)
	}

	if err := clearScope("user", settingsPath, claudeDir); err != nil {
		t.Fatalf("clearScope(user): %v", err)
	}

	data, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatal(err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("result is not valid JSON: %v", err)
	}

	if plugins, ok := raw["enabledPlugins"].(map[string]interface{}); ok && len(plugins) != 0 {
		t.Errorf("expected enabledPlugins to be empty, got %v", plugins)
	}
}

func TestClearScope_ProjectScope_NonExistentFile(t *testing.T) {
	// Removing a non-existent project settings file should succeed silently
	claudeDir := t.TempDir()
	projectDir := t.TempDir()
	settingsPath := filepath.Join(projectDir, ".claude", "settings.json")

	if err := clearScope("project", settingsPath, claudeDir); err != nil {
		t.Errorf("clearScope(project) with non-existent file: unexpected error: %v", err)
	}
}

func TestClearScope_LocalScope_NonExistentFile(t *testing.T) {
	// Removing a non-existent local settings file should succeed silently
	claudeDir := t.TempDir()
	projectDir := t.TempDir()
	settingsPath := filepath.Join(projectDir, ".claude", "settings.local.json")

	if err := clearScope("local", settingsPath, claudeDir); err != nil {
		t.Errorf("clearScope(local) with non-existent file: unexpected error: %v", err)
	}
}

func TestClearScope_ProjectScope_RemovesExistingFile(t *testing.T) {
	claudeDir := t.TempDir()
	projectDir := t.TempDir()
	claudeSubDir := filepath.Join(projectDir, ".claude")
	if err := os.MkdirAll(claudeSubDir, 0755); err != nil {
		t.Fatal(err)
	}
	settingsPath := filepath.Join(claudeSubDir, "settings.json")
	if err := os.WriteFile(settingsPath, []byte(`{"enabledPlugins":{}}`), 0644); err != nil {
		t.Fatal(err)
	}

	if err := clearScope("project", settingsPath, claudeDir); err != nil {
		t.Fatalf("clearScope(project): %v", err)
	}

	if _, err := os.Stat(settingsPath); !os.IsNotExist(err) {
		t.Error("expected settings.json to be removed after clearScope(project)")
	}
}

func TestClearScope_LocalScope_RemovesExistingFile(t *testing.T) {
	claudeDir := t.TempDir()
	projectDir := t.TempDir()
	claudeSubDir := filepath.Join(projectDir, ".claude")
	if err := os.MkdirAll(claudeSubDir, 0755); err != nil {
		t.Fatal(err)
	}
	settingsPath := filepath.Join(claudeSubDir, "settings.local.json")
	if err := os.WriteFile(settingsPath, []byte(`{"enabledPlugins":{}}`), 0644); err != nil {
		t.Fatal(err)
	}

	if err := clearScope("local", settingsPath, claudeDir); err != nil {
		t.Fatalf("clearScope(local): %v", err)
	}

	if _, err := os.Stat(settingsPath); !os.IsNotExist(err) {
		t.Error("expected settings.local.json to be removed after clearScope(local)")
	}
}

func TestClearScope_InvalidScope_ReturnsError(t *testing.T) {
	claudeDir := t.TempDir()

	err := clearScope("bogus", "", claudeDir)
	if err == nil {
		t.Fatal("expected error for invalid scope, got nil")
	}
	if !strings.Contains(err.Error(), "invalid scope") {
		t.Errorf("expected 'invalid scope' in error message, got: %v", err)
	}
}

func TestClearScope_UserScope_CorruptJSON_PropagatesError(t *testing.T) {
	// If settings.json contains corrupt JSON, LoadSettings returns an error that
	// is not fs.ErrNotExist, so LoadSettingsOrEmpty propagates it to the caller.
	claudeDir := t.TempDir()
	settingsPath := filepath.Join(claudeDir, "settings.json")
	if err := os.WriteFile(settingsPath, []byte("not valid json {{{"), 0644); err != nil {
		t.Fatal(err)
	}

	err := clearScope("user", settingsPath, claudeDir)
	if err == nil {
		t.Error("expected error propagation for corrupt settings.json, got nil")
	}
}

// ---- RenderPluginsByScope tests ----

func TestRenderPluginsByScope_InvalidScope_ReturnsError(t *testing.T) {
	claudeDir := t.TempDir()
	projectDir := t.TempDir()

	err := RenderPluginsByScope(claudeDir, projectDir, "invalid")
	if err == nil {
		t.Fatal("expected error for invalid scope, got nil")
	}
}

func TestRenderPluginsByScope_UserScope_NoSettings(t *testing.T) {
	// User scope with no settings.json present should succeed (empty plugin list)
	claudeDir := t.TempDir()
	projectDir := t.TempDir()

	if err := RenderPluginsByScope(claudeDir, projectDir, "user"); err != nil {
		t.Errorf("RenderPluginsByScope(user) with no settings: unexpected error: %v", err)
	}
}

func TestRenderPluginsByScope_UserScope_WithPlugins(t *testing.T) {
	claudeDir := t.TempDir()
	projectDir := t.TempDir()

	content := `{"enabledPlugins":{"plugin-a":true,"plugin-b":false}}`
	if err := os.WriteFile(filepath.Join(claudeDir, "settings.json"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	if err := RenderPluginsByScope(claudeDir, projectDir, "user"); err != nil {
		t.Errorf("RenderPluginsByScope(user) with plugins: unexpected error: %v", err)
	}
}

func TestRenderPluginsByScope_AllScopes_OutsideProjectDir(t *testing.T) {
	// No .claude subdirectory → project/local scopes are skipped, should not error
	claudeDir := t.TempDir()
	projectDir := t.TempDir() // no .claude directory inside

	if err := RenderPluginsByScope(claudeDir, projectDir, ""); err != nil {
		t.Errorf("RenderPluginsByScope(all) outside project dir: unexpected error: %v", err)
	}
}

func TestRenderPluginsByScope_AllScopes_InsideProjectDir(t *testing.T) {
	claudeDir := t.TempDir()
	projectDir := t.TempDir()

	// Simulate being inside a project by creating a .claude subdirectory
	if err := os.MkdirAll(filepath.Join(projectDir, ".claude"), 0755); err != nil {
		t.Fatal(err)
	}

	if err := RenderPluginsByScope(claudeDir, projectDir, ""); err != nil {
		t.Errorf("RenderPluginsByScope(all) inside project dir: unexpected error: %v", err)
	}
}

func TestRenderPluginsByScope_ProjectScope_EmptyProjectDir_ReturnsError(t *testing.T) {
	// Requesting project scope with an empty projectDir should propagate the error
	// from SettingsPathForScope("project", claudeDir, "").
	claudeDir := t.TempDir()

	err := RenderPluginsByScope(claudeDir, "", "project")
	if err == nil {
		t.Fatal("expected error for project scope with empty projectDir, got nil")
	}
}

func TestRenderPluginsByScope_LocalScope_EmptyProjectDir_ReturnsError(t *testing.T) {
	claudeDir := t.TempDir()

	err := RenderPluginsByScope(claudeDir, "", "local")
	if err == nil {
		t.Fatal("expected error for local scope with empty projectDir, got nil")
	}
}
