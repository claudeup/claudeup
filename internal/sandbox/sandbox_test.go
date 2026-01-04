// ABOUTME: Unit tests for sandbox package.
// ABOUTME: Tests state management and mount parsing functionality.
package sandbox

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStateDir(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("creates directory for profile", func(t *testing.T) {
		dir, err := StateDir(tmpDir, "test-profile")
		if err != nil {
			t.Fatalf("StateDir failed: %v", err)
		}

		expected := filepath.Join(tmpDir, "sandboxes", "test-profile")
		if dir != expected {
			t.Errorf("got %q, want %q", dir, expected)
		}

		if _, err := os.Stat(dir); os.IsNotExist(err) {
			t.Error("directory was not created")
		}
	})

	t.Run("returns error for empty profile", func(t *testing.T) {
		_, err := StateDir(tmpDir, "")
		if err == nil {
			t.Error("expected error for empty profile")
		}
	})

	t.Run("creates directory with restricted permissions", func(t *testing.T) {
		dir, err := StateDir(tmpDir, "perm-test")
		if err != nil {
			t.Fatalf("StateDir failed: %v", err)
		}

		info, err := os.Stat(dir)
		if err != nil {
			t.Fatalf("failed to stat directory: %v", err)
		}

		// Directory should be owner-only (0700)
		if info.Mode().Perm() != 0700 {
			t.Errorf("wrong permissions: got %o, want 0700", info.Mode().Perm())
		}
	})

	t.Run("handles base directory with spaces", func(t *testing.T) {
		baseDirWithSpaces := filepath.Join(t.TempDir(), "path with spaces", "claudeup")
		if err := os.MkdirAll(baseDirWithSpaces, 0755); err != nil {
			t.Fatalf("failed to create base dir: %v", err)
		}

		dir, err := StateDir(baseDirWithSpaces, "test-profile")
		if err != nil {
			t.Fatalf("StateDir failed: %v", err)
		}

		expected := filepath.Join(baseDirWithSpaces, "sandboxes", "test-profile")
		if dir != expected {
			t.Errorf("got %q, want %q", dir, expected)
		}

		if _, err := os.Stat(dir); os.IsNotExist(err) {
			t.Error("directory was not created")
		}
	})
}

func TestCleanState(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("removes existing state", func(t *testing.T) {
		// Create state directory
		dir, err := StateDir(tmpDir, "clean-test")
		if err != nil {
			t.Fatalf("StateDir failed: %v", err)
		}

		// Create a file in it
		testFile := filepath.Join(dir, "test.txt")
		if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}

		// Clean it
		if err := CleanState(tmpDir, "clean-test"); err != nil {
			t.Fatalf("CleanState failed: %v", err)
		}

		// Verify it's gone
		if _, err := os.Stat(dir); !os.IsNotExist(err) {
			t.Error("directory still exists after clean")
		}
	})

	t.Run("returns error for empty profile", func(t *testing.T) {
		err := CleanState(tmpDir, "")
		if err == nil {
			t.Error("expected error for empty profile")
		}
	})

	t.Run("succeeds for non-existent profile", func(t *testing.T) {
		err := CleanState(tmpDir, "non-existent")
		if err != nil {
			t.Errorf("CleanState failed for non-existent: %v", err)
		}
	})
}

