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

func TestImport(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewManager(tmpDir)

	// Create .library directory structure (empty)
	libraryDir := filepath.Join(tmpDir, ".library")
	os.MkdirAll(filepath.Join(libraryDir, "agents"), 0755)
	os.MkdirAll(filepath.Join(libraryDir, "hooks"), 0755)

	// Create files directly in active directories (simulating GSD install)
	activeAgentsDir := filepath.Join(tmpDir, "agents")
	activeHooksDir := filepath.Join(tmpDir, "hooks")
	os.MkdirAll(activeAgentsDir, 0755)
	os.MkdirAll(activeHooksDir, 0755)

	os.WriteFile(filepath.Join(activeAgentsDir, "gsd-planner.md"), []byte("# Planner"), 0644)
	os.WriteFile(filepath.Join(activeAgentsDir, "gsd-executor.md"), []byte("# Executor"), 0644)
	os.WriteFile(filepath.Join(activeAgentsDir, "other-agent.md"), []byte("# Other"), 0644)
	os.WriteFile(filepath.Join(activeHooksDir, "gsd-check-update.js"), []byte("// JS"), 0644)

	// Import GSD agents with wildcard
	imported, notFound, err := manager.Import("agents", []string{"gsd-*"})
	if err != nil {
		t.Fatalf("Import() error = %v", err)
	}
	if len(imported) != 2 {
		t.Errorf("Import() imported %d items, want 2: %v", len(imported), imported)
	}
	if len(notFound) != 0 {
		t.Errorf("Import() notFound = %v, want []", notFound)
	}

	// Verify files moved to .library
	if _, err := os.Stat(filepath.Join(libraryDir, "agents", "gsd-planner.md")); os.IsNotExist(err) {
		t.Error("gsd-planner.md was not moved to .library")
	}
	if _, err := os.Stat(filepath.Join(libraryDir, "agents", "gsd-executor.md")); os.IsNotExist(err) {
		t.Error("gsd-executor.md was not moved to .library")
	}

	// Verify other-agent.md was NOT moved (didn't match pattern)
	if _, err := os.Stat(filepath.Join(activeAgentsDir, "other-agent.md")); os.IsNotExist(err) {
		t.Error("other-agent.md should not have been moved")
	}

	// Verify symlinks were created in active directory
	symlinkPath := filepath.Join(activeAgentsDir, "gsd-planner.md")
	info, err := os.Lstat(symlinkPath)
	if err != nil {
		t.Fatalf("Lstat() error = %v", err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Error("gsd-planner.md should be a symlink after import")
	}

	// Verify symlink target
	target, _ := os.Readlink(symlinkPath)
	expectedTarget := filepath.Join("..", ".library", "agents", "gsd-planner.md")
	if target != expectedTarget {
		t.Errorf("Symlink target = %q, want %q", target, expectedTarget)
	}

	// Verify enabled.json was updated
	config, _ := manager.LoadConfig()
	if !config["agents"]["gsd-planner.md"] {
		t.Error("gsd-planner.md should be enabled in config")
	}
}

func TestImportDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewManager(tmpDir)

	// Create .library directory structure (empty)
	libraryDir := filepath.Join(tmpDir, ".library")
	os.MkdirAll(filepath.Join(libraryDir, "commands"), 0755)

	// Create commands/gsd directory directly (simulating GSD install)
	activeCommandsDir := filepath.Join(tmpDir, "commands")
	gsdCommandsDir := filepath.Join(activeCommandsDir, "gsd")
	os.MkdirAll(gsdCommandsDir, 0755)

	os.WriteFile(filepath.Join(gsdCommandsDir, "new-project.md"), []byte("# New Project"), 0644)
	os.WriteFile(filepath.Join(gsdCommandsDir, "execute-phase.md"), []byte("# Execute"), 0644)

	// Import the gsd directory
	imported, _, err := manager.Import("commands", []string{"gsd"})
	if err != nil {
		t.Fatalf("Import() error = %v", err)
	}
	if len(imported) != 1 || imported[0] != "gsd" {
		t.Errorf("Import() imported = %v, want [gsd]", imported)
	}

	// Verify directory moved to .library
	if _, err := os.Stat(filepath.Join(libraryDir, "commands", "gsd", "new-project.md")); os.IsNotExist(err) {
		t.Error("gsd directory was not moved to .library")
	}

	// Verify symlink was created
	symlinkPath := filepath.Join(activeCommandsDir, "gsd")
	info, err := os.Lstat(symlinkPath)
	if err != nil {
		t.Fatalf("Lstat() error = %v", err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Error("gsd should be a symlink after import")
	}
}

func TestImportSkipsSymlinks(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewManager(tmpDir)

	// Create .library with an item
	libraryDir := filepath.Join(tmpDir, ".library")
	os.MkdirAll(filepath.Join(libraryDir, "agents"), 0755)
	os.WriteFile(filepath.Join(libraryDir, "agents", "existing.md"), []byte("# Existing"), 0644)

	// Create active directory with a symlink (already managed)
	activeAgentsDir := filepath.Join(tmpDir, "agents")
	os.MkdirAll(activeAgentsDir, 0755)
	os.Symlink(filepath.Join("..", ".library", "agents", "existing.md"), filepath.Join(activeAgentsDir, "existing.md"))

	// Also create a real file
	os.WriteFile(filepath.Join(activeAgentsDir, "new-agent.md"), []byte("# New"), 0644)

	// Import all
	imported, _, err := manager.Import("agents", []string{"*"})
	if err != nil {
		t.Fatalf("Import() error = %v", err)
	}

	// Should only import the real file, not the symlink
	if len(imported) != 1 || imported[0] != "new-agent.md" {
		t.Errorf("Import() imported = %v, want [new-agent.md]", imported)
	}
}
