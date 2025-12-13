// ABOUTME: Tests for profile command functions
// ABOUTME: Validates profile loading fallback behavior
package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/claudeup/claudeup/internal/profile"
)

func TestLoadProfileWithFallback_LoadsFromDiskFirst(t *testing.T) {
	tmpDir := t.TempDir()
	profilesDir := filepath.Join(tmpDir, "profiles")
	os.MkdirAll(profilesDir, 0755)

	// Create a custom profile on disk with same name as embedded
	customProfile := &profile.Profile{
		Name:        "default",
		Description: "Custom default profile",
	}
	if err := profile.Save(profilesDir, customProfile); err != nil {
		t.Fatalf("Failed to save custom profile: %v", err)
	}

	// Load should return the disk version, not embedded
	p, err := loadProfileWithFallback(profilesDir, "default")
	if err != nil {
		t.Fatalf("loadProfileWithFallback failed: %v", err)
	}

	if p.Description != "Custom default profile" {
		t.Errorf("Expected custom profile from disk, got description: %q", p.Description)
	}
}

func TestLoadProfileWithFallback_FallsBackToEmbedded(t *testing.T) {
	tmpDir := t.TempDir()
	profilesDir := filepath.Join(tmpDir, "profiles")
	// Don't create any profiles on disk

	// Load should fall back to embedded "frontend" profile
	p, err := loadProfileWithFallback(profilesDir, "frontend")
	if err != nil {
		t.Fatalf("loadProfileWithFallback failed: %v", err)
	}

	if p.Name != "frontend" {
		t.Errorf("Expected embedded frontend profile, got: %q", p.Name)
	}

	if len(p.Plugins) == 0 {
		t.Error("Expected embedded profile to have plugins")
	}
}

func TestLoadProfileWithFallback_ReturnsErrorIfNeitherExists(t *testing.T) {
	tmpDir := t.TempDir()
	profilesDir := filepath.Join(tmpDir, "profiles")

	// Try to load a profile that doesn't exist anywhere
	_, err := loadProfileWithFallback(profilesDir, "nonexistent-profile")
	if err == nil {
		t.Error("Expected error for nonexistent profile, got nil")
	}
}

func TestLoadProfileWithFallback_PrefersDiskOverEmbedded(t *testing.T) {
	tmpDir := t.TempDir()
	profilesDir := filepath.Join(tmpDir, "profiles")
	os.MkdirAll(profilesDir, 0755)

	// Create a modified "frontend" profile on disk
	customFrontend := &profile.Profile{
		Name:        "frontend",
		Description: "My customized frontend profile",
		Plugins:     []string{"custom-plugin@marketplace"},
	}
	if err := profile.Save(profilesDir, customFrontend); err != nil {
		t.Fatalf("Failed to save custom frontend profile: %v", err)
	}

	// Load should return disk version with our customizations
	p, err := loadProfileWithFallback(profilesDir, "frontend")
	if err != nil {
		t.Fatalf("loadProfileWithFallback failed: %v", err)
	}

	if p.Description != "My customized frontend profile" {
		t.Errorf("Expected customized profile, got description: %q", p.Description)
	}

	if len(p.Plugins) != 1 || p.Plugins[0] != "custom-plugin@marketplace" {
		t.Errorf("Expected custom plugins, got: %v", p.Plugins)
	}
}

func TestPromptProfileSelection_ReturnsErrorOnEmptyInput(t *testing.T) {
	tmpDir := t.TempDir()
	profilesDir := filepath.Join(tmpDir, "profiles")
	os.MkdirAll(profilesDir, 0755)

	// Create a profile so selection menu has something to show
	testProfile := &profile.Profile{
		Name:        "test-profile",
		Description: "Test profile",
	}
	if err := profile.Save(profilesDir, testProfile); err != nil {
		t.Fatalf("Failed to save test profile: %v", err)
	}

	// Create a pipe to simulate stdin with empty input (just newline)
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("Failed to create pipe: %v", err)
	}

	// Write empty input (just Enter)
	w.WriteString("\n")
	w.Close()

	// Swap stdin
	oldStdin := os.Stdin
	os.Stdin = r
	defer func() { os.Stdin = oldStdin }()

	// Call promptProfileSelection - should return error for empty input
	_, err = promptProfileSelection(profilesDir, "new-profile")
	if err == nil {
		t.Error("Expected error for empty input, got nil")
	}

	expectedErr := "no selection made"
	if err.Error() != expectedErr {
		t.Errorf("Expected error %q, got %q", expectedErr, err.Error())
	}
}

