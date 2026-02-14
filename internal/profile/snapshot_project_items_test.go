// ABOUTME: Tests for project-scoped local item snapshot capture
// ABOUTME: Validates scanning .claude/{agents,rules}/ for non-symlink files
package profile

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadProjectLocalItemsCapturesRegularFiles(t *testing.T) {
	projectDir := t.TempDir()

	// Create regular files in project .claude/ directories
	rulesDir := filepath.Join(projectDir, ".claude", "rules")
	mustMkdir(t, rulesDir)
	mustWriteFile(t, filepath.Join(rulesDir, "golang.md"), "# Go Rules")
	mustWriteFile(t, filepath.Join(rulesDir, "testing.md"), "# Testing Rules")

	agentsDir := filepath.Join(projectDir, ".claude", "agents")
	mustMkdir(t, agentsDir)
	mustWriteFile(t, filepath.Join(agentsDir, "reviewer.md"), "# Reviewer")

	items := readProjectLocalItems(projectDir)
	if items == nil {
		t.Fatal("expected non-nil LocalItemSettings")
	}

	if len(items.Rules) != 2 {
		t.Errorf("expected 2 rules, got %d: %v", len(items.Rules), items.Rules)
	}
	if len(items.Agents) != 1 || items.Agents[0] != "reviewer.md" {
		t.Errorf("expected agents [reviewer.md], got %v", items.Agents)
	}
}

func TestReadProjectLocalItemsSkipsSymlinks(t *testing.T) {
	projectDir := t.TempDir()
	claudeupHome := t.TempDir()

	// Create a source file in claudeup local storage
	localRulesDir := filepath.Join(claudeupHome, "local", "rules")
	mustMkdir(t, localRulesDir)
	mustWriteFile(t, filepath.Join(localRulesDir, "user-rule.md"), "# User Rule")

	// Create a symlink in project .claude/rules/ pointing to the local storage
	projectRulesDir := filepath.Join(projectDir, ".claude", "rules")
	mustMkdir(t, projectRulesDir)
	if err := os.Symlink(
		filepath.Join(localRulesDir, "user-rule.md"),
		filepath.Join(projectRulesDir, "user-rule.md"),
	); err != nil {
		t.Fatal(err)
	}

	// Also create a regular file (project-scoped)
	mustWriteFile(t, filepath.Join(projectRulesDir, "project-rule.md"), "# Project Rule")

	items := readProjectLocalItems(projectDir)
	if items == nil {
		t.Fatal("expected non-nil LocalItemSettings")
	}

	// Should capture regular file, skip symlink
	if len(items.Rules) != 1 || items.Rules[0] != "project-rule.md" {
		t.Errorf("expected rules [project-rule.md], got %v", items.Rules)
	}
}

func TestReadProjectLocalItemsReturnsNilWhenEmpty(t *testing.T) {
	projectDir := t.TempDir()

	// No .claude directory at all
	items := readProjectLocalItems(projectDir)
	if items != nil {
		t.Errorf("expected nil for empty project, got %v", items)
	}
}

func TestReadProjectLocalItemsReturnsNilWhenOnlySymlinks(t *testing.T) {
	projectDir := t.TempDir()

	// Create a rules directory with only a symlink
	projectRulesDir := filepath.Join(projectDir, ".claude", "rules")
	mustMkdir(t, projectRulesDir)

	// Create target file elsewhere
	targetDir := t.TempDir()
	mustWriteFile(t, filepath.Join(targetDir, "rule.md"), "# Rule")

	if err := os.Symlink(
		filepath.Join(targetDir, "rule.md"),
		filepath.Join(projectRulesDir, "rule.md"),
	); err != nil {
		t.Fatal(err)
	}

	items := readProjectLocalItems(projectDir)
	if items != nil {
		t.Errorf("expected nil when only symlinks present, got %v", items)
	}
}

func TestReadProjectLocalItemsSkipsHiddenAndCLAUDE(t *testing.T) {
	projectDir := t.TempDir()

	rulesDir := filepath.Join(projectDir, ".claude", "rules")
	mustMkdir(t, rulesDir)
	mustWriteFile(t, filepath.Join(rulesDir, ".hidden-rule.md"), "# Hidden")
	mustWriteFile(t, filepath.Join(rulesDir, "CLAUDE.md"), "# Claude")
	mustWriteFile(t, filepath.Join(rulesDir, "visible-rule.md"), "# Visible")

	items := readProjectLocalItems(projectDir)
	if items == nil {
		t.Fatal("expected non-nil LocalItemSettings")
	}

	if len(items.Rules) != 1 || items.Rules[0] != "visible-rule.md" {
		t.Errorf("expected rules [visible-rule.md], got %v", items.Rules)
	}
}

