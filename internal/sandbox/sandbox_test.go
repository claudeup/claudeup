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
