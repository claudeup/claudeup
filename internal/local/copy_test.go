// ABOUTME: Tests for CopyToProject - copies local items into project .claude/ directory
// ABOUTME: Validates file copy, directory creation, skill dirs, wildcards, and overwrite
package local

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCopyToProjectCopiesFiles(t *testing.T) {
	localDir := t.TempDir()
	projectDir := t.TempDir()

	// Create source files in local storage
	rulesDir := filepath.Join(localDir, "rules")
	if err := os.MkdirAll(rulesDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(rulesDir, "golang.md"), []byte("# Go Rules"), 0644); err != nil {
		t.Fatal(err)
	}

	copied, notFound, err := CopyToProject(localDir, "rules", []string{"golang.md"}, projectDir)
	if err != nil {
		t.Fatalf("CopyToProject failed: %v", err)
	}

	if len(notFound) != 0 {
		t.Errorf("expected no notFound, got %v", notFound)
	}
	if len(copied) != 1 || copied[0] != "golang.md" {
		t.Errorf("expected copied [golang.md], got %v", copied)
	}

	// Verify file was copied
	destPath := filepath.Join(projectDir, ".claude", "rules", "golang.md")
	content, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatalf("expected file at %s: %v", destPath, err)
	}
	if string(content) != "# Go Rules" {
		t.Errorf("expected '# Go Rules', got %q", string(content))
	}

	// Verify it's a regular file, not a symlink
	info, err := os.Lstat(destPath)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		t.Error("expected regular file, got symlink")
	}
}

func TestCopyToProjectCreatesDirectories(t *testing.T) {
	localDir := t.TempDir()
	projectDir := t.TempDir()

	// Create a nested agent file
	agentDir := filepath.Join(localDir, "agents", "review-team")
	if err := os.MkdirAll(agentDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(agentDir, "reviewer.md"), []byte("# Reviewer"), 0644); err != nil {
		t.Fatal(err)
	}

	copied, _, err := CopyToProject(localDir, "agents", []string{"review-team/reviewer.md"}, projectDir)
	if err != nil {
		t.Fatalf("CopyToProject failed: %v", err)
	}

	if len(copied) != 1 {
		t.Errorf("expected 1 copied, got %d", len(copied))
	}

	// Verify nested directory was created
	destPath := filepath.Join(projectDir, ".claude", "agents", "review-team", "reviewer.md")
	if _, err := os.Stat(destPath); os.IsNotExist(err) {
		t.Errorf("expected file at %s", destPath)
	}
}

func TestCopyToProjectHandlesSkillDirectories(t *testing.T) {
	localDir := t.TempDir()
	projectDir := t.TempDir()

	// Create a skill directory with SKILL.md and supporting files
	skillDir := filepath.Join(localDir, "skills", "session-notes")
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("# Skill"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "helper.md"), []byte("# Helper"), 0644); err != nil {
		t.Fatal(err)
	}

	copied, _, err := CopyToProject(localDir, "skills", []string{"session-notes"}, projectDir)
	if err != nil {
		t.Fatalf("CopyToProject failed: %v", err)
	}

	if len(copied) != 1 || copied[0] != "session-notes" {
		t.Errorf("expected copied [session-notes], got %v", copied)
	}

	// Verify entire directory was copied
	destSkillMd := filepath.Join(projectDir, ".claude", "skills", "session-notes", "SKILL.md")
	if _, err := os.Stat(destSkillMd); os.IsNotExist(err) {
		t.Error("expected SKILL.md to be copied")
	}
	destHelperMd := filepath.Join(projectDir, ".claude", "skills", "session-notes", "helper.md")
	if _, err := os.Stat(destHelperMd); os.IsNotExist(err) {
		t.Error("expected helper.md to be copied")
	}
}

func TestCopyToProjectWithWildcards(t *testing.T) {
	localDir := t.TempDir()
	projectDir := t.TempDir()

	// Create multiple rules
	rulesDir := filepath.Join(localDir, "rules")
	if err := os.MkdirAll(rulesDir, 0755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"golang.md", "go-testing.md", "bash.md"} {
		if err := os.WriteFile(filepath.Join(rulesDir, name), []byte("# "+name), 0644); err != nil {
			t.Fatal(err)
		}
	}

	copied, _, err := CopyToProject(localDir, "rules", []string{"go*"}, projectDir)
	if err != nil {
		t.Fatalf("CopyToProject failed: %v", err)
	}

	if len(copied) != 2 {
		t.Errorf("expected 2 copied (go* matching golang.md, go-testing.md), got %d: %v", len(copied), copied)
	}

	// bash.md should not be copied
	bashPath := filepath.Join(projectDir, ".claude", "rules", "bash.md")
	if _, err := os.Stat(bashPath); !os.IsNotExist(err) {
		t.Error("bash.md should not have been copied")
	}
}

