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
