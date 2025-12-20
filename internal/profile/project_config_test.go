package profile

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSaveAndLoadProjectConfig(t *testing.T) {
	// Create temp directory
	tempDir, err := os.MkdirTemp("", "claudeup-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create config
	cfg := &ProjectConfig{
		Profile:       "frontend",
		ProfileSource: "embedded",
		Marketplaces: []Marketplace{
			{Source: "github", Repo: "test/plugins"},
		},
		Plugins: []string{"plugin-a", "plugin-b"},
	}

	// Save
	if err := SaveProjectConfig(tempDir, cfg); err != nil {
		t.Fatalf("SaveProjectConfig failed: %v", err)
	}

	// Verify file exists
	path := filepath.Join(tempDir, ProjectConfigFile)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatalf("config file was not created")
	}

	// Load
	loaded, err := LoadProjectConfig(tempDir)
	if err != nil {
		t.Fatalf("LoadProjectConfig failed: %v", err)
	}

	// Verify fields
	if loaded.Version != "1" {
		t.Errorf("Version = %q, want %q", loaded.Version, "1")
	}
	if loaded.Profile != "frontend" {
		t.Errorf("Profile = %q, want %q", loaded.Profile, "frontend")
	}
	if loaded.ProfileSource != "embedded" {
		t.Errorf("ProfileSource = %q, want %q", loaded.ProfileSource, "embedded")
	}
	if len(loaded.Marketplaces) != 1 {
		t.Errorf("len(Marketplaces) = %d, want 1", len(loaded.Marketplaces))
	}
	if len(loaded.Plugins) != 2 {
		t.Errorf("len(Plugins) = %d, want 2", len(loaded.Plugins))
	}
	if loaded.AppliedAt.IsZero() {
		t.Error("AppliedAt should be set")
	}
}

func TestProjectConfigExists(t *testing.T) {
	// Create temp directory
	tempDir, err := os.MkdirTemp("", "claudeup-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Should not exist initially
	if ProjectConfigExists(tempDir) {
		t.Error("ProjectConfigExists should return false for empty directory")
	}

	// Create file
	cfg := &ProjectConfig{Profile: "test"}
	if err := SaveProjectConfig(tempDir, cfg); err != nil {
		t.Fatalf("SaveProjectConfig failed: %v", err)
	}

	// Should exist now
	if !ProjectConfigExists(tempDir) {
		t.Error("ProjectConfigExists should return true after saving")
	}
}

func TestLoadProjectConfig_InvalidJSON(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "claudeup-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Write invalid JSON
	path := filepath.Join(tempDir, ProjectConfigFile)
	if err := os.WriteFile(path, []byte("not json"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	_, err = LoadProjectConfig(tempDir)
	if err == nil {
		t.Error("LoadProjectConfig should fail for invalid JSON")
	}
}

func TestLoadProjectConfig_NotFound(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "claudeup-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	_, err = LoadProjectConfig(tempDir)
	if err == nil {
		t.Error("LoadProjectConfig should fail when file doesn't exist")
	}
}

func TestNewProjectConfig(t *testing.T) {
	p := &Profile{
		Name: "test-profile",
		Marketplaces: []Marketplace{
			{Source: "github", Repo: "test/repo"},
		},
		Plugins: []string{"plugin-a"},
	}

	cfg := NewProjectConfig(p)

	if cfg.Profile != "test-profile" {
		t.Errorf("Profile = %q, want %q", cfg.Profile, "test-profile")
	}
	if cfg.ProfileSource != "custom" {
		t.Errorf("ProfileSource = %q, want %q", cfg.ProfileSource, "custom")
	}
	if len(cfg.Marketplaces) != 1 {
		t.Errorf("len(Marketplaces) = %d, want 1", len(cfg.Marketplaces))
	}
	if len(cfg.Plugins) != 1 {
		t.Errorf("len(Plugins) = %d, want 1", len(cfg.Plugins))
	}
}

func TestProjectConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     ProjectConfig
		wantErr bool
	}{
		{
			name:    "valid config",
			cfg:     ProjectConfig{Profile: "test"},
			wantErr: false,
		},
		{
			name:    "missing profile",
			cfg:     ProjectConfig{},
			wantErr: true,
		},
		{
			name:    "empty profile",
			cfg:     ProjectConfig{Profile: ""},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLoadProjectConfig_ValidationError(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "claudeup-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Write invalid config (missing profile)
	path := filepath.Join(tempDir, ProjectConfigFile)
	data := []byte(`{"version": "1", "plugins": ["plugin-a"]}`)
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	_, err = LoadProjectConfig(tempDir)
	if err == nil {
		t.Error("LoadProjectConfig should fail for config missing profile")
	}
	if !strings.Contains(err.Error(), "profile") {
		t.Errorf("error should mention missing profile: %v", err)
	}
}
