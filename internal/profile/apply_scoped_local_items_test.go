// ABOUTME: Tests for scope-aware extension application
// ABOUTME: Validates user-scope symlinks, project-scope copies, and backward compat
package profile

import (
	"os"
	"path/filepath"
	"testing"
)

func TestApplyExtensionsProjectScopeCopiesFiles(t *testing.T) {
	tempDir := t.TempDir()
	claudeDir := filepath.Join(tempDir, ".claude")
	claudeupHome := filepath.Join(tempDir, ".claudeup")
	projectDir := filepath.Join(tempDir, "project")

	mustMkdir(t, claudeDir)
	mustMkdir(t, projectDir)

	// Create extensions in claudeup storage
	rulesDir := filepath.Join(claudeupHome, "ext", "rules")
	mustMkdir(t, rulesDir)
	mustWriteFile(t, filepath.Join(rulesDir, "golang.md"), "# Go Rules")

	agentsDir := filepath.Join(claudeupHome, "ext", "agents")
	mustMkdir(t, agentsDir)
	mustWriteFile(t, filepath.Join(agentsDir, "reviewer.md"), "# Reviewer Agent")

	profile := &Profile{
		Name: "test",
	}

	localItems := &ExtensionSettings{
		Rules:  []string{"golang.md"},
		Agents: []string{"reviewer.md"},
	}

	// Apply at project scope
	err := applyExtensionsScoped(profile, localItems, ScopeProject, claudeDir, claudeupHome, projectDir)
	if err != nil {
		t.Fatalf("applyExtensionsScoped failed: %v", err)
	}

	// Verify files were COPIED (not symlinked) to project .claude/
	rulesPath := filepath.Join(projectDir, ".claude", "rules", "golang.md")
	content, err := os.ReadFile(rulesPath)
	if err != nil {
		t.Fatalf("expected file at %s: %v", rulesPath, err)
	}
	if string(content) != "# Go Rules" {
		t.Errorf("expected '# Go Rules', got %q", string(content))
	}
	// Verify it's a regular file, not a symlink
	info, err := os.Lstat(rulesPath)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		t.Error("expected regular file at project scope, got symlink")
	}

	agentsPath := filepath.Join(projectDir, ".claude", "agents", "reviewer.md")
	if _, err := os.Stat(agentsPath); os.IsNotExist(err) {
		t.Error("expected reviewer.md to be copied to project .claude/agents/")
	}
}

func TestApplyExtensionsProjectScopeRejectsUnsupportedCategories(t *testing.T) {
	tempDir := t.TempDir()
	claudeDir := filepath.Join(tempDir, ".claude")
	claudeupHome := filepath.Join(tempDir, ".claudeup")
	projectDir := filepath.Join(tempDir, "project")

	mustMkdir(t, claudeDir)
	mustMkdir(t, projectDir)

	// Create a skill in extension storage
	skillDir := filepath.Join(claudeupHome, "ext", "skills", "test-skill")
	mustMkdir(t, skillDir)
	mustWriteFile(t, filepath.Join(skillDir, "SKILL.md"), "# Skill")

	profile := &Profile{Name: "test"}
	localItems := &ExtensionSettings{
		Skills: []string{"test-skill"},
	}

	// Apply at project scope should fail for unsupported category
	err := applyExtensionsScoped(profile, localItems, ScopeProject, claudeDir, claudeupHome, projectDir)
	if err == nil {
		t.Fatal("expected error for unsupported category at project scope")
	}
}