func TestParseMount(t *testing.T) {
	home, _ := os.UserHomeDir()

	tests := []struct {
		name    string
		input   string
		want    Mount
		wantErr bool
	}{
		{
			name:  "simple mount",
			input: "/host:/container",
			want:  Mount{Host: "/host", Container: "/container", ReadOnly: false},
		},
		{
			name:  "readonly mount",
			input: "/host:/container:ro",
			want:  Mount{Host: "/host", Container: "/container", ReadOnly: true},
		},
		{
			name:  "home directory expansion",
			input: "~/data:/data",
			want:  Mount{Host: home + "/data", Container: "/data", ReadOnly: false},
		},
		{
			name:  "home directory only",
			input: "~:/home",
			want:  Mount{Host: home, Container: "/home", ReadOnly: false},
		},
		{
			name:  "path with spaces",
			input: "/path/with spaces/data:/container/data",
			want:  Mount{Host: "/path/with spaces/data", Container: "/container/data", ReadOnly: false},
		},
		{
			name:  "path with special characters",
			input: "/path/with-dashes_and_underscores:/container",
			want:  Mount{Host: "/path/with-dashes_and_underscores", Container: "/container", ReadOnly: false},
		},
		{
			name:    "invalid format - single path",
			input:   "/only-one-path",
			wantErr: true,
		},
		{
			name:    "invalid format - too many parts",
			input:   "/a:/b:/c:/d",
			wantErr: true,
		},
		{
			name:    "invalid option",
			input:   "/host:/container:rw",
			wantErr: true,
		},
		{
			name:    "empty host path",
			input:   ":/container",
			wantErr: true,
		},
		{
			name:    "empty container path",
			input:   "/host:",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseMount(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestExpandHome(t *testing.T) {
	home, _ := os.UserHomeDir()

	tests := []struct {
		input string
		want  string
	}{
		{"~/foo", home + "/foo"},
		{"~", home},
		{"/absolute/path", "/absolute/path"},
		{"relative/path", "relative/path"},
		{"~user/foo", "~user/foo"}, // Only ~ or ~/ is expanded, not ~user
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := expandHome(tt.input)
			if got != tt.want {
				t.Errorf("expandHome(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestDefaultImage(t *testing.T) {
	image := DefaultImage()
	if image == "" {
		t.Error("DefaultImage returned empty string")
	}
	if image != "ghcr.io/claudeup/claudeup-sandbox:latest" {
		t.Errorf("unexpected default image: %s", image)
	}
}

func TestCopyAuthFile(t *testing.T) {
	t.Run("copies auth file to sandbox state directory", func(t *testing.T) {
		// Setup temp directories
		homeDir := t.TempDir()
		claudeUpDir := t.TempDir()
		profile := "test-profile"

		// Create source .claude.json
		sourceFile := filepath.Join(homeDir, ".claude.json")
		authContent := []byte(`{"oauthAccount": {"email": "test@example.com"}}`)
		if err := os.WriteFile(sourceFile, authContent, 0600); err != nil {
			t.Fatalf("failed to create source auth file: %v", err)
		}

		// Copy auth file
		if err := CopyAuthFile(homeDir, claudeUpDir, profile); err != nil {
			t.Fatalf("CopyAuthFile failed: %v", err)
		}

		// Verify destination file exists and has correct content
		stateDir := filepath.Join(claudeUpDir, "sandboxes", profile)
		destFile := filepath.Join(stateDir, ".claude.json")

		destContent, err := os.ReadFile(destFile)
		if err != nil {
			t.Fatalf("failed to read destination file: %v", err)
		}

		if string(destContent) != string(authContent) {
			t.Errorf("content mismatch: got %q, want %q", destContent, authContent)
		}

		// Verify file permissions
		info, err := os.Stat(destFile)
		if err != nil {
			t.Fatalf("failed to stat destination file: %v", err)
		}
		if info.Mode().Perm() != 0600 {
			t.Errorf("wrong permissions: got %o, want 0600", info.Mode().Perm())
		}
	})

	t.Run("returns error when source file doesn't exist", func(t *testing.T) {
		homeDir := t.TempDir()
		claudeUpDir := t.TempDir()

		err := CopyAuthFile(homeDir, claudeUpDir, "test-profile")
		if err == nil {
			t.Error("expected error when source file doesn't exist")
		}
	})

	t.Run("returns error for empty profile", func(t *testing.T) {
		homeDir := t.TempDir()
		claudeUpDir := t.TempDir()

		err := CopyAuthFile(homeDir, claudeUpDir, "")
		if err == nil {
			t.Error("expected error for empty profile")
		}
	})

	t.Run("overwrites existing auth file", func(t *testing.T) {
		homeDir := t.TempDir()
		claudeUpDir := t.TempDir()
		profile := "test-profile"

		// Create source file
		sourceFile := filepath.Join(homeDir, ".claude.json")
		newContent := []byte(`{"oauthAccount": {"email": "new@example.com"}}`)
		if err := os.WriteFile(sourceFile, newContent, 0600); err != nil {
			t.Fatalf("failed to create source file: %v", err)
		}

		// Create state directory with existing auth file
		stateDir := filepath.Join(claudeUpDir, "sandboxes", profile)
		if err := os.MkdirAll(stateDir, 0755); err != nil {
			t.Fatalf("failed to create state dir: %v", err)
		}
		destFile := filepath.Join(stateDir, ".claude.json")
		oldContent := []byte(`{"oauthAccount": {"email": "old@example.com"}}`)
		if err := os.WriteFile(destFile, oldContent, 0600); err != nil {
			t.Fatalf("failed to create existing auth file: %v", err)
		}

		// Copy should overwrite
		if err := CopyAuthFile(homeDir, claudeUpDir, profile); err != nil {
			t.Fatalf("CopyAuthFile failed: %v", err)
		}

		// Verify new content
		destContent, err := os.ReadFile(destFile)
		if err != nil {
			t.Fatalf("failed to read destination file: %v", err)
		}

		if string(destContent) != string(newContent) {
			t.Errorf("content not overwritten: got %q, want %q", destContent, newContent)
		}
	})
}
