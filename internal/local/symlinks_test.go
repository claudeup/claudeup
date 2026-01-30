// ABOUTME: Tests for symlink-based enable/disable operations
// ABOUTME: Verifies symlink creation, removal, and sync behavior
package local

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnableDisable(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewManager(tmpDir)

	// Create library structure
	libraryDir := filepath.Join(tmpDir, ".library")
	hooksDir := filepath.Join(libraryDir, "hooks")
	os.MkdirAll(hooksDir, 0755)
	os.WriteFile(filepath.Join(hooksDir, "format-on-save.sh"), []byte("#!/bin/bash"), 0644)

	// Enable
	enabled, notFound, err := manager.Enable("hooks", []string{"format-on-save"})
	if err != nil {
		t.Fatalf("Enable() error = %v", err)
	}
	if len(enabled) != 1 || enabled[0] != "format-on-save.sh" {
		t.Errorf("Enable() enabled = %v, want [format-on-save.sh]", enabled)
	}
	if len(notFound) != 0 {
		t.Errorf("Enable() notFound = %v, want []", notFound)
	}

	// Verify symlink exists
	targetDir := filepath.Join(tmpDir, "hooks")
	symlinkPath := filepath.Join(targetDir, "format-on-save.sh")
	if _, err := os.Lstat(symlinkPath); os.IsNotExist(err) {
		t.Error("Symlink was not created")
	}

	// Verify config was updated
	config, _ := manager.LoadConfig()
	if !config["hooks"]["format-on-save.sh"] {
		t.Error("Config was not updated")
	}

	// Disable
	disabled, notFound, err := manager.Disable("hooks", []string{"format-on-save"})
	if err != nil {
		t.Fatalf("Disable() error = %v", err)
	}
	if len(disabled) != 1 {
		t.Errorf("Disable() disabled = %v, want [format-on-save.sh]", disabled)
	}

	// Verify symlink removed
	if _, err := os.Lstat(symlinkPath); !os.IsNotExist(err) {
		t.Error("Symlink was not removed")
	}

	// Verify config was updated
	config, _ = manager.LoadConfig()
	if config["hooks"]["format-on-save.sh"] {
		t.Error("Config still shows enabled")
	}
}

func TestEnableAgentWithGroup(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewManager(tmpDir)

	// Create library structure with groups
	libraryDir := filepath.Join(tmpDir, ".library")
	groupDir := filepath.Join(libraryDir, "agents", "business-product")
	os.MkdirAll(groupDir, 0755)
	os.WriteFile(filepath.Join(groupDir, "analyst.md"), []byte("# Analyst"), 0644)

	// Enable grouped agent
	enabled, _, err := manager.Enable("agents", []string{"business-product/analyst"})
	if err != nil {
		t.Fatalf("Enable() error = %v", err)
	}
	if len(enabled) != 1 || enabled[0] != "business-product/analyst.md" {
		t.Errorf("Enable() enabled = %v, want [business-product/analyst.md]", enabled)
	}

	// Verify symlink exists in correct location
	symlinkPath := filepath.Join(tmpDir, "agents", "business-product", "analyst.md")
	if _, err := os.Lstat(symlinkPath); os.IsNotExist(err) {
		t.Error("Symlink was not created in group directory")
	}

	// Verify it's a symlink pointing to the right place
	target, err := os.Readlink(symlinkPath)
	if err != nil {
		t.Fatalf("Readlink() error = %v", err)
	}
	expectedTarget := filepath.Join("..", "..", ".library", "agents", "business-product", "analyst.md")
	if target != expectedTarget {
		t.Errorf("Symlink target = %q, want %q", target, expectedTarget)
	}
}

func TestEnableWildcard(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewManager(tmpDir)

	// Create library structure
	libraryDir := filepath.Join(tmpDir, ".library")
	agentsDir := filepath.Join(libraryDir, "agents")
	os.MkdirAll(agentsDir, 0755)
	os.WriteFile(filepath.Join(agentsDir, "gsd-planner.md"), []byte("# Planner"), 0644)
	os.WriteFile(filepath.Join(agentsDir, "gsd-executor.md"), []byte("# Executor"), 0644)
	os.WriteFile(filepath.Join(agentsDir, "other-agent.md"), []byte("# Other"), 0644)

	// Enable with wildcard
	enabled, _, err := manager.Enable("agents", []string{"gsd-*"})
	if err != nil {
		t.Fatalf("Enable() error = %v", err)
	}
	if len(enabled) != 2 {
		t.Errorf("Enable() enabled %d items, want 2: %v", len(enabled), enabled)
	}
}
