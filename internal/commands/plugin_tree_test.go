// ABOUTME: Unit tests for plugin tree generation
// ABOUTME: Tests tree output formatting with unicode box characters
package commands

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGenerateTree(t *testing.T) {
	// Create temp directory structure
	tempDir := t.TempDir()

	// Create: agents/, skills/foo/, commands/
	os.MkdirAll(filepath.Join(tempDir, "agents"), 0755)
	os.MkdirAll(filepath.Join(tempDir, "skills", "foo"), 0755)
	os.MkdirAll(filepath.Join(tempDir, "commands"), 0755)

	// Create files
	os.WriteFile(filepath.Join(tempDir, "agents", "test.md"), []byte(""), 0644)
	os.WriteFile(filepath.Join(tempDir, "skills", "foo", "SKILL.md"), []byte(""), 0644)

	tree, dirs, files := generateTree(tempDir)

	if tree == "" {
		t.Error("expected non-empty tree output")
	}
	// 4 directories: agents/, skills/, skills/foo/, commands/
	if dirs != 4 {
		t.Errorf("expected 4 directories, got %d", dirs)
	}
	// 2 files: agents/test.md, skills/foo/SKILL.md
	if files != 2 {
		t.Errorf("expected 2 files, got %d", files)
	}
}

func TestGenerateTreeContainsBoxChars(t *testing.T) {
	tempDir := t.TempDir()
	os.MkdirAll(filepath.Join(tempDir, "agents"), 0755)
	os.WriteFile(filepath.Join(tempDir, "agents", "test.md"), []byte(""), 0644)

	tree, _, _ := generateTree(tempDir)

	// Should contain box-drawing characters
	hasBoxChars := false
	for _, r := range tree {
		if r == '├' || r == '└' || r == '│' {
			hasBoxChars = true
			break
		}
	}
	if !hasBoxChars {
		t.Error("expected tree to contain unicode box-drawing characters")
	}
}

func TestGenerateTreeEmptyDir(t *testing.T) {
	tempDir := t.TempDir()

	tree, dirs, files := generateTree(tempDir)

	// Empty directory should return empty tree
	if tree != "" {
		t.Errorf("expected empty tree for empty dir, got: %s", tree)
	}
	if dirs != 0 || files != 0 {
		t.Errorf("expected 0 dirs and 0 files, got %d dirs, %d files", dirs, files)
	}
}