func TestCopyToProjectOverwrites(t *testing.T) {
	localDir := t.TempDir()
	projectDir := t.TempDir()

	// Create source
	rulesDir := filepath.Join(localDir, "rules")
	if err := os.MkdirAll(rulesDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(rulesDir, "golang.md"), []byte("updated content"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create pre-existing file at destination
	destDir := filepath.Join(projectDir, ".claude", "rules")
	if err := os.MkdirAll(destDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(destDir, "golang.md"), []byte("old content"), 0644); err != nil {
		t.Fatal(err)
	}

	_, _, err := CopyToProject(localDir, "rules", []string{"golang.md"}, projectDir)
	if err != nil {
		t.Fatalf("CopyToProject failed: %v", err)
	}

	// Verify overwritten
	content, err := os.ReadFile(filepath.Join(destDir, "golang.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "updated content" {
		t.Errorf("expected 'updated content', got %q", string(content))
	}
}

func TestCopyToProjectNotFound(t *testing.T) {
	localDir := t.TempDir()
	projectDir := t.TempDir()

	// Create rules dir but no matching files
	if err := os.MkdirAll(filepath.Join(localDir, "rules"), 0755); err != nil {
		t.Fatal(err)
	}

	copied, notFound, err := CopyToProject(localDir, "rules", []string{"nonexistent.md"}, projectDir)
	if err != nil {
		t.Fatalf("CopyToProject failed: %v", err)
	}

	if len(copied) != 0 {
		t.Errorf("expected 0 copied, got %v", copied)
	}
	if len(notFound) != 1 || notFound[0] != "nonexistent.md" {
		t.Errorf("expected notFound [nonexistent.md], got %v", notFound)
	}
}

func TestProjectScopeCategories(t *testing.T) {
	// agents and rules should be valid project-scope categories
	if !ProjectScopeCategories[CategoryAgents] {
		t.Error("expected agents to be a valid project-scope category")
	}
	if !ProjectScopeCategories[CategoryRules] {
		t.Error("expected rules to be a valid project-scope category")
	}

	// commands, skills, hooks, output-styles should NOT be
	for _, cat := range []string{CategoryCommands, CategorySkills, CategoryHooks, CategoryOutputStyles} {
		if ProjectScopeCategories[cat] {
			t.Errorf("expected %s to NOT be a valid project-scope category", cat)
		}
	}
}

func TestPathTraversalValidation(t *testing.T) {
	// Defense-in-depth: filesystem entries can't contain "/" or ".." as names,
	// so traversal through listLocalItems is impossible. This validates the
	// prefix check catches traversal if item names were ever crafted externally.
	projectDir := "/project"
	destBase := filepath.Clean(filepath.Join(projectDir, ".claude", "rules"))

	tests := []struct {
		name    string
		item    string
		escapes bool
	}{
		{"normal item", "golang.md", false},
		{"nested item", "team/reviewer.md", false},
		{"traversal escapes category", "../agents/evil.md", true},
		{"traversal escapes .claude", "../../etc/evil.md", true},
		{"dot-dot only", "..", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			destPath := filepath.Clean(filepath.Join(destBase, tt.item))
			escaped := !strings.HasPrefix(destPath, destBase+string(filepath.Separator))
			if escaped != tt.escapes {
				t.Errorf("item %q: expected escapes=%v, got %v (resolved to %s)", tt.item, tt.escapes, escaped, destPath)
			}
		})
	}
}

func TestValidateProjectScopeCategories(t *testing.T) {
	// Valid categories
	if err := ValidateProjectScope("agents"); err != nil {
		t.Errorf("agents should be valid: %v", err)
	}
	if err := ValidateProjectScope("rules"); err != nil {
		t.Errorf("rules should be valid: %v", err)
	}

	// Invalid categories
	if err := ValidateProjectScope("commands"); err == nil {
		t.Error("commands should be invalid for project scope")
	}
	if err := ValidateProjectScope("skills"); err == nil {
		t.Error("skills should be invalid for project scope")
	}
}
