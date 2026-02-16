package commands

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetUntrackedScopes(t *testing.T) {
	t.Run("returns empty when no project settings exist", func(t *testing.T) {
		cwd := t.TempDir()
		claudeDir := t.TempDir()
		result := getUntrackedScopes(cwd, claudeDir, nil)
		if len(result) != 0 {
			t.Errorf("expected 0 untracked scopes, got %d", len(result))
		}
	})

	t.Run("detects project scope with enabled plugins", func(t *testing.T) {
		cwd := t.TempDir()
		claudeDir := t.TempDir()

		claudeSubdir := filepath.Join(cwd, ".claude")
		if err := os.MkdirAll(claudeSubdir, 0755); err != nil {
			t.Fatal(err)
		}
		settingsJSON := `{"enabledPlugins":{"plugin-a@marketplace":true,"plugin-b@marketplace":true}}`
		if err := os.WriteFile(filepath.Join(claudeSubdir, "settings.json"), []byte(settingsJSON), 0644); err != nil {
			t.Fatal(err)
		}

		result := getUntrackedScopes(cwd, claudeDir, nil)
		if len(result) != 1 {
			t.Fatalf("expected 1 untracked scope, got %d", len(result))
		}
		if result[0].Scope != "project" {
			t.Errorf("expected scope 'project', got %q", result[0].Scope)
		}
		if result[0].PluginCount != 2 {
			t.Errorf("expected 2 plugins, got %d", result[0].PluginCount)
		}
		if result[0].SettingsFile != ".claude/settings.json" {
			t.Errorf("expected settings file '.claude/settings.json', got %q", result[0].SettingsFile)
		}
	})

	t.Run("detects local scope with enabled plugins", func(t *testing.T) {
		cwd := t.TempDir()
		claudeDir := t.TempDir()

		claudeSubdir := filepath.Join(cwd, ".claude")
		if err := os.MkdirAll(claudeSubdir, 0755); err != nil {
			t.Fatal(err)
		}
		settingsJSON := `{"enabledPlugins":{"plugin-x@marketplace":true}}`
		if err := os.WriteFile(filepath.Join(claudeSubdir, "settings.local.json"), []byte(settingsJSON), 0644); err != nil {
			t.Fatal(err)
		}

		result := getUntrackedScopes(cwd, claudeDir, nil)
		if len(result) != 1 {
			t.Fatalf("expected 1 untracked scope, got %d", len(result))
		}
		if result[0].Scope != "local" {
			t.Errorf("expected scope 'local', got %q", result[0].Scope)
		}
		if result[0].SettingsFile != ".claude/settings.local.json" {
			t.Errorf("expected settings file '.claude/settings.local.json', got %q", result[0].SettingsFile)
		}
	})

	t.Run("skips scope when profile is tracked there", func(t *testing.T) {
		cwd := t.TempDir()
		claudeDir := t.TempDir()

		claudeSubdir := filepath.Join(cwd, ".claude")
		if err := os.MkdirAll(claudeSubdir, 0755); err != nil {
			t.Fatal(err)
		}
		settingsJSON := `{"enabledPlugins":{"plugin-a@marketplace":true}}`
		if err := os.WriteFile(filepath.Join(claudeSubdir, "settings.json"), []byte(settingsJSON), 0644); err != nil {
			t.Fatal(err)
		}

		tracked := []ActiveProfileInfo{{Name: "team-profile", Scope: "project"}}
		result := getUntrackedScopes(cwd, claudeDir, tracked)
		if len(result) != 0 {
			t.Errorf("expected 0 untracked scopes when project is tracked, got %d", len(result))
		}
	})

	t.Run("skips scope when plugins are all disabled", func(t *testing.T) {
		cwd := t.TempDir()
		claudeDir := t.TempDir()

		claudeSubdir := filepath.Join(cwd, ".claude")
		if err := os.MkdirAll(claudeSubdir, 0755); err != nil {
			t.Fatal(err)
		}
		settingsJSON := `{"enabledPlugins":{"plugin-a@marketplace":false}}`
		if err := os.WriteFile(filepath.Join(claudeSubdir, "settings.json"), []byte(settingsJSON), 0644); err != nil {
			t.Fatal(err)
		}

		result := getUntrackedScopes(cwd, claudeDir, nil)
		if len(result) != 0 {
			t.Errorf("expected 0 untracked scopes when all plugins disabled, got %d", len(result))
		}
	})

	t.Run("detects both project and local simultaneously", func(t *testing.T) {
		cwd := t.TempDir()
		claudeDir := t.TempDir()

		claudeSubdir := filepath.Join(cwd, ".claude")
		if err := os.MkdirAll(claudeSubdir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(claudeSubdir, "settings.json"), []byte(`{"enabledPlugins":{"plugin-a@marketplace":true}}`), 0644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(claudeSubdir, "settings.local.json"), []byte(`{"enabledPlugins":{"plugin-b@marketplace":true}}`), 0644); err != nil {
			t.Fatal(err)
		}

		result := getUntrackedScopes(cwd, claudeDir, nil)
		if len(result) != 2 {
			t.Fatalf("expected 2 untracked scopes, got %d", len(result))
		}
		if result[0].Scope != "project" {
			t.Errorf("expected first scope 'project', got %q", result[0].Scope)
		}
		if result[1].Scope != "local" {
			t.Errorf("expected second scope 'local', got %q", result[1].Scope)
		}
	})
}