func TestPromptProfileSelection_ReturnsErrorOnInvalidNumber(t *testing.T) {
	tmpDir := t.TempDir()
	profilesDir := filepath.Join(tmpDir, "profiles")
	os.MkdirAll(profilesDir, 0755)

	// Create a single profile
	testProfile := &profile.Profile{
		Name:        "test-profile",
		Description: "Test profile",
	}
	if err := profile.Save(profilesDir, testProfile); err != nil {
		t.Fatalf("Failed to save test profile: %v", err)
	}

	tests := []struct {
		name        string
		input       string
		errContains string
	}{
		{"zero", "0\n", "invalid selection: 0"},
		{"negative", "-1\n", "invalid selection: -1"},
		{"too large", "999\n", "invalid selection: 999"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, w, err := os.Pipe()
			if err != nil {
				t.Fatalf("Failed to create pipe: %v", err)
			}

			w.WriteString(tt.input)
			w.Close()

			oldStdin := os.Stdin
			os.Stdin = r
			defer func() { os.Stdin = oldStdin }()

			_, err = promptProfileSelection(profilesDir, "new-profile")
			if err == nil {
				t.Errorf("Expected error for input %q, got nil", tt.input)
				return
			}

			if !strings.Contains(err.Error(), tt.errContains) {
				t.Errorf("Expected error containing %q, got %q", tt.errContains, err.Error())
			}
		})
	}
}

func TestPromptProfileSelection_ReturnsErrorOnInvalidName(t *testing.T) {
	tmpDir := t.TempDir()
	profilesDir := filepath.Join(tmpDir, "profiles")
	os.MkdirAll(profilesDir, 0755)

	// Create a profile
	testProfile := &profile.Profile{
		Name:        "test-profile",
		Description: "Test profile",
	}
	if err := profile.Save(profilesDir, testProfile); err != nil {
		t.Fatalf("Failed to save test profile: %v", err)
	}

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("Failed to create pipe: %v", err)
	}

	w.WriteString("nonexistent-profile\n")
	w.Close()

	oldStdin := os.Stdin
	os.Stdin = r
	defer func() { os.Stdin = oldStdin }()

	_, err = promptProfileSelection(profilesDir, "new-profile")
	if err == nil {
		t.Error("Expected error for nonexistent profile name, got nil")
		return
	}

	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("Expected error containing 'not found', got %q", err.Error())
	}
}

func TestPromptProfileSelection_ReturnsErrorOnIOError(t *testing.T) {
	tmpDir := t.TempDir()
	profilesDir := filepath.Join(tmpDir, "profiles")
	os.MkdirAll(profilesDir, 0755)

	// Create a profile
	testProfile := &profile.Profile{
		Name:        "test-profile",
		Description: "Test profile",
	}
	if err := profile.Save(profilesDir, testProfile); err != nil {
		t.Fatalf("Failed to save test profile: %v", err)
	}

	// Create a pipe and close the write end immediately to simulate EOF
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("Failed to create pipe: %v", err)
	}
	w.Close() // Close immediately - no data written

	oldStdin := os.Stdin
	os.Stdin = r
	defer func() { os.Stdin = oldStdin }()

	_, err = promptProfileSelection(profilesDir, "new-profile")
	if err == nil {
		t.Error("Expected error for EOF, got nil")
		return
	}

	if !strings.Contains(err.Error(), "failed to read input") {
		t.Errorf("Expected error containing 'failed to read input', got %q", err.Error())
	}
}
