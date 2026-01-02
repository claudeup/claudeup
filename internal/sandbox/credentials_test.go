// ABOUTME: Unit tests for credential resolution.
// ABOUTME: Tests credential type mapping and mount generation.
package sandbox

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCredentialTypes(t *testing.T) {
	t.Run("git credential has correct paths", func(t *testing.T) {
		cred := GetCredentialType("git")
		if cred == nil {
			t.Fatal("git credential type not found")
		}
		if cred.SourceSuffix != ".gitconfig" {
			t.Errorf("wrong source: got %q, want %q", cred.SourceSuffix, ".gitconfig")
		}
		if cred.TargetPath != "/root/.gitconfig" {
			t.Errorf("wrong target: got %q, want %q", cred.TargetPath, "/root/.gitconfig")
		}
	})

	t.Run("ssh credential has correct paths", func(t *testing.T) {
		cred := GetCredentialType("ssh")
		if cred == nil {
			t.Fatal("ssh credential type not found")
		}
		if cred.SourceSuffix != ".ssh" {
			t.Errorf("wrong source: got %q, want %q", cred.SourceSuffix, ".ssh")
		}
		if cred.TargetPath != "/root/.ssh" {
			t.Errorf("wrong target: got %q, want %q", cred.TargetPath, "/root/.ssh")
		}
	})

	t.Run("gh credential has correct paths", func(t *testing.T) {
		cred := GetCredentialType("gh")
		if cred == nil {
			t.Fatal("gh credential type not found")
		}
		if cred.SourceSuffix != ".config/gh" {
			t.Errorf("wrong source: got %q, want %q", cred.SourceSuffix, ".config/gh")
		}
		if cred.TargetPath != "/root/.config/gh" {
			t.Errorf("wrong target: got %q, want %q", cred.TargetPath, "/root/.config/gh")
		}
	})

	t.Run("unknown credential returns nil", func(t *testing.T) {
		cred := GetCredentialType("unknown")
		if cred != nil {
			t.Error("expected nil for unknown credential type")
		}
	})
}

func TestMergeCredentials(t *testing.T) {
	t.Run("empty inputs returns empty", func(t *testing.T) {
		result := MergeCredentials(nil, nil, nil)
		if len(result) != 0 {
			t.Errorf("expected empty, got %v", result)
		}
	})

	t.Run("profile credentials returned when no overrides", func(t *testing.T) {
		result := MergeCredentials([]string{"git", "ssh"}, nil, nil)
		if len(result) != 2 {
			t.Fatalf("expected 2 credentials, got %d", len(result))
		}
		if result[0] != "git" || result[1] != "ssh" {
			t.Errorf("unexpected result: %v", result)
		}
	})

	t.Run("add credentials extends list", func(t *testing.T) {
		result := MergeCredentials([]string{"git"}, []string{"ssh"}, nil)
		if len(result) != 2 {
			t.Fatalf("expected 2 credentials, got %d", len(result))
		}
	})

	t.Run("exclude credentials removes from list", func(t *testing.T) {
		result := MergeCredentials([]string{"git", "ssh", "gh"}, nil, []string{"ssh"})
		if len(result) != 2 {
			t.Fatalf("expected 2 credentials, got %d", len(result))
		}
		for _, c := range result {
			if c == "ssh" {
				t.Error("ssh should have been excluded")
			}
		}
	})

	t.Run("add and exclude together", func(t *testing.T) {
		result := MergeCredentials([]string{"git"}, []string{"ssh", "gh"}, []string{"gh"})
		// Start: [git], Add: [ssh, gh], Exclude: [gh] -> [git, ssh]
		if len(result) != 2 {
			t.Fatalf("expected 2 credentials, got %d", len(result))
		}
	})

	t.Run("ignores unknown credential types", func(t *testing.T) {
		result := MergeCredentials([]string{"git", "unknown"}, nil, nil)
		if len(result) != 1 {
			t.Fatalf("expected 1 credential, got %d", len(result))
		}
		if result[0] != "git" {
			t.Errorf("expected git, got %s", result[0])
		}
	})

	t.Run("deduplicates credentials", func(t *testing.T) {
		result := MergeCredentials([]string{"git"}, []string{"git", "ssh"}, nil)
		gitCount := 0
		for _, c := range result {
			if c == "git" {
				gitCount++
			}
		}
		if gitCount != 1 {
			t.Errorf("expected 1 git, got %d", gitCount)
		}
	})
}

func TestResolveCredentialMounts(t *testing.T) {
	t.Run("resolves git credential to mount", func(t *testing.T) {
		homeDir := t.TempDir()
		gitconfig := filepath.Join(homeDir, ".gitconfig")
		if err := os.WriteFile(gitconfig, []byte("[user]\nname = Test"), 0644); err != nil {
			t.Fatalf("failed to create gitconfig: %v", err)
		}

		mounts, warnings := ResolveCredentialMounts([]string{"git"}, homeDir, "")
		if len(warnings) > 0 {
			t.Errorf("unexpected warnings: %v", warnings)
		}
		if len(mounts) != 1 {
			t.Fatalf("expected 1 mount, got %d", len(mounts))
		}
		if mounts[0].Host != gitconfig {
			t.Errorf("wrong host path: got %q, want %q", mounts[0].Host, gitconfig)
		}
		if mounts[0].Container != "/root/.gitconfig" {
			t.Errorf("wrong container path: got %q", mounts[0].Container)
		}
		if !mounts[0].ReadOnly {
			t.Error("mount should be read-only")
		}
	})

	t.Run("missing credential warns and skips", func(t *testing.T) {
		homeDir := t.TempDir()
		// No .gitconfig created

		mounts, warnings := ResolveCredentialMounts([]string{"git"}, homeDir, "")
		if len(mounts) != 0 {
			t.Errorf("expected no mounts, got %d", len(mounts))
		}
		if len(warnings) != 1 {
			t.Fatalf("expected 1 warning, got %d", len(warnings))
		}
	})

	t.Run("resolves multiple credentials", func(t *testing.T) {
		homeDir := t.TempDir()

		// Create git and ssh
		if err := os.WriteFile(filepath.Join(homeDir, ".gitconfig"), []byte(""), 0644); err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(filepath.Join(homeDir, ".ssh"), 0700); err != nil {
			t.Fatal(err)
		}

		mounts, warnings := ResolveCredentialMounts([]string{"git", "ssh"}, homeDir, "")
		if len(warnings) > 0 {
			t.Errorf("unexpected warnings: %v", warnings)
		}
		if len(mounts) != 2 {
			t.Fatalf("expected 2 mounts, got %d", len(mounts))
		}
	})
}