func TestSnapshotAllScopesCapuresProjectLocalItems(t *testing.T) {
	tempDir := t.TempDir()
	claudeDir := filepath.Join(tempDir, ".claude")
	claudeupHome := filepath.Join(tempDir, ".claudeup")
	projectDir := filepath.Join(tempDir, "project")
	claudeJSONPath := filepath.Join(claudeDir, ".claude.json")

	mustMkdir(t, claudeDir)
	mustMkdir(t, filepath.Join(claudeDir, "plugins"))
	mustMkdir(t, projectDir)

	// Initialize required files
	mustWriteJSON(t, filepath.Join(claudeDir, "settings.json"), map[string]any{"enabledPlugins": map[string]bool{}})
	mustWriteJSON(t, filepath.Join(claudeDir, "plugins", "known_marketplaces.json"), map[string]any{})
	mustWriteJSON(t, claudeJSONPath, map[string]any{"mcpServers": map[string]any{}})
	mustWriteFile(t, filepath.Join(claudeupHome, "enabled.json"), "{}")

	// Create project-scoped rule files (regular files in project .claude/)
	projectRulesDir := filepath.Join(projectDir, ".claude", "rules")
	mustMkdir(t, projectRulesDir)
	mustWriteFile(t, filepath.Join(projectRulesDir, "golang.md"), "# Golang Rules")

	// Create project-scoped agent files
	projectAgentsDir := filepath.Join(projectDir, ".claude", "agents")
	mustMkdir(t, projectAgentsDir)
	mustWriteFile(t, filepath.Join(projectAgentsDir, "reviewer.md"), "# Reviewer")

	profile, err := SnapshotAllScopes("test", claudeDir, claudeJSONPath, projectDir, claudeupHome)
	if err != nil {
		t.Fatalf("SnapshotAllScopes failed: %v", err)
	}

	// Project-scoped local items should be in PerScope.Project.LocalItems
	if profile.PerScope == nil || profile.PerScope.Project == nil {
		t.Fatal("expected PerScope.Project to be set")
	}
	if profile.PerScope.Project.LocalItems == nil {
		t.Fatal("expected PerScope.Project.LocalItems to be set")
	}

	projectItems := profile.PerScope.Project.LocalItems
	if len(projectItems.Rules) != 1 || projectItems.Rules[0] != "golang.md" {
		t.Errorf("expected project rules [golang.md], got %v", projectItems.Rules)
	}
	if len(projectItems.Agents) != 1 || projectItems.Agents[0] != "reviewer.md" {
		t.Errorf("expected project agents [reviewer.md], got %v", projectItems.Agents)
	}
}

func TestSnapshotAllScopesPutsUserItemsInPerScopeUser(t *testing.T) {
	tempDir := t.TempDir()
	claudeDir := filepath.Join(tempDir, ".claude")
	claudeupHome := filepath.Join(tempDir, ".claudeup")
	projectDir := filepath.Join(tempDir, "project")
	claudeJSONPath := filepath.Join(claudeDir, ".claude.json")

	mustMkdir(t, claudeDir)
	mustMkdir(t, filepath.Join(claudeDir, "plugins"))
	mustMkdir(t, projectDir)

	// Initialize required files
	mustWriteJSON(t, filepath.Join(claudeDir, "settings.json"), map[string]any{"enabledPlugins": map[string]bool{}})
	mustWriteJSON(t, filepath.Join(claudeDir, "plugins", "known_marketplaces.json"), map[string]any{})
	mustWriteJSON(t, claudeJSONPath, map[string]any{"mcpServers": map[string]any{}})

	// Create local items in claudeup storage and enable them
	localRulesDir := filepath.Join(claudeupHome, "local", "rules")
	mustMkdir(t, localRulesDir)
	mustWriteFile(t, filepath.Join(localRulesDir, "coding.md"), "# Coding")

	// Create symlink in claudeDir (simulating enabled state)
	claudeRulesDir := filepath.Join(claudeDir, "rules")
	mustMkdir(t, claudeRulesDir)
	if err := os.Symlink(
		filepath.Join(localRulesDir, "coding.md"),
		filepath.Join(claudeRulesDir, "coding.md"),
	); err != nil {
		t.Fatal(err)
	}

	// Create enabled.json tracking the enabled item
	mustWriteJSON(t, filepath.Join(claudeupHome, "enabled.json"), map[string]any{
		"rules": map[string]bool{"coding.md": true},
	})

	profile, err := SnapshotAllScopes("test", claudeDir, claudeJSONPath, projectDir, claudeupHome)
	if err != nil {
		t.Fatalf("SnapshotAllScopes failed: %v", err)
	}

	// User-scoped local items should be in PerScope.User.LocalItems
	if profile.PerScope == nil || profile.PerScope.User == nil {
		t.Fatal("expected PerScope.User to be set")
	}
	if profile.PerScope.User.LocalItems == nil {
		t.Fatal("expected PerScope.User.LocalItems to be set")
	}
	if len(profile.PerScope.User.LocalItems.Rules) != 1 || profile.PerScope.User.LocalItems.Rules[0] != "coding.md" {
		t.Errorf("expected user rules [coding.md], got %v", profile.PerScope.User.LocalItems.Rules)
	}

	// Should NOT be in flat LocalItems anymore
	if profile.LocalItems != nil {
		t.Errorf("expected flat LocalItems to be nil for multi-scope snapshot, got %v", profile.LocalItems)
	}
}
