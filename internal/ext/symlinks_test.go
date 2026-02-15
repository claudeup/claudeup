// ABOUTME: Tests for symlink-based enable/disable operations
// ABOUTME: Verifies symlink creation, removal, and sync behavior
package ext

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnableDisable(t *testing.T) {
	claudeDir := t.TempDir()
	claudeupHome := t.TempDir()
	manager := NewManager(claudeDir, claudeupHome)

	// Create local directory structure
	localDir := filepath.Join(claudeupHome, "ext")
	hooksDir := filepath.Join(localDir, "hooks")
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
	targetDir := filepath.Join(claudeDir, "hooks")
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
	claudeDir := t.TempDir()
	claudeupHome := t.TempDir()
	manager := NewManager(claudeDir, claudeupHome)

	// Create local directory structure with groups
	localDir := filepath.Join(claudeupHome, "ext")
	groupDir := filepath.Join(localDir, "agents", "business-product")
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
	symlinkPath := filepath.Join(claudeDir, "agents", "business-product", "analyst.md")
	if _, err := os.Lstat(symlinkPath); os.IsNotExist(err) {
		t.Error("Symlink was not created in group directory")
	}

	// Verify symlink target is an absolute path to extension storage
	target, err := os.Readlink(symlinkPath)
	if err != nil {
		t.Fatalf("Readlink() error = %v", err)
	}
	expectedTarget := filepath.Join(claudeupHome, "ext", "agents", "business-product", "analyst.md")
	if target != expectedTarget {
		t.Errorf("Symlink target = %q, want %q", target, expectedTarget)
	}
}

