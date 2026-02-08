// ABOUTME: Tests for resolving and displaying plugin file contents
// ABOUTME: Verifies exact match, extension inference, and skill directory resolution
package commands

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolvePluginFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a plugin directory structure
	agentsDir := filepath.Join(tmpDir, "agents")
	skillsDir := filepath.Join(tmpDir, "skills", "awesome-skill")
	hooksDir := filepath.Join(tmpDir, "hooks")

	if err := os.MkdirAll(agentsDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(skillsDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(hooksDir, 0755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(agentsDir, "test.md"), []byte("# Agent"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(agentsDir, "runner.py"), []byte("# python"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillsDir, "SKILL.md"), []byte("# Skill"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(hooksDir, "check.sh"), []byte("#!/bin/bash"), 0644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		file    string
		want    string
		wantErr bool
		errMsg  string
	}{
		{"exact match with extension", "agents/test.md", filepath.Join(agentsDir, "test.md"), false, ""},
		{"infer .md extension", "agents/test", filepath.Join(agentsDir, "test.md"), false, ""},
		{"infer .py extension", "agents/runner", filepath.Join(agentsDir, "runner.py"), false, ""},
		{"skill directory resolves to SKILL.md", "skills/awesome-skill", filepath.Join(skillsDir, "SKILL.md"), false, ""},
		{"nonexistent file", "agents/nonexistent", "", true, ""},
		{"nonexistent directory", "commands/test", "", true, ""},

		// Path traversal attacks
		{"rejects absolute path", "/etc/passwd", "", true, "path traversal"},
		{"rejects ../ traversal", "../../../etc/passwd", "", true, "path traversal"},
		{"rejects agents/../.. traversal", "agents/../../etc/passwd", "", true, "path traversal"},

		// Non-skill directory gives generic error
		{"non-skill directory gives generic error", "agents", "", true, "is a directory"},
		{"hooks directory gives generic error", "hooks", "", true, "is a directory"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolvePluginFile(tmpDir, tt.file)
			if (err != nil) != tt.wantErr {
				t.Errorf("resolvePluginFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.errMsg != "" && err != nil {
				if !contains(err.Error(), tt.errMsg) {
					t.Errorf("resolvePluginFile() error = %q, want error containing %q", err, tt.errMsg)
				}
			}
			if got != tt.want {
				t.Errorf("resolvePluginFile() = %q, want %q", got, tt.want)
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
