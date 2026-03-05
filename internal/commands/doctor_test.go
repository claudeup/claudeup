// ABOUTME: Unit tests for doctor path-repair logic
// ABOUTME: Verifies getExpectedPath handles all known marketplace path patterns
package commands

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCheckDirectorySymlinks(t *testing.T) {
	t.Run("detects symlink pointing to directory", func(t *testing.T) {
		claudeDir := t.TempDir()
		extDir := t.TempDir()

		// Create a category directory in claudeDir
		agentsDir := filepath.Join(claudeDir, "agents")
		if err := os.MkdirAll(agentsDir, 0o755); err != nil {
			t.Fatal(err)
		}

		// Create a source directory with some agent files
		groupDir := filepath.Join(extDir, "developer-experience")
		if err := os.MkdirAll(groupDir, 0o755); err != nil {
			t.Fatal(err)
		}
		for _, name := range []string{"build-engineer.md", "cli-developer.md"} {
			if err := os.WriteFile(filepath.Join(groupDir, name), []byte("agent"), 0o644); err != nil {
				t.Fatal(err)
			}
		}

		// Create directory symlink pointing to the group directory
		if err := os.Symlink(groupDir, filepath.Join(agentsDir, "developer-experience")); err != nil {
			t.Fatal(err)
		}

		results := checkDirectorySymlinks(claudeDir)

		if len(results) != 1 {
			t.Fatalf("expected 1 directory symlink, got %d", len(results))
		}
		if results[0].Category != "agents" {
			t.Errorf("expected category 'agents', got %q", results[0].Category)
		}
		if results[0].ItemCount != 2 {
			t.Errorf("expected 2 exposed items, got %d", results[0].ItemCount)
		}
	})

	t.Run("ignores regular file symlinks", func(t *testing.T) {
		claudeDir := t.TempDir()
		extDir := t.TempDir()

		agentsDir := filepath.Join(claudeDir, "agents")
		if err := os.MkdirAll(agentsDir, 0o755); err != nil {
			t.Fatal(err)
		}

		// Create a source file
		srcFile := filepath.Join(extDir, "my-agent.md")
		if err := os.WriteFile(srcFile, []byte("agent"), 0o644); err != nil {
			t.Fatal(err)
		}

		// Create file symlink to a single agent
		if err := os.Symlink(srcFile, filepath.Join(agentsDir, "my-agent.md")); err != nil {
			t.Fatal(err)
		}

		results := checkDirectorySymlinks(claudeDir)

		if len(results) != 0 {
			t.Fatalf("expected 0 directory symlinks, got %d", len(results))
		}
	})

	t.Run("ignores non-symlink directories", func(t *testing.T) {
		claudeDir := t.TempDir()

		// Create a real subdirectory (not a symlink) -- this is normal for grouped agents
		agentsDir := filepath.Join(claudeDir, "agents", "test-runner")
		if err := os.MkdirAll(agentsDir, 0o755); err != nil {
			t.Fatal(err)
		}

		results := checkDirectorySymlinks(claudeDir)

		if len(results) != 0 {
			t.Fatalf("expected 0 directory symlinks, got %d", len(results))
		}
	})

	t.Run("ignores skill directories with SKILL.md", func(t *testing.T) {
		claudeDir := t.TempDir()
		extDir := t.TempDir()

		skillsDir := filepath.Join(claudeDir, "skills")
		if err := os.MkdirAll(skillsDir, 0o755); err != nil {
			t.Fatal(err)
		}

		// Create a skill directory with SKILL.md (legitimate directory symlink)
		skillDir := filepath.Join(extDir, "golang")
		if err := os.MkdirAll(skillDir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("skill"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(skillDir, "helpers.md"), []byte("ref"), 0o644); err != nil {
			t.Fatal(err)
		}

		if err := os.Symlink(skillDir, filepath.Join(skillsDir, "golang")); err != nil {
			t.Fatal(err)
		}

		results := checkDirectorySymlinks(claudeDir)

		if len(results) != 0 {
			t.Fatalf("expected 0 directory symlinks (skill dir should be excluded), got %d", len(results))
		}
	})

	t.Run("skips missing category directories", func(t *testing.T) {
		claudeDir := t.TempDir()
		// Don't create any category dirs

		results := checkDirectorySymlinks(claudeDir)

		if len(results) != 0 {
			t.Fatalf("expected 0 directory symlinks, got %d", len(results))
		}
	})
}

func TestGetExpectedPath(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "claude-code-plugins adds plugins subdirectory",
			input:    "/home/user/.claude/plugins/marketplaces/claude-code-plugins/my-plugin",
			expected: "/home/user/.claude/plugins/marketplaces/claude-code-plugins/plugins/my-plugin",
		},
		{
			name:     "claude-plugins-official adds plugins subdirectory",
			input:    "/home/user/.claude/plugins/marketplaces/claude-plugins-official/my-plugin",
			expected: "/home/user/.claude/plugins/marketplaces/claude-plugins-official/plugins/my-plugin",
		},
		{
			name:     "claude-code-templates adds plugins subdirectory",
			input:    "/home/user/.claude/plugins/marketplaces/claude-code-templates/my-plugin",
			expected: "/home/user/.claude/plugins/marketplaces/claude-code-templates/plugins/my-plugin",
		},
		{
			name:     "every-marketplace adds plugins subdirectory",
			input:    "/home/user/.claude/plugins/marketplaces/every-marketplace/my-plugin",
			expected: "/home/user/.claude/plugins/marketplaces/every-marketplace/plugins/my-plugin",
		},
		{
			name:     "awesome-claude-code-plugins adds plugins subdirectory",
			input:    "/home/user/.claude/plugins/marketplaces/awesome-claude-code-plugins/my-plugin",
			expected: "/home/user/.claude/plugins/marketplaces/awesome-claude-code-plugins/plugins/my-plugin",
		},
		{
			name:     "anthropic-agent-skills adds skills subdirectory",
			input:    "/home/user/.claude/plugins/marketplaces/anthropic-agent-skills/my-skill",
			expected: "/home/user/.claude/plugins/marketplaces/anthropic-agent-skills/skills/my-skill",
		},
		{
			name:     "platform-k8s-architect removes duplicate directory",
			input:    "/home/user/.claude/plugins/marketplaces/platform-k8s-architect/platform-k8s-architect",
			expected: "/home/user/.claude/plugins/marketplaces/platform-k8s-architect",
		},
		{
			name:     "platform-k8s-architect non-duplicate returns empty",
			input:    "/home/user/.claude/plugins/marketplaces/platform-k8s-architect/other-plugin",
			expected: "",
		},
		{
			name:     "unknown marketplace returns empty",
			input:    "/home/user/.claude/plugins/marketplaces/unknown-marketplace/my-plugin",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getExpectedPath(tt.input)
			if got != tt.expected {
				t.Errorf("getExpectedPath(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}