func TestApplyExtensionsUserScopeUsesSymlinks(t *testing.T) {
	tempDir := t.TempDir()
	claudeDir := filepath.Join(tempDir, ".claude")
	claudeupHome := filepath.Join(tempDir, ".claudeup")

	mustMkdir(t, claudeDir)

	// Create a rule in extension storage
	rulesDir := filepath.Join(claudeupHome, "ext", "rules")
	mustMkdir(t, rulesDir)
	mustWriteFile(t, filepath.Join(rulesDir, "golang.md"), "# Go Rules")

	// Create enabled.json so Manager works
	mustWriteFile(t, filepath.Join(claudeupHome, "enabled.json"), "{}")

	profile := &Profile{Name: "test"}
	localItems := &ExtensionSettings{
		Rules: []string{"golang.md"},
	}

	// Apply at user scope should use symlinks (existing behavior)
	err := applyExtensionsScoped(profile, localItems, ScopeUser, claudeDir, claudeupHome, "")
	if err != nil {
		t.Fatalf("applyExtensionsScoped failed: %v", err)
	}

	// Verify it created a symlink
	rulesPath := filepath.Join(claudeDir, "rules", "golang.md")
	info, err := os.Lstat(rulesPath)
	if err != nil {
		t.Fatalf("expected file at %s: %v", rulesPath, err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Error("expected symlink at user scope, got regular file")
	}
}

func TestApplyExtensionsBackwardCompatFlatField(t *testing.T) {
	tempDir := t.TempDir()
	claudeDir := filepath.Join(tempDir, ".claude")
	claudeupHome := filepath.Join(tempDir, ".claudeup")

	mustMkdir(t, claudeDir)

	// Create a rule in extension storage
	rulesDir := filepath.Join(claudeupHome, "ext", "rules")
	mustMkdir(t, rulesDir)
	mustWriteFile(t, filepath.Join(rulesDir, "golang.md"), "# Go Rules")
	mustWriteFile(t, filepath.Join(claudeupHome, "enabled.json"), "{}")

	// Profile with flat Extensions (legacy format)
	profile := &Profile{
		Name: "test",
		Extensions: &ExtensionSettings{
			Rules: []string{"golang.md"},
		},
	}

	// applyExtensions (the existing function) should still work
	err := applyExtensions(profile, claudeDir, claudeupHome)
	if err != nil {
		t.Fatalf("applyExtensions failed: %v", err)
	}

	// Verify symlink was created
	rulesPath := filepath.Join(claudeDir, "rules", "golang.md")
	info, err := os.Lstat(rulesPath)
	if err != nil {
		t.Fatalf("expected file at %s: %v", rulesPath, err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Error("expected symlink for backward compat flat Extensions")
	}
}

func TestApplyAllScopesWithScopedExtensions(t *testing.T) {
	tempDir := t.TempDir()
	claudeDir := filepath.Join(tempDir, ".claude")
	claudeupHome := filepath.Join(tempDir, ".claudeup")
	projectDir := filepath.Join(tempDir, "project")
	claudeJSONPath := filepath.Join(claudeDir, ".claude.json")

	mustMkdir(t, claudeDir)
	mustMkdir(t, filepath.Join(claudeDir, "plugins"))
	mustMkdir(t, projectDir)
	mustMkdir(t, filepath.Join(projectDir, ".claude"))

	// Initialize required files
	mustWriteJSON(t, filepath.Join(claudeDir, "settings.json"), map[string]any{"enabledPlugins": map[string]bool{}})
	mustWriteJSON(t, filepath.Join(claudeDir, "plugins", "known_marketplaces.json"), map[string]any{})
	mustWriteJSON(t, claudeJSONPath, map[string]any{"mcpServers": map[string]any{}})
	mustWriteFile(t, filepath.Join(claudeupHome, "enabled.json"), "{}")

	// Create extensions
	userRulesDir := filepath.Join(claudeupHome, "ext", "rules")
	mustMkdir(t, userRulesDir)
	mustWriteFile(t, filepath.Join(userRulesDir, "coding-standards.md"), "# Coding Standards")
	mustWriteFile(t, filepath.Join(userRulesDir, "golang.md"), "# Golang Rules")

	userSkillsDir := filepath.Join(claudeupHome, "ext", "skills", "session-notes")
	mustMkdir(t, userSkillsDir)
	mustWriteFile(t, filepath.Join(userSkillsDir, "SKILL.md"), "# Session Notes Skill")

	agentsDir := filepath.Join(claudeupHome, "ext", "agents")
	mustMkdir(t, agentsDir)
	mustWriteFile(t, filepath.Join(agentsDir, "reviewer.md"), "# Reviewer")

	// Multi-scope profile with scoped extensions
	profile := &Profile{
		Name: "test",
		PerScope: &PerScopeSettings{
			User: &ScopeSettings{
				Extensions: &ExtensionSettings{
					Skills: []string{"session-notes"},
					Rules:  []string{"coding-standards.md"},
				},
			},
			Project: &ScopeSettings{
				Extensions: &ExtensionSettings{
					Rules:  []string{"golang.md"},
					Agents: []string{"reviewer.md"},
				},
			},
		},
	}

	result, err := ApplyAllScopes(profile, claudeDir, claudeJSONPath, projectDir, claudeupHome, nil, nil)
	if err != nil {
		t.Fatalf("ApplyAllScopes failed: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}

	// Verify user-scope items are symlinked in claudeDir
	userSkillPath := filepath.Join(claudeDir, "skills", "session-notes")
	info, err := os.Lstat(userSkillPath)
	if err != nil {
		t.Fatalf("expected user-scope skill symlink: %v", err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Error("expected symlink for user-scope skill")
	}

	userRulePath := filepath.Join(claudeDir, "rules", "coding-standards.md")
	info, err = os.Lstat(userRulePath)
	if err != nil {
		t.Fatalf("expected user-scope rule symlink: %v", err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Error("expected symlink for user-scope rule")
	}

	// Verify project-scope items are COPIED to project .claude/
	projectRulePath := filepath.Join(projectDir, ".claude", "rules", "golang.md")
	content, err := os.ReadFile(projectRulePath)
	if err != nil {
		t.Fatalf("expected project-scope rule at %s: %v", projectRulePath, err)
	}
	if string(content) != "# Golang Rules" {
		t.Errorf("expected '# Golang Rules', got %q", string(content))
	}
	info, err = os.Lstat(projectRulePath)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		t.Error("expected regular file for project-scope rule, got symlink")
	}

	projectAgentPath := filepath.Join(projectDir, ".claude", "agents", "reviewer.md")
	if _, err := os.Stat(projectAgentPath); os.IsNotExist(err) {
		t.Error("expected reviewer.md copied to project .claude/agents/")
	}
}

// Test helpers

func mustWriteFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}
