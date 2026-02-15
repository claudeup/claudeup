// ABOUTME: Tests for enabled.json config loading and saving
// ABOUTME: Verifies round-trip serialization and default behavior
package ext

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfigNonexistent(t *testing.T) {
	claudeDir := t.TempDir()
	claudeupHome := t.TempDir()
	manager := NewManager(claudeDir, claudeupHome)

	config, err := manager.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	// Should return empty config, not error
	if config == nil {
		t.Fatal("LoadConfig() returned nil config")
	}
	if len(config) != 0 {
		t.Errorf("LoadConfig() returned non-empty config: %v", config)
	}
}

func TestConfigRoundTrip(t *testing.T) {
	claudeDir := t.TempDir()
	claudeupHome := t.TempDir()
	manager := NewManager(claudeDir, claudeupHome)

	// Create test config
	config := Config{
		"agents": {
			"gsd-planner.md":  true,
			"gsd-executor.md": false,
		},
		"commands": {
			"gsd/new-project.md": true,
		},
	}

	// Save
	if err := manager.SaveConfig(config); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	// Verify file exists in claudeupHome, NOT claudeDir
	configPath := filepath.Join(claudeupHome, "enabled.json")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Fatal("enabled.json was not created in claudeupHome")
	}

	// Verify file does NOT exist in claudeDir
	wrongPath := filepath.Join(claudeDir, "enabled.json")
	if _, err := os.Stat(wrongPath); !os.IsNotExist(err) {
		t.Fatal("enabled.json should NOT be in claudeDir")
	}

	// Load back
	loaded, err := manager.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	// Verify contents
	if !loaded["agents"]["gsd-planner.md"] {
		t.Error("agents/gsd-planner.md should be true")
	}
	if loaded["agents"]["gsd-executor.md"] {
		t.Error("agents/gsd-executor.md should be false")
	}
	if !loaded["commands"]["gsd/new-project.md"] {
		t.Error("commands/gsd/new-project.md should be true")
	}
}

func TestManagerMigratesOldDirectory(t *testing.T) {
	claudeDir := t.TempDir()
	claudeupHome := t.TempDir()

	// Create the old "local" directory with content
	oldDir := filepath.Join(claudeupHome, "local", "agents")
	os.MkdirAll(oldDir, 0755)
	os.WriteFile(filepath.Join(oldDir, "my-agent.md"), []byte("# Agent"), 0644)

	// Create manager -- should auto-migrate
	manager := NewManager(claudeDir, claudeupHome)

	// Old directory should be gone
	if _, err := os.Stat(filepath.Join(claudeupHome, "local")); !os.IsNotExist(err) {
		t.Error("Old 'local' directory should have been removed after migration")
	}

	// New directory should exist with the content
	newPath := filepath.Join(claudeupHome, "ext", "agents", "my-agent.md")
	if _, err := os.Stat(newPath); os.IsNotExist(err) {
		t.Error("Content should be in new 'ext' directory after migration")
	}

	// Manager should use new directory
	items, err := manager.ListItems("agents")
	if err != nil {
		t.Fatalf("ListItems() error = %v", err)
	}
	if len(items) != 1 {
		t.Errorf("ListItems() = %d, want 1", len(items))
	}
}

func TestManagerSkipsMigrationWhenExtExists(t *testing.T) {
	claudeDir := t.TempDir()
	claudeupHome := t.TempDir()

	// Create both old and new directories
	oldDir := filepath.Join(claudeupHome, "local", "agents")
	os.MkdirAll(oldDir, 0755)
	os.WriteFile(filepath.Join(oldDir, "old-agent.md"), []byte("# Old"), 0644)

	newDir := filepath.Join(claudeupHome, "ext", "agents")
	os.MkdirAll(newDir, 0755)
	os.WriteFile(filepath.Join(newDir, "new-agent.md"), []byte("# New"), 0644)

	// Create manager -- should NOT overwrite ext
	_ = NewManager(claudeDir, claudeupHome)

	// Both directories should still exist
	if _, err := os.Stat(filepath.Join(claudeupHome, "local")); os.IsNotExist(err) {
		t.Error("Old 'local' directory should be preserved when 'ext' already exists")
	}
	if _, err := os.Stat(filepath.Join(claudeupHome, "ext", "agents", "new-agent.md")); os.IsNotExist(err) {
		t.Error("New 'ext' content should be preserved")
	}
}
