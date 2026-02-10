// ABOUTME: Tests for listing local storage items
// ABOUTME: Verifies directory scanning and item discovery
package local

import (
	"os"
	"path/filepath"
	"testing"
)

func TestListItems(t *testing.T) {
	claudeDir := t.TempDir()
	claudeupHome := t.TempDir()
	manager := NewManager(claudeDir, claudeupHome)

	// Create local directory structure
	localDir := filepath.Join(claudeupHome, "local")
	agentsDir := filepath.Join(localDir, "agents")
	os.MkdirAll(agentsDir, 0755)

	// Create some agent files
	os.WriteFile(filepath.Join(agentsDir, "planner.md"), []byte("# Planner"), 0644)
	os.WriteFile(filepath.Join(agentsDir, "executor.md"), []byte("# Executor"), 0644)
	os.WriteFile(filepath.Join(agentsDir, ".hidden.md"), []byte("# Hidden"), 0644) // Should be excluded

	items, err := manager.ListItems("agents")
	if err != nil {
		t.Fatalf("ListItems() error = %v", err)
	}

	if len(items) != 2 {
		t.Errorf("ListItems() returned %d items, want 2", len(items))
	}

	// Should be sorted
	if items[0] != "executor.md" || items[1] != "planner.md" {
		t.Errorf("ListItems() = %v, want [executor.md, planner.md]", items)
	}
}

func TestListItemsWithGroups(t *testing.T) {
	claudeDir := t.TempDir()
	claudeupHome := t.TempDir()
	manager := NewManager(claudeDir, claudeupHome)

	// Create local directory structure with groups (for agents)
	localDir := filepath.Join(claudeupHome, "local")
	agentsDir := filepath.Join(localDir, "agents")
	groupDir := filepath.Join(agentsDir, "business-product")
	os.MkdirAll(groupDir, 0755)

	// Flat agents
	os.WriteFile(filepath.Join(agentsDir, "gsd-planner.md"), []byte("# GSD Planner"), 0644)

	// Grouped agents
	os.WriteFile(filepath.Join(groupDir, "analyst.md"), []byte("# Analyst"), 0644)
	os.WriteFile(filepath.Join(groupDir, "strategist.md"), []byte("# Strategist"), 0644)

	items, err := manager.ListItems("agents")
	if err != nil {
		t.Fatalf("ListItems() error = %v", err)
	}

	// Should include both flat and grouped items
	expected := []string{
		"business-product/analyst.md",
		"business-product/strategist.md",
		"gsd-planner.md",
	}

	if len(items) != len(expected) {
		t.Errorf("ListItems() returned %d items, want %d: %v", len(items), len(expected), items)
	}

	for i, want := range expected {
		if i < len(items) && items[i] != want {
			t.Errorf("ListItems()[%d] = %q, want %q", i, items[i], want)
		}
	}
}

func TestListItemsEmptyCategory(t *testing.T) {
	claudeDir := t.TempDir()
	claudeupHome := t.TempDir()
	manager := NewManager(claudeDir, claudeupHome)

	items, err := manager.ListItems("agents")
	if err != nil {
		t.Fatalf("ListItems() error = %v", err)
	}

	if len(items) != 0 {
		t.Errorf("ListItems() returned %d items for empty category, want 0", len(items))
	}
}

func TestListItemsErrorPropagation(t *testing.T) {
	claudeDir := t.TempDir()
	claudeupHome := t.TempDir()
	manager := NewManager(claudeDir, claudeupHome)

	// Create local directory structure
	localDir := filepath.Join(claudeupHome, "local")
	agentsDir := filepath.Join(localDir, "agents")
	os.MkdirAll(agentsDir, 0755)
	os.WriteFile(filepath.Join(agentsDir, "test.md"), []byte("# Test"), 0644)

	// Make directory unreadable to trigger permission error
	os.Chmod(agentsDir, 0000)
	defer os.Chmod(agentsDir, 0755) // Restore for cleanup

	// ListItems should propagate the error, not return empty slice
	_, err := manager.ListItems("agents")
	if err == nil {
		t.Error("ListItems() should return error for unreadable directory")
	}
}

func TestListItemsNestedCommands(t *testing.T) {
	claudeDir := t.TempDir()
	claudeupHome := t.TempDir()
	manager := NewManager(claudeDir, claudeupHome)

	// Create local directory structure for commands with subdirectories
	localDir := filepath.Join(claudeupHome, "local")
	commandsDir := filepath.Join(localDir, "commands")
	gsdDir := filepath.Join(commandsDir, "gsd")
	os.MkdirAll(gsdDir, 0755)

	// Flat commands
	os.WriteFile(filepath.Join(commandsDir, "commit.md"), []byte("# Commit"), 0644)
	os.WriteFile(filepath.Join(commandsDir, "review.md"), []byte("# Review"), 0644)

	// Nested commands in gsd/ directory
	os.WriteFile(filepath.Join(gsdDir, "new-project.md"), []byte("# New Project"), 0644)
	os.WriteFile(filepath.Join(gsdDir, "execute-phase.md"), []byte("# Execute Phase"), 0644)

	items, err := manager.ListItems("commands")
	if err != nil {
		t.Fatalf("ListItems() error = %v", err)
	}

	// Should include both flat files AND files in subdirectories
	// Format for nested: subdir/filename.ext (like agents do)
	expected := []string{
		"commit.md",
		"gsd/execute-phase.md",
		"gsd/new-project.md",
		"review.md",
	}

	if len(items) != len(expected) {
		t.Errorf("ListItems(commands) returned %d items, want %d: got %v", len(items), len(expected), items)
	}

	for i, want := range expected {
		if i < len(items) && items[i] != want {
			t.Errorf("ListItems(commands)[%d] = %q, want %q", i, items[i], want)
		}
	}
}

func TestListItemsSkillsWithSKILLMD(t *testing.T) {
	claudeDir := t.TempDir()
	claudeupHome := t.TempDir()
	manager := NewManager(claudeDir, claudeupHome)

	// Create local directory structure for skills (directories containing SKILL.md)
	localDir := filepath.Join(claudeupHome, "local")
	skillsDir := filepath.Join(localDir, "skills")
	bashSkill := filepath.Join(skillsDir, "bash")
	webDesignSkill := filepath.Join(skillsDir, "web-design-guidelines")
	os.MkdirAll(bashSkill, 0755)
	os.MkdirAll(webDesignSkill, 0755)

	// Skills have SKILL.md inside directories
	os.WriteFile(filepath.Join(bashSkill, "SKILL.md"), []byte("# Bash Skill"), 0644)
	os.WriteFile(filepath.Join(webDesignSkill, "SKILL.md"), []byte("# Web Design"), 0644)

	// Also add a flat skill file (less common but valid)
	os.WriteFile(filepath.Join(skillsDir, "quick-tip.md"), []byte("# Quick Tip"), 0644)

	items, err := manager.ListItems("skills")
	if err != nil {
		t.Fatalf("ListItems() error = %v", err)
	}

	// Skills with SKILL.md should be listed by directory name only, not dir/SKILL.md
	expected := []string{
		"bash",
		"quick-tip.md",
		"web-design-guidelines",
	}

	if len(items) != len(expected) {
		t.Errorf("ListItems(skills) returned %d items, want %d: got %v", len(items), len(expected), items)
	}

	for i, want := range expected {
		if i < len(items) && items[i] != want {
			t.Errorf("ListItems(skills)[%d] = %q, want %q", i, items[i], want)
		}
	}
}