func TestEnableWildcard(t *testing.T) {
	claudeDir := t.TempDir()
	claudeupHome := t.TempDir()
	manager := NewManager(claudeDir, claudeupHome)

	// Create local directory structure
	localDir := filepath.Join(claudeupHome, "ext")
	agentsDir := filepath.Join(localDir, "agents")
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

func TestEnableNestedCommand(t *testing.T) {
	claudeDir := t.TempDir()
	claudeupHome := t.TempDir()
	manager := NewManager(claudeDir, claudeupHome)

	// Create local directory structure for commands with subdirectory
	localDir := filepath.Join(claudeupHome, "ext")
	gsdCommandsDir := filepath.Join(localDir, "commands", "gsd")
	os.MkdirAll(gsdCommandsDir, 0755)
	os.WriteFile(filepath.Join(gsdCommandsDir, "new-project.md"), []byte("# New Project"), 0644)
	os.WriteFile(filepath.Join(gsdCommandsDir, "execute-phase.md"), []byte("# Execute Phase"), 0644)

	// Enable nested command using gsd/* wildcard
	enabled, _, err := manager.Enable("commands", []string{"gsd/*"})
	if err != nil {
		t.Fatalf("Enable() error = %v", err)
	}
	if len(enabled) != 2 {
		t.Errorf("Enable() enabled %d items, want 2: %v", len(enabled), enabled)
	}

	// Verify symlinks exist in nested structure (commands/gsd/new-project.md)
	symlinkPath := filepath.Join(claudeDir, "commands", "gsd", "new-project.md")
	info, err := os.Lstat(symlinkPath)
	if err != nil {
		t.Fatalf("Lstat() error = %v (symlink not created)", err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Error("Expected symlink, got regular file")
	}

	// Verify symlink target is an absolute path to extension storage
	target, err := os.Readlink(symlinkPath)
	if err != nil {
		t.Fatalf("Readlink() error = %v", err)
	}
	expectedTarget := filepath.Join(claudeupHome, "ext", "commands", "gsd", "new-project.md")
	if target != expectedTarget {
		t.Errorf("Symlink target = %q, want %q", target, expectedTarget)
	}
}

func TestImport(t *testing.T) {
	claudeDir := t.TempDir()
	claudeupHome := t.TempDir()
	manager := NewManager(claudeDir, claudeupHome)

	// Create local directory structure (empty)
	localDir := filepath.Join(claudeupHome, "ext")
	os.MkdirAll(filepath.Join(localDir, "agents"), 0755)
	os.MkdirAll(filepath.Join(localDir, "hooks"), 0755)

	// Create files directly in active directories (simulating GSD install)
	activeAgentsDir := filepath.Join(claudeDir, "agents")
	activeHooksDir := filepath.Join(claudeDir, "hooks")
	os.MkdirAll(activeAgentsDir, 0755)
	os.MkdirAll(activeHooksDir, 0755)

	os.WriteFile(filepath.Join(activeAgentsDir, "gsd-planner.md"), []byte("# Planner"), 0644)
	os.WriteFile(filepath.Join(activeAgentsDir, "gsd-executor.md"), []byte("# Executor"), 0644)
	os.WriteFile(filepath.Join(activeAgentsDir, "other-agent.md"), []byte("# Other"), 0644)
	os.WriteFile(filepath.Join(activeHooksDir, "gsd-check-update.js"), []byte("// JS"), 0644)

	// Import GSD agents with wildcard
	imported, _, notFound, err := manager.Import("agents", []string{"gsd-*"})
	if err != nil {
		t.Fatalf("Import() error = %v", err)
	}
	if len(imported) != 2 {
		t.Errorf("Import() imported %d items, want 2: %v", len(imported), imported)
	}
	if len(notFound) != 0 {
		t.Errorf("Import() notFound = %v, want []", notFound)
	}

	// Verify files moved to extension storage
	if _, err := os.Stat(filepath.Join(localDir, "agents", "gsd-planner.md")); os.IsNotExist(err) {
		t.Error("gsd-planner.md was not moved to extension storage")
	}
	if _, err := os.Stat(filepath.Join(localDir, "agents", "gsd-executor.md")); os.IsNotExist(err) {
		t.Error("gsd-executor.md was not moved to extension storage")
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

	// Verify enabled.json was updated
	config, _ := manager.LoadConfig()
	if !config["agents"]["gsd-planner.md"] {
		t.Error("gsd-planner.md should be enabled in config")
	}
}

func TestImportDirectory(t *testing.T) {
	claudeDir := t.TempDir()
	claudeupHome := t.TempDir()
	manager := NewManager(claudeDir, claudeupHome)

	// Create local directory structure (empty)
	localDir := filepath.Join(claudeupHome, "ext")
	os.MkdirAll(filepath.Join(localDir, "commands"), 0755)

	// Create commands/gsd directory directly (simulating GSD install)
	activeCommandsDir := filepath.Join(claudeDir, "commands")
	gsdCommandsDir := filepath.Join(activeCommandsDir, "gsd")
	os.MkdirAll(gsdCommandsDir, 0755)

	os.WriteFile(filepath.Join(gsdCommandsDir, "new-project.md"), []byte("# New Project"), 0644)
	os.WriteFile(filepath.Join(gsdCommandsDir, "execute-phase.md"), []byte("# Execute"), 0644)

	// Import the gsd directory
	imported, _, _, err := manager.Import("commands", []string{"gsd"})
	if err != nil {
		t.Fatalf("Import() error = %v", err)
	}
	if len(imported) != 1 || imported[0] != "gsd" {
		t.Errorf("Import() imported = %v, want [gsd]", imported)
	}

	// Verify directory moved to extension storage
	if _, err := os.Stat(filepath.Join(localDir, "commands", "gsd", "new-project.md")); os.IsNotExist(err) {
		t.Error("gsd directory was not moved to extension storage")
	}

	// Verify gsd directory was created (as regular dir with symlinks inside)
	// The enable logic expands directories to their individual files for proper list display
	gsdDir := filepath.Join(activeCommandsDir, "gsd")
	dirInfo, err := os.Stat(gsdDir)
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}
	if !dirInfo.IsDir() {
		t.Error("gsd should be a directory")
	}

	// Verify individual files inside are symlinks
	symlinkPath := filepath.Join(gsdDir, "new-project.md")
	info, err := os.Lstat(symlinkPath)
	if err != nil {
		t.Fatalf("Lstat() error = %v", err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Error("gsd/new-project.md should be a symlink after import")
	}

	// Verify config has individual items enabled (for correct list display)
	config, _ := manager.LoadConfig()
	if !config["commands"]["gsd/new-project.md"] {
		t.Error("Config should have 'gsd/new-project.md' enabled")
	}
	if !config["commands"]["gsd/execute-phase.md"] {
		t.Error("Config should have 'gsd/execute-phase.md' enabled")
	}
}

func TestImportSkipsSymlinks(t *testing.T) {
	claudeDir := t.TempDir()
	claudeupHome := t.TempDir()
	manager := NewManager(claudeDir, claudeupHome)

	// Create extension storage with an item
	localDir := filepath.Join(claudeupHome, "ext")
	os.MkdirAll(filepath.Join(localDir, "agents"), 0755)
	os.WriteFile(filepath.Join(localDir, "agents", "existing.md"), []byte("# Existing"), 0644)

	// Create active directory with a symlink (already managed)
	activeAgentsDir := filepath.Join(claudeDir, "agents")
	os.MkdirAll(activeAgentsDir, 0755)
	os.Symlink(filepath.Join(localDir, "agents", "existing.md"), filepath.Join(activeAgentsDir, "existing.md"))

	// Also create a real file
	os.WriteFile(filepath.Join(activeAgentsDir, "new-agent.md"), []byte("# New"), 0644)

	// Import all
	imported, _, _, err := manager.Import("agents", []string{"*"})
	if err != nil {
		t.Fatalf("Import() error = %v", err)
	}

	// Should only import the real file, not the symlink
	if len(imported) != 1 || imported[0] != "new-agent.md" {
		t.Errorf("Import() imported = %v, want [new-agent.md]", imported)
	}
}

func TestImportAll(t *testing.T) {
	claudeDir := t.TempDir()
	claudeupHome := t.TempDir()
	manager := NewManager(claudeDir, claudeupHome)

	// Create files directly in active directories (simulating GSD install)
	activeAgentsDir := filepath.Join(claudeDir, "agents")
	activeCommandsDir := filepath.Join(claudeDir, "commands")
	activeHooksDir := filepath.Join(claudeDir, "hooks")
	os.MkdirAll(activeAgentsDir, 0755)
	os.MkdirAll(activeCommandsDir, 0755)
	os.MkdirAll(activeHooksDir, 0755)

	os.WriteFile(filepath.Join(activeAgentsDir, "gsd-planner.md"), []byte("# Planner"), 0644)
	os.WriteFile(filepath.Join(activeAgentsDir, "gsd-executor.md"), []byte("# Executor"), 0644)
	os.WriteFile(filepath.Join(activeAgentsDir, "other-agent.md"), []byte("# Other"), 0644)
	os.MkdirAll(filepath.Join(activeCommandsDir, "gsd"), 0755)
	os.WriteFile(filepath.Join(activeCommandsDir, "gsd", "start.md"), []byte("# Start"), 0644)
	os.WriteFile(filepath.Join(activeHooksDir, "gsd-check-update.js"), []byte("// JS"), 0644)

	// Import all with pattern
	results, _, err := manager.ImportAll([]string{"gsd-*", "gsd"})
	if err != nil {
		t.Fatalf("ImportAll() error = %v", err)
	}

	// Should import gsd-* agents, gsd commands dir, gsd-* hooks
	if len(results["agents"]) != 2 {
		t.Errorf("ImportAll() agents = %v, want 2 items", results["agents"])
	}
	if len(results["commands"]) != 1 || results["commands"][0] != "gsd" {
		t.Errorf("ImportAll() commands = %v, want [gsd]", results["commands"])
	}
	if len(results["hooks"]) != 1 {
		t.Errorf("ImportAll() hooks = %v, want 1 item", results["hooks"])
	}

	// Verify files moved to extension storage
	localDir := filepath.Join(claudeupHome, "ext")
	if _, err := os.Stat(filepath.Join(localDir, "agents", "gsd-planner.md")); os.IsNotExist(err) {
		t.Error("gsd-planner.md was not moved to extension storage")
	}

	// Verify other-agent.md was NOT moved (didn't match pattern)
	if _, err := os.Stat(filepath.Join(activeAgentsDir, "other-agent.md")); os.IsNotExist(err) {
		t.Error("other-agent.md should not have been moved")
	}
}

// TestEnableDirectoryByName tests that enabling a directory by name (without wildcard)
// expands to enable all items inside it. This was a bug where:
// - `enable commands vsphere-architect` would set config["vsphere-architect"]=true
// - `list commands` would check config["vsphere-architect/capacity-plan.md"] and find nothing
func TestEnableDirectoryByName(t *testing.T) {
	claudeDir := t.TempDir()
	claudeupHome := t.TempDir()
	manager := NewManager(claudeDir, claudeupHome)

	// Create local directory structure for commands with a subdirectory
	localDir := filepath.Join(claudeupHome, "ext")
	vsphereDir := filepath.Join(localDir, "commands", "vsphere-architect")
	os.MkdirAll(vsphereDir, 0755)
	os.WriteFile(filepath.Join(vsphereDir, "capacity-plan.md"), []byte("# Capacity Plan"), 0644)
	os.WriteFile(filepath.Join(vsphereDir, "ha-design.md"), []byte("# HA Design"), 0644)
	os.WriteFile(filepath.Join(vsphereDir, "storage-design.md"), []byte("# Storage Design"), 0644)

	// Enable using just the directory name (no wildcard)
	enabled, notFound, err := manager.Enable("commands", []string{"vsphere-architect"})
	if err != nil {
		t.Fatalf("Enable() error = %v", err)
	}
	if len(notFound) != 0 {
		t.Errorf("Enable() notFound = %v, want []", notFound)
	}
	if len(enabled) != 3 {
		t.Errorf("Enable() enabled %d items, want 3: %v", len(enabled), enabled)
	}

	// Verify config has individual items (not the directory)
	config, _ := manager.LoadConfig()
	if config["commands"]["vsphere-architect"] {
		t.Error("Config should NOT have 'vsphere-architect' as a single item")
	}
	if !config["commands"]["vsphere-architect/capacity-plan.md"] {
		t.Error("Config should have 'vsphere-architect/capacity-plan.md' enabled")
	}
	if !config["commands"]["vsphere-architect/ha-design.md"] {
		t.Error("Config should have 'vsphere-architect/ha-design.md' enabled")
	}
	if !config["commands"]["vsphere-architect/storage-design.md"] {
		t.Error("Config should have 'vsphere-architect/storage-design.md' enabled")
	}

	// Verify symlinks exist for each individual item
	symlinkPath := filepath.Join(claudeDir, "commands", "vsphere-architect", "capacity-plan.md")
	info, err := os.Lstat(symlinkPath)
	if err != nil {
		t.Fatalf("Lstat() error = %v (symlink not created)", err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Error("Expected symlink, got regular file")
	}
}

func TestEnableRejectsPathTraversal(t *testing.T) {
	claudeDir := t.TempDir()
	claudeupHome := t.TempDir()
	manager := NewManager(claudeDir, claudeupHome)

	// Create local directory structure
	localDir := filepath.Join(claudeupHome, "ext")
	commandsDir := filepath.Join(localDir, "commands")
	os.MkdirAll(commandsDir, 0755)
	os.WriteFile(filepath.Join(commandsDir, "legit.md"), []byte("# Legit"), 0644)

	// Manually write a malicious config with path traversal
	config := Config{
		"commands": {
			"../../../etc/passwd":     true,
			"legit/../../../tmp/evil": true,
			"gsd/../../outside":       true,
			"legit.md":                true, // This one is fine
		},
	}
	manager.SaveConfig(config)

	// Sync should reject path traversal attempts
	err := manager.Sync()
	if err == nil {
		t.Fatal("Sync() should have rejected path traversal, got nil error")
	}

	// Error should mention path traversal
	if !strings.Contains(err.Error(), "path traversal") {
		t.Errorf("Error should mention path traversal, got: %v", err)
	}

	// Verify no symlinks were created outside the target directory
	// (the legit.md should NOT have been created either - fail fast)
	legitPath := filepath.Join(claudeDir, "commands", "legit.md")
	if _, err := os.Lstat(legitPath); err == nil {
		t.Error("Sync should fail fast - no symlinks created when traversal detected")
	}
}

func TestImportAllNoPattern(t *testing.T) {
	claudeDir := t.TempDir()
	claudeupHome := t.TempDir()
	manager := NewManager(claudeDir, claudeupHome)

	// Create files directly in active directories
	activeAgentsDir := filepath.Join(claudeDir, "agents")
	activeHooksDir := filepath.Join(claudeDir, "hooks")
	os.MkdirAll(activeAgentsDir, 0755)
	os.MkdirAll(activeHooksDir, 0755)

	os.WriteFile(filepath.Join(activeAgentsDir, "agent1.md"), []byte("# Agent1"), 0644)
	os.WriteFile(filepath.Join(activeAgentsDir, "agent2.md"), []byte("# Agent2"), 0644)
	os.WriteFile(filepath.Join(activeHooksDir, "hook1.js"), []byte("// JS"), 0644)

	// Import all without pattern (should import everything)
	results, _, err := manager.ImportAll(nil)
	if err != nil {
		t.Fatalf("ImportAll() error = %v", err)
	}

	// Should import all items
	if len(results["agents"]) != 2 {
		t.Errorf("ImportAll() agents = %v, want 2 items", results["agents"])
	}
	if len(results["hooks"]) != 1 {
		t.Errorf("ImportAll() hooks = %v, want 1 item", results["hooks"])
	}
}

// TestImportReconciliation verifies that when importing items that already exist
// in extension storage, the active copies are removed and symlinks are created.
func TestImportReconciliation(t *testing.T) {
	claudeDir := t.TempDir()
	claudeupHome := t.TempDir()
	manager := NewManager(claudeDir, claudeupHome)

	// Create extension storage with an existing item
	localDir := filepath.Join(claudeupHome, "ext")
	os.MkdirAll(filepath.Join(localDir, "agents"), 0755)
	os.WriteFile(filepath.Join(localDir, "agents", "existing-agent.md"), []byte("# Stored Version"), 0644)

	// Create active directory with a duplicate (local version)
	activeAgentsDir := filepath.Join(claudeDir, "agents")
	os.MkdirAll(activeAgentsDir, 0755)
	os.WriteFile(filepath.Join(activeAgentsDir, "existing-agent.md"), []byte("# Local Version"), 0644)

	// Import the duplicate item
	imported, skipped, _, err := manager.Import("agents", []string{"existing-agent.md"})
	if err != nil {
		t.Fatalf("Import() error = %v", err)
	}

	// Should be reported as skipped (reconciled), not imported
	if len(imported) != 0 {
		t.Errorf("Import() imported = %v, want []", imported)
	}
	if len(skipped) != 1 || skipped[0] != "existing-agent.md" {
		t.Errorf("Import() skipped = %v, want [existing-agent.md]", skipped)
	}

	// Local file should be removed
	localPath := filepath.Join(activeAgentsDir, "existing-agent.md")
	info, err := os.Lstat(localPath)
	if err != nil {
		t.Fatalf("Lstat() error = %v (symlink should exist)", err)
	}

	// Should now be a symlink (not the original file)
	if info.Mode()&os.ModeSymlink == 0 {
		t.Error("Local path should be a symlink after reconciliation")
	}

	// Stored version should be preserved (not overwritten)
	content, _ := os.ReadFile(filepath.Join(localDir, "agents", "existing-agent.md"))
	if string(content) != "# Stored Version" {
		t.Error("Stored version should be preserved during reconciliation")
	}
}

// TestEnableMixedItems tests enabling both a directory and individual files simultaneously
func TestEnableMixedItems(t *testing.T) {
	claudeDir := t.TempDir()
	claudeupHome := t.TempDir()
	manager := NewManager(claudeDir, claudeupHome)

	// Create local directory structure
	localDir := filepath.Join(claudeupHome, "ext")
	commandsDir := filepath.Join(localDir, "commands")
	groupDir := filepath.Join(commandsDir, "group")
	os.MkdirAll(groupDir, 0755)
	os.WriteFile(filepath.Join(commandsDir, "standalone.md"), []byte("# Standalone"), 0644)
	os.WriteFile(filepath.Join(groupDir, "item1.md"), []byte("# Item 1"), 0644)
	os.WriteFile(filepath.Join(groupDir, "item2.md"), []byte("# Item 2"), 0644)

	// Enable both a directory and a standalone file
	enabled, notFound, err := manager.Enable("commands", []string{"group", "standalone.md"})
	if err != nil {
		t.Fatalf("Enable() error = %v", err)
	}
	if len(notFound) != 0 {
		t.Errorf("Enable() notFound = %v, want []", notFound)
	}
	// Should have 3 items: 2 from group + 1 standalone
	if len(enabled) != 3 {
		t.Errorf("Enable() enabled %d items, want 3: %v", len(enabled), enabled)
	}

	// Verify config
	config, _ := manager.LoadConfig()
	if !config["commands"]["group/item1.md"] {
		t.Error("Config should have 'group/item1.md' enabled")
	}
	if !config["commands"]["group/item2.md"] {
		t.Error("Config should have 'group/item2.md' enabled")
	}
	if !config["commands"]["standalone.md"] {
		t.Error("Config should have 'standalone.md' enabled")
	}
}

// TestEnableEmptyDirectory tests behavior when enabling an empty directory
func TestEnableEmptyDirectory(t *testing.T) {
	claudeDir := t.TempDir()
	claudeupHome := t.TempDir()
	manager := NewManager(claudeDir, claudeupHome)

	// Create extension storage with empty directory
	localDir := filepath.Join(claudeupHome, "ext")
	emptyDir := filepath.Join(localDir, "commands", "empty-dir")
	os.MkdirAll(emptyDir, 0755)

	// Enable empty directory - should report as not found (no items inside)
	enabled, notFound, err := manager.Enable("commands", []string{"empty-dir"})
	if err != nil {
		t.Fatalf("Enable() error = %v", err)
	}
	if len(enabled) != 0 {
		t.Errorf("Enable() enabled = %v, want [] for empty directory", enabled)
	}
	if len(notFound) != 1 || notFound[0] != "empty-dir" {
		t.Errorf("Enable() notFound = %v, want [empty-dir] for empty directory", notFound)
	}
}

// TestEnableNestedDirectories tests enabling items in nested directory structures
func TestEnableNestedDirectories(t *testing.T) {
	claudeDir := t.TempDir()
	claudeupHome := t.TempDir()
	manager := NewManager(claudeDir, claudeupHome)

	// Create nested directory structure (only one level is expanded)
	localDir := filepath.Join(claudeupHome, "ext")
	topDir := filepath.Join(localDir, "commands", "top")
	os.MkdirAll(topDir, 0755)
	os.WriteFile(filepath.Join(topDir, "item.md"), []byte("# Item"), 0644)

	// Enable top-level directory
	enabled, _, err := manager.Enable("commands", []string{"top"})
	if err != nil {
		t.Fatalf("Enable() error = %v", err)
	}

	// Should enable the item inside
	if len(enabled) != 1 || enabled[0] != "top/item.md" {
		t.Errorf("Enable() enabled = %v, want [top/item.md]", enabled)
	}

	// Verify symlink exists
	symlinkPath := filepath.Join(claudeDir, "commands", "top", "item.md")
	info, err := os.Lstat(symlinkPath)
	if err != nil {
		t.Fatalf("Lstat() error = %v", err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Error("Expected symlink")
	}
}

// TestImportSkipsGitkeep verifies that .gitkeep files are not imported or enabled
func TestImportSkipsGitkeep(t *testing.T) {
	claudeDir := t.TempDir()
	claudeupHome := t.TempDir()
	manager := NewManager(claudeDir, claudeupHome)

	// Create active directories with .gitkeep files and real items
	activeHooksDir := filepath.Join(claudeDir, "hooks")
	activeCommandsDir := filepath.Join(claudeDir, "commands")
	os.MkdirAll(activeHooksDir, 0755)
	os.MkdirAll(activeCommandsDir, 0755)

	// Add .gitkeep files (should be ignored)
	os.WriteFile(filepath.Join(activeHooksDir, ".gitkeep"), []byte(""), 0644)
	os.WriteFile(filepath.Join(activeCommandsDir, ".gitkeep"), []byte(""), 0644)

	// Add real items
	os.WriteFile(filepath.Join(activeHooksDir, "format.sh"), []byte("#!/bin/bash"), 0755)
	os.WriteFile(filepath.Join(activeCommandsDir, "build.md"), []byte("# Build"), 0644)

	// Import all
	imported, skipped, err := manager.ImportAll([]string{"*"})
	if err != nil {
		t.Fatalf("ImportAll() error = %v", err)
	}

	// Should import the real items
	if len(imported["hooks"]) != 1 || imported["hooks"][0] != "format.sh" {
		t.Errorf("ImportAll() hooks = %v, want [format.sh]", imported["hooks"])
	}
	if len(imported["commands"]) != 1 || imported["commands"][0] != "build.md" {
		t.Errorf("ImportAll() commands = %v, want [build.md]", imported["commands"])
	}

	// .gitkeep should not appear in skipped either
	for category, items := range skipped {
		for _, item := range items {
			if item == ".gitkeep" {
				t.Errorf("ImportAll() skipped[%s] should not contain .gitkeep", category)
			}
		}
	}

	// Verify config does not contain .gitkeep
	config, _ := manager.LoadConfig()
	for category, items := range config {
		for item := range items {
			if item == ".gitkeep" {
				t.Errorf("Config[%s] should not contain .gitkeep", category)
			}
		}
	}
}
