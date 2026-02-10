// ABOUTME: Integration tests for local item management
// ABOUTME: Tests end-to-end enable/disable/list/view flows
package local

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/claudeup/claudeup/v4/internal/local"
)

func TestLocalIntegration(t *testing.T) {
	claudeDir := t.TempDir()
	claudeupHome := t.TempDir()

	// Create full library structure
	libraryDir := filepath.Join(claudeupHome, "local")
	agentsDir := filepath.Join(libraryDir, "agents")
	commandsDir := filepath.Join(libraryDir, "commands", "gsd")
	hooksDir := filepath.Join(libraryDir, "hooks")

	os.MkdirAll(agentsDir, 0755)
	os.MkdirAll(commandsDir, 0755)
	os.MkdirAll(hooksDir, 0755)

	// Create test items
	os.WriteFile(filepath.Join(agentsDir, "gsd-planner.md"), []byte("# Planner"), 0644)
	os.WriteFile(filepath.Join(agentsDir, "gsd-executor.md"), []byte("# Executor"), 0644)
	os.WriteFile(filepath.Join(commandsDir, "new-project.md"), []byte("# New Project"), 0644)
	os.WriteFile(filepath.Join(hooksDir, "gsd-check-update.js"), []byte("// JS"), 0644)

	manager := local.NewManager(claudeDir, claudeupHome)

	// Test: List items
	agents, err := manager.ListItems("agents")
	if err != nil {
		t.Fatalf("ListItems(agents) error = %v", err)
	}
	if len(agents) != 2 {
		t.Errorf("ListItems(agents) = %d items, want 2", len(agents))
	}

	// Test: Enable with wildcard
	enabled, notFound, err := manager.Enable("agents", []string{"gsd-*"})
	if err != nil {
		t.Fatalf("Enable(agents, gsd-*) error = %v", err)
	}
	if len(enabled) != 2 {
		t.Errorf("Enable() enabled %d items, want 2", len(enabled))
	}
	if len(notFound) != 0 {
		t.Errorf("Enable() notFound = %v, want []", notFound)
	}

	// Verify symlinks
	for _, item := range enabled {
		symlinkPath := filepath.Join(claudeDir, "agents", item)
		if _, err := os.Lstat(symlinkPath); os.IsNotExist(err) {
			t.Errorf("Symlink not created for %s", item)
		}
	}

	// Test: Disable
	disabled, _, err := manager.Disable("agents", []string{"gsd-planner"})
	if err != nil {
		t.Fatalf("Disable() error = %v", err)
	}
	if len(disabled) != 1 {
		t.Errorf("Disable() disabled %d items, want 1", len(disabled))
	}

	// Verify symlink removed
	if _, err := os.Lstat(filepath.Join(claudeDir, "agents", "gsd-planner.md")); !os.IsNotExist(err) {
		t.Error("Symlink was not removed")
	}

	// Test: View
	content, err := manager.View("agents", "gsd-executor")
	if err != nil {
		t.Fatalf("View() error = %v", err)
	}
	if content != "# Executor" {
		t.Errorf("View() content = %q, want '# Executor'", content)
	}

	// Test: Sync
	err = manager.Sync()
	if err != nil {
		t.Fatalf("Sync() error = %v", err)
	}
}

