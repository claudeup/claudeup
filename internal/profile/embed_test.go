// ABOUTME: Tests for embedded profile functionality
// ABOUTME: Validates that default profiles are properly embedded and extracted
package profile

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetEmbeddedProfile(t *testing.T) {
	p, err := GetEmbeddedProfile("default")
	if err != nil {
		t.Fatalf("Failed to get embedded default profile: %v", err)
	}

	if p.Name != "default" {
		t.Errorf("Expected name 'default', got %q", p.Name)
	}

	if p.Description == "" {
		t.Error("Expected description to be set")
	}
}

func TestGetEmbeddedProfileNotFound(t *testing.T) {
	_, err := GetEmbeddedProfile("nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent profile")
	}
}

func TestGetEmbeddedFrontendProfile(t *testing.T) {
	p, err := GetEmbeddedProfile("frontend")
	if err != nil {
		t.Fatalf("Failed to get embedded frontend profile: %v", err)
	}

	if p.Name != "frontend" {
		t.Errorf("Expected name 'frontend', got %q", p.Name)
	}

	if p.Description == "" {
		t.Error("Expected description to be set")
	}

	// Verify marketplaces
	if len(p.Marketplaces) != 3 {
		t.Errorf("Expected 3 marketplaces, got %d", len(p.Marketplaces))
	}

	// Verify plugins
	expectedPlugins := []string{
		"frontend-design@claude-plugins-official",
		"nextjs-vercel-pro@claude-code-templates",
		"superpowers@superpowers-marketplace",
		"episodic-memory@superpowers-marketplace",
		"commit-commands@claude-plugins-official",
	}
	if len(p.Plugins) != len(expectedPlugins) {
		t.Errorf("Expected %d plugins, got %d", len(expectedPlugins), len(p.Plugins))
	}

	// Verify detect rules
	if len(p.Detect.Files) == 0 {
		t.Error("Expected detect.files to be populated")
	}
}

func TestGetEmbeddedFrontendFullProfile(t *testing.T) {
	p, err := GetEmbeddedProfile("frontend-full")
	if err != nil {
		t.Fatalf("Failed to get embedded frontend-full profile: %v", err)
	}

	if p.Name != "frontend-full" {
		t.Errorf("Expected name 'frontend-full', got %q", p.Name)
	}

	// Should have more plugins than frontend (lean)
	if len(p.Plugins) < 7 {
		t.Errorf("Expected at least 7 plugins for frontend-full, got %d", len(p.Plugins))
	}

	// Should include testing-suite
	hasTestingSuite := false
	for _, plugin := range p.Plugins {
		if plugin == "testing-suite@claude-code-templates" {
			hasTestingSuite = true
			break
		}
	}
	if !hasTestingSuite {
		t.Error("Expected frontend-full to include testing-suite@claude-code-templates")
	}
}

func TestAllEmbeddedProfilesAreValid(t *testing.T) {
	profiles, err := ListEmbeddedProfiles()
	if err != nil {
		t.Fatalf("Failed to list embedded profiles: %v", err)
	}

	if len(profiles) == 0 {
		t.Error("Expected at least one embedded profile")
	}

	names := make(map[string]bool)
	for _, p := range profiles {
		if p.Name == "" {
			t.Error("Found profile with empty name")
		}
		if names[p.Name] {
			t.Errorf("Duplicate profile name: %s", p.Name)
		}
		names[p.Name] = true

		// Verify required fields
		if p.Description == "" {
			t.Errorf("Profile %q has empty description", p.Name)
		}
	}

	// Verify expected profiles exist
	expectedProfiles := []string{"default", "frontend", "frontend-full"}
	for _, expected := range expectedProfiles {
		if !names[expected] {
			t.Errorf("Expected embedded profile %q not found", expected)
		}
	}
}

func TestEnsureDefaultProfiles(t *testing.T) {
	tmpDir := t.TempDir()
	profilesDir := filepath.Join(tmpDir, "profiles")

	err := EnsureDefaultProfiles(profilesDir)
	if err != nil {
		t.Fatalf("EnsureDefaultProfiles failed: %v", err)
	}

	// Check that default.json was created
	defaultPath := filepath.Join(profilesDir, "default.json")
	if _, err := os.Stat(defaultPath); os.IsNotExist(err) {
		t.Error("default.json was not created")
	}

	// Load and verify
	p, err := Load(profilesDir, "default")
	if err != nil {
		t.Fatalf("Failed to load extracted profile: %v", err)
	}

	if p.Name != "default" {
		t.Errorf("Expected name 'default', got %q", p.Name)
	}
}

func TestEnsureDefaultProfilesDoesNotOverwrite(t *testing.T) {
	tmpDir := t.TempDir()
	profilesDir := filepath.Join(tmpDir, "profiles")
	os.MkdirAll(profilesDir, 0755)

	// Create a custom default.json
	customContent := `{"name": "default", "description": "Custom description"}`
	defaultPath := filepath.Join(profilesDir, "default.json")
	os.WriteFile(defaultPath, []byte(customContent), 0644)

	// Run ensure - should not overwrite
	err := EnsureDefaultProfiles(profilesDir)
	if err != nil {
		t.Fatalf("EnsureDefaultProfiles failed: %v", err)
	}

	// Verify custom content is preserved
	p, err := Load(profilesDir, "default")
	if err != nil {
		t.Fatalf("Failed to load profile: %v", err)
	}

	if p.Description != "Custom description" {
		t.Errorf("Profile was overwritten, got description: %q", p.Description)
	}
}
