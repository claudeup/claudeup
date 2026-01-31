// ABOUTME: Tests for Install function that copies items from external paths to .library
// ABOUTME: Covers single file, directory, container detection, skip existing, and auto-enable
package local

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInstall_SingleFile(t *testing.T) {
	// Setup: Create temp claude dir and a source file outside of it
	claudeDir := t.TempDir()
	sourceDir := t.TempDir()

	manager := NewManager(claudeDir)

	// Create source file
	sourceFile := filepath.Join(sourceDir, "my-hook.sh")
	if err := os.WriteFile(sourceFile, []byte("#!/bin/bash\necho hello"), 0755); err != nil {
		t.Fatal(err)
	}

	// Act: Install the file
	installed, skipped, err := manager.Install(CategoryHooks, sourceFile)
	if err != nil {
		t.Fatalf("Install failed: %v", err)
	}

	// Assert: File was installed
	if len(installed) != 1 || installed[0] != "my-hook.sh" {
		t.Errorf("expected [my-hook.sh], got %v", installed)
	}
	if len(skipped) != 0 {
		t.Errorf("expected no skipped, got %v", skipped)
	}

	// Assert: File exists in .library
	libraryFile := filepath.Join(claudeDir, ".library", "hooks", "my-hook.sh")
	if _, err := os.Stat(libraryFile); err != nil {
		t.Errorf("file not in library: %v", err)
	}

	// Assert: Item is enabled in config
	config, err := manager.LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if !config[CategoryHooks]["my-hook.sh"] {
		t.Error("item not enabled in config")
	}

	// Assert: Symlink was created
	symlinkPath := filepath.Join(claudeDir, "hooks", "my-hook.sh")
	info, err := os.Lstat(symlinkPath)
	if err != nil {
		t.Errorf("symlink not created: %v", err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Error("expected symlink, got regular file")
	}
}

func TestInstall_SingleDirectory(t *testing.T) {
	claudeDir := t.TempDir()
	sourceDir := t.TempDir()

	manager := NewManager(claudeDir)

	// Create source skill directory with SKILL.md
	skillDir := filepath.Join(sourceDir, "my-skill")
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("# My Skill"), 0644); err != nil {
		t.Fatal(err)
	}

	// Act
	installed, _, err := manager.Install(CategorySkills, skillDir)
	if err != nil {
		t.Fatalf("Install failed: %v", err)
	}

	// Assert
	if len(installed) != 1 || installed[0] != "my-skill" {
		t.Errorf("expected [my-skill], got %v", installed)
	}

	// Assert: Directory exists in .library with contents
	librarySkill := filepath.Join(claudeDir, ".library", "skills", "my-skill", "SKILL.md")
	if _, err := os.Stat(librarySkill); err != nil {
		t.Errorf("skill not in library: %v", err)
	}
}

func TestInstall_ContainerOfMultipleItems(t *testing.T) {
	claudeDir := t.TempDir()
	sourceDir := t.TempDir()

	manager := NewManager(claudeDir)

	// Create a container directory with multiple command files
	containerDir := filepath.Join(sourceDir, "my-commands")
	if err := os.MkdirAll(containerDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(containerDir, "cmd1.md"), []byte("# Cmd 1"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(containerDir, "cmd2.md"), []byte("# Cmd 2"), 0644); err != nil {
		t.Fatal(err)
	}

	// Act: Install the container
	installed, _, err := manager.Install(CategoryCommands, containerDir)
	if err != nil {
		t.Fatalf("Install failed: %v", err)
	}

	// Assert: Both items were installed individually
	if len(installed) != 2 {
		t.Errorf("expected 2 installed, got %d: %v", len(installed), installed)
	}

	// Assert: Each file exists in .library
	for _, name := range []string{"cmd1.md", "cmd2.md"} {
		path := filepath.Join(claudeDir, ".library", "commands", name)
		if _, err := os.Stat(path); err != nil {
			t.Errorf("command %s not in library: %v", name, err)
		}
	}
}

func TestInstall_SkipsExisting(t *testing.T) {
	claudeDir := t.TempDir()
	sourceDir := t.TempDir()

	manager := NewManager(claudeDir)

	// Pre-create existing item in library
	libraryDir := filepath.Join(claudeDir, ".library", "hooks")
	if err := os.MkdirAll(libraryDir, 0755); err != nil {
		t.Fatal(err)
	}
	existingFile := filepath.Join(libraryDir, "existing.sh")
	if err := os.WriteFile(existingFile, []byte("original content"), 0755); err != nil {
		t.Fatal(err)
	}

	// Create source file with same name but different content
	sourceFile := filepath.Join(sourceDir, "existing.sh")
	if err := os.WriteFile(sourceFile, []byte("new content"), 0755); err != nil {
		t.Fatal(err)
	}

	// Act
	installed, skipped, err := manager.Install(CategoryHooks, sourceFile)
	if err != nil {
		t.Fatalf("Install failed: %v", err)
	}

	// Assert: Item was skipped, not overwritten
	if len(installed) != 0 {
		t.Errorf("expected no installed, got %v", installed)
	}
	if len(skipped) != 1 || skipped[0] != "existing.sh" {
		t.Errorf("expected [existing.sh] skipped, got %v", skipped)
	}

	// Assert: Original content preserved
	content, _ := os.ReadFile(existingFile)
	if string(content) != "original content" {
		t.Errorf("existing file was overwritten: %s", content)
	}
}

func TestInstall_InvalidCategory(t *testing.T) {
	manager := NewManager(t.TempDir())

	_, _, err := manager.Install("invalid", "/some/path")
	if err == nil {
		t.Error("expected error for invalid category")
	}
}

func TestInstall_SourceNotFound(t *testing.T) {
	manager := NewManager(t.TempDir())

	_, _, err := manager.Install(CategoryHooks, "/nonexistent/path")
	if err == nil {
		t.Error("expected error for nonexistent source")
	}
}

func TestInstall_AgentGroup(t *testing.T) {
	claudeDir := t.TempDir()
	sourceDir := t.TempDir()

	manager := NewManager(claudeDir)

	// Create agent group directory with agent files
	agentGroup := filepath.Join(sourceDir, "my-agents")
	if err := os.MkdirAll(agentGroup, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(agentGroup, "agent1.md"), []byte("# Agent 1"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(agentGroup, "agent2.md"), []byte("# Agent 2"), 0644); err != nil {
		t.Fatal(err)
	}

	// Act
	installed, _, err := manager.Install(CategoryAgents, agentGroup)
	if err != nil {
		t.Fatalf("Install failed: %v", err)
	}

	// Assert: Agent group was installed as a directory
	if len(installed) != 1 || installed[0] != "my-agents" {
		t.Errorf("expected [my-agents], got %v", installed)
	}

	// Assert: Group directory exists with agents inside
	agentFile := filepath.Join(claudeDir, ".library", "agents", "my-agents", "agent1.md")
	if _, err := os.Stat(agentFile); err != nil {
		t.Errorf("agent not in library: %v", err)
	}
}