func TestLocalWithAgentGroups(t *testing.T) {
	claudeDir := t.TempDir()
	claudeupHome := t.TempDir()

	// Create library structure with agent groups
	libraryDir := filepath.Join(claudeupHome, "local")
	agentsDir := filepath.Join(libraryDir, "agents")
	groupDir := filepath.Join(agentsDir, "business-product")

	os.MkdirAll(groupDir, 0755)

	// Create flat and grouped agents
	os.WriteFile(filepath.Join(agentsDir, "gsd-planner.md"), []byte("# Planner"), 0644)
	os.WriteFile(filepath.Join(groupDir, "analyst.md"), []byte("# Analyst"), 0644)
	os.WriteFile(filepath.Join(groupDir, "strategist.md"), []byte("# Strategist"), 0644)

	manager := local.NewManager(claudeDir, claudeupHome)

	// Test: List returns both flat and grouped
	agents, err := manager.ListItems("agents")
	if err != nil {
		t.Fatalf("ListItems(agents) error = %v", err)
	}
	if len(agents) != 3 {
		t.Errorf("ListItems(agents) = %d items, want 3: %v", len(agents), agents)
	}

	// Test: Enable grouped agent with directory wildcard
	enabled, _, err := manager.Enable("agents", []string{"business-product/*"})
	if err != nil {
		t.Fatalf("Enable() error = %v", err)
	}
	if len(enabled) != 2 {
		t.Errorf("Enable() enabled %d items, want 2: %v", len(enabled), enabled)
	}

	// Verify symlinks in group directory
	symlinkPath := filepath.Join(claudeDir, "agents", "business-product", "analyst.md")
	if _, err := os.Lstat(symlinkPath); os.IsNotExist(err) {
		t.Error("Symlink was not created in group directory")
	}

	// Verify it's a symlink (target path correctness deferred to Task 5 absolute symlinks)
	_, err = os.Readlink(symlinkPath)
	if err != nil {
		t.Fatalf("Readlink() error = %v", err)
	}
}

func TestLocalCommandsWithDirectoryStructure(t *testing.T) {
	claudeDir := t.TempDir()
	claudeupHome := t.TempDir()

	// Create library structure for commands with directories
	libraryDir := filepath.Join(claudeupHome, "local")
	commandsDir := filepath.Join(libraryDir, "commands")
	gsdDir := filepath.Join(commandsDir, "gsd")

	os.MkdirAll(gsdDir, 0755)

	// Create command files
	os.WriteFile(filepath.Join(gsdDir, "new-project.md"), []byte("# New Project"), 0644)
	os.WriteFile(filepath.Join(gsdDir, "execute-phase.md"), []byte("# Execute Phase"), 0644)
	os.WriteFile(filepath.Join(commandsDir, "other-command.md"), []byte("# Other"), 0644)

	manager := local.NewManager(claudeDir, claudeupHome)

	// Test: List commands
	commands, err := manager.ListItems("commands")
	if err != nil {
		t.Fatalf("ListItems(commands) error = %v", err)
	}
	// Commands include both flat files and files in directories
	if len(commands) < 1 {
		t.Errorf("ListItems(commands) = %d items, expected at least 1", len(commands))
	}

	// Test: Enable specific command
	enabled, _, err := manager.Enable("commands", []string{"other-command"})
	if err != nil {
		t.Fatalf("Enable() error = %v", err)
	}
	if len(enabled) != 1 {
		t.Errorf("Enable() enabled %d items, want 1", len(enabled))
	}

	// Verify symlink
	symlinkPath := filepath.Join(claudeDir, "commands", "other-command.md")
	if _, err := os.Lstat(symlinkPath); os.IsNotExist(err) {
		t.Error("Symlink was not created")
	}
}

func TestLocalConfigPersistence(t *testing.T) {
	claudeDir := t.TempDir()
	claudeupHome := t.TempDir()

	// Create library structure
	libraryDir := filepath.Join(claudeupHome, "local")
	hooksDir := filepath.Join(libraryDir, "hooks")
	os.MkdirAll(hooksDir, 0755)
	os.WriteFile(filepath.Join(hooksDir, "format-on-save.sh"), []byte("#!/bin/bash"), 0644)

	manager := local.NewManager(claudeDir, claudeupHome)

	// Enable hook
	_, _, err := manager.Enable("hooks", []string{"format-on-save"})
	if err != nil {
		t.Fatalf("Enable() error = %v", err)
	}

	// Create new manager instance (simulating restart)
	manager2 := local.NewManager(claudeDir, claudeupHome)

	// Load config should show enabled item
	config, err := manager2.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if !config["hooks"]["format-on-save.sh"] {
		t.Error("Config did not persist enabled state")
	}
}
