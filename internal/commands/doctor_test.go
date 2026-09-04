// ABOUTME: Unit tests for doctor path-repair logic
// ABOUTME: Verifies getExpectedPath handles all known marketplace path patterns
package commands

import (
	"errors"
	"io/fs"
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

		results, _ := checkDirectorySymlinks(claudeDir)

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

		results, _ := checkDirectorySymlinks(claudeDir)

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

		results, _ := checkDirectorySymlinks(claudeDir)

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

		results, _ := checkDirectorySymlinks(claudeDir)

		if len(results) != 0 {
			t.Fatalf("expected 0 directory symlinks (skill dir should be excluded), got %d", len(results))
		}
	})

	t.Run("skips missing category directories", func(t *testing.T) {
		claudeDir := t.TempDir()
		// Don't create any category dirs

		results, unchecked := checkDirectorySymlinks(claudeDir)

		if len(results) != 0 {
			t.Fatalf("expected 0 directory symlinks, got %d", len(results))
		}
		if len(unchecked) != 0 {
			t.Fatalf("expected 0 unchecked paths, got %d", len(unchecked))
		}
	})

	t.Run("reports unresolvable symlink as unchecked", func(t *testing.T) {
		claudeDir := t.TempDir()

		agentsDir := filepath.Join(claudeDir, "agents")
		if err := os.MkdirAll(agentsDir, 0o755); err != nil {
			t.Fatal(err)
		}

		// A self-referencing symlink fails EvalSymlinks with ELOOP, not ErrNotExist
		loop := filepath.Join(agentsDir, "loop")
		if err := os.Symlink("loop", loop); err != nil {
			t.Fatal(err)
		}

		results, unchecked := checkDirectorySymlinks(claudeDir)

		if len(results) != 0 {
			t.Fatalf("expected 0 directory symlinks, got %d", len(results))
		}
		if len(unchecked) != 1 {
			t.Fatalf("expected 1 unchecked path, got %d", len(unchecked))
		}
		if unchecked[0].Path != loop {
			t.Errorf("expected unchecked path %q, got %q", loop, unchecked[0].Path)
		}
		if unchecked[0].Err == nil {
			t.Error("expected unchecked entry to carry the underlying error")
		}
	})

	t.Run("does not report dangling symlink as unchecked", func(t *testing.T) {
		claudeDir := t.TempDir()

		agentsDir := filepath.Join(claudeDir, "agents")
		if err := os.MkdirAll(agentsDir, 0o755); err != nil {
			t.Fatal(err)
		}

		// Dangling symlinks are checkBrokenSymlinks' job; reporting them here too
		// would double-count them in doctor output
		if err := os.Symlink(filepath.Join(claudeDir, "missing"), filepath.Join(agentsDir, "dangling")); err != nil {
			t.Fatal(err)
		}

		results, unchecked := checkDirectorySymlinks(claudeDir)

		if len(results) != 0 {
			t.Fatalf("expected 0 directory symlinks, got %d", len(results))
		}
		if len(unchecked) != 0 {
			t.Fatalf("expected 0 unchecked paths, got %d", len(unchecked))
		}
	})
}

