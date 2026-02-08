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
	os.MkdirAll(agentsDir, 0755)
	os.MkdirAll(skillsDir, 0755)

	os.WriteFile(filepath.Join(agentsDir, "test.md"), []byte("# Agent"), 0644)
	os.WriteFile(filepath.Join(agentsDir, "runner.py"), []byte("# python"), 0644)
	os.WriteFile(filepath.Join(skillsDir, "SKILL.md"), []byte("# Skill"), 0644)

	tests := []struct {
		name    string
		file    string
		want    string
		wantErr bool
	}{
		{"exact match with extension", "agents/test.md", filepath.Join(agentsDir, "test.md"), false},
		{"infer .md extension", "agents/test", filepath.Join(agentsDir, "test.md"), false},
		{"infer .py extension", "agents/runner", filepath.Join(agentsDir, "runner.py"), false},
		{"skill directory resolves to SKILL.md", "skills/awesome-skill", filepath.Join(skillsDir, "SKILL.md"), false},
		{"nonexistent file", "agents/nonexistent", "", true},
		{"nonexistent directory", "commands/test", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolvePluginFile(tmpDir, tt.file)
			if (err != nil) != tt.wantErr {
				t.Errorf("resolvePluginFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("resolvePluginFile() = %q, want %q", got, tt.want)
			}
		})
	}
}
