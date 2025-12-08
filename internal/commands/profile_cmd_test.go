// ABOUTME: Tests for profile command functions
// ABOUTME: Validates profile loading fallback behavior
package commands

import (
	"os"
	"path/filepath"
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