func TestCheckBrokenSymlinks(t *testing.T) {
	t.Run("reports dangling symlink as broken, not unchecked", func(t *testing.T) {
		claudeDir := t.TempDir()

		agentsDir := filepath.Join(claudeDir, "agents")
		if err := os.MkdirAll(agentsDir, 0o755); err != nil {
			t.Fatal(err)
		}

		target := filepath.Join(claudeDir, "missing.md")
		dangling := filepath.Join(agentsDir, "dangling.md")
		if err := os.Symlink(target, dangling); err != nil {
			t.Fatal(err)
		}

		broken, unchecked := checkBrokenSymlinks(claudeDir)

		if len(broken) != 1 {
			t.Fatalf("expected 1 broken symlink, got %d", len(broken))
		}
		if broken[0].Path != dangling || broken[0].Target != target {
			t.Errorf("expected broken symlink %q -> %q, got %q -> %q", dangling, target, broken[0].Path, broken[0].Target)
		}
		if len(unchecked) != 0 {
			t.Fatalf("expected 0 unchecked paths, got %d", len(unchecked))
		}
	})

	t.Run("reports symlink loop as unchecked, not broken", func(t *testing.T) {
		claudeDir := t.TempDir()

		agentsDir := filepath.Join(claudeDir, "agents")
		if err := os.MkdirAll(agentsDir, 0o755); err != nil {
			t.Fatal(err)
		}

		// os.Stat on a self-referencing symlink fails with ELOOP, not ErrNotExist
		loop := filepath.Join(agentsDir, "loop.md")
		if err := os.Symlink("loop.md", loop); err != nil {
			t.Fatal(err)
		}

		broken, unchecked := checkBrokenSymlinks(claudeDir)

		if len(broken) != 0 {
			t.Fatalf("expected 0 broken symlinks, got %d", len(broken))
		}
		if len(unchecked) != 1 {
			t.Fatalf("expected 1 unchecked path, got %d", len(unchecked))
		}
		if unchecked[0].Path != loop {
			t.Errorf("expected unchecked path %q, got %q", loop, unchecked[0].Path)
		}
		if unchecked[0].Err == nil {
			t.Error("expected unchecked entry to carry the underlying error")
		}
	})

	t.Run("reports unreadable subdirectory as unchecked", func(t *testing.T) {
		if os.Geteuid() == 0 {
			t.Skip("running as root: permission bits are not enforced")
		}

		claudeDir := t.TempDir()

		agentsDir := filepath.Join(claudeDir, "agents")
		lockedDir := filepath.Join(agentsDir, "locked")
		if err := os.MkdirAll(lockedDir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(lockedDir, "hidden.md"), []byte("agent"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(lockedDir, 0o000); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = os.Chmod(lockedDir, 0o755) })

		broken, unchecked := checkBrokenSymlinks(claudeDir)

		if len(broken) != 0 {
			t.Fatalf("expected 0 broken symlinks, got %d", len(broken))
		}
		if len(unchecked) != 1 {
			t.Fatalf("expected 1 unchecked path, got %d", len(unchecked))
		}
		if unchecked[0].Path != lockedDir {
			t.Errorf("expected unchecked path %q, got %q", lockedDir, unchecked[0].Path)
		}
		if unchecked[0].Err == nil {
			t.Error("expected unchecked entry to carry the underlying error")
		}
	})

	t.Run("skips missing category directories", func(t *testing.T) {
		claudeDir := t.TempDir()

		broken, unchecked := checkBrokenSymlinks(claudeDir)

		if len(broken) != 0 {
			t.Fatalf("expected 0 broken symlinks, got %d", len(broken))
		}
		if len(unchecked) != 0 {
			t.Fatalf("expected 0 unchecked paths, got %d", len(unchecked))
		}
	})
}

func TestMergeUncheckedPaths(t *testing.T) {
	errWalk := errors.New("from walk")
	errDirs := errors.New("from dirs")

	t.Run("keeps first entry for a duplicate path and sorts by path", func(t *testing.T) {
		fromWalk := []UncheckedPath{
			{Path: "/claude/agents/loop", Err: errWalk},
			{Path: "/claude/skills/locked", Err: errWalk},
		}
		fromDirs := []UncheckedPath{
			{Path: "/claude/agents/loop", Err: errDirs},
			{Path: "/claude/agents/alpha", Err: errDirs},
		}

		merged := mergeUncheckedPaths(fromWalk, fromDirs)

		if len(merged) != 3 {
			t.Fatalf("expected 3 merged paths, got %d", len(merged))
		}
		wantPaths := []string{"/claude/agents/alpha", "/claude/agents/loop", "/claude/skills/locked"}
		for i, want := range wantPaths {
			if merged[i].Path != want {
				t.Errorf("merged[%d].Path = %q, want %q", i, merged[i].Path, want)
			}
		}
		if merged[1].Err != errWalk {
			t.Errorf("expected the first-seen error for the duplicate path, got %v", merged[1].Err)
		}
	})

	t.Run("returns nil for no input", func(t *testing.T) {
		if merged := mergeUncheckedPaths(nil, nil); merged != nil {
			t.Fatalf("expected nil, got %v", merged)
		}
	})
}

func TestUnderlyingError(t *testing.T) {
	t.Run("returns empty string for nil", func(t *testing.T) {
		if got := underlyingError(nil); got != "" {
			t.Errorf("underlyingError(nil) = %q, want empty string", got)
		}
	})

	t.Run("strips the operation and path from a PathError", func(t *testing.T) {
		err := &fs.PathError{Op: "stat", Path: "/some/path", Err: errors.New("permission denied")}
		if got := underlyingError(err); got != "permission denied" {
			t.Errorf("underlyingError(PathError) = %q, want %q", got, "permission denied")
		}
	})

	t.Run("returns other errors unchanged", func(t *testing.T) {
		err := errors.New("plain error")
		if got := underlyingError(err); got != "plain error" {
			t.Errorf("underlyingError(plain) = %q, want %q", got, "plain error")
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
