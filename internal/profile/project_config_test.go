package profile

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/claudeup/claudeup/internal/claude"
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
		Profile: "frontend",
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
	if cfg.AppliedAt.IsZero() {
		t.Error("AppliedAt should be set")
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

func TestDetectConfigDrift(t *testing.T) {
	t.Run("no drift when configs don't exist", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "claudeup-test-*")
		if err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tempDir)

		profilesDir := filepath.Join(tempDir, "profiles")

		// Create mock plugin registry with installed plugins
		mockRegistry := &MockPluginRegistry{
			plugins: map[string]bool{
				"plugin-a@marketplace": true,
				"plugin-b@marketplace": true,
			},
		}

		drift, err := DetectConfigDrift(profilesDir, tempDir, tempDir, mockRegistry)
		if err != nil {
			t.Fatalf("DetectConfigDrift failed: %v", err)
		}

		if len(drift) != 0 {
			t.Errorf("expected no drift, got %d drifted plugins", len(drift))
		}
	})

	t.Run("detects drift from project config", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "claudeup-test-*")
		if err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tempDir)

		// Create profiles directory and test profile
		profilesDir := filepath.Join(tempDir, "profiles")
		if err := os.MkdirAll(profilesDir, 0755); err != nil {
			t.Fatalf("failed to create profiles dir: %v", err)
		}

		testProfile := &Profile{
			Name: "test-profile",
			Plugins: []string{
				"plugin-a@marketplace",  // installed
				"plugin-b@marketplace",  // NOT installed (drift)
				"plugin-c@marketplace",  // NOT installed (drift)
			},
		}
		if err := Save(profilesDir, testProfile); err != nil {
			t.Fatalf("failed to save test profile: %v", err)
		}

		// Create project config referencing the profile
		projectCfg := &ProjectConfig{
			Profile: "test-profile",
		}
		if err := SaveProjectConfig(tempDir, projectCfg); err != nil {
			t.Fatalf("SaveProjectConfig failed: %v", err)
		}

		// Create mock registry with only plugin-a installed
		mockRegistry := &MockPluginRegistry{
			plugins: map[string]bool{
				"plugin-a@marketplace": true,
			},
		}

		drift, err := DetectConfigDrift(profilesDir, tempDir, tempDir, mockRegistry)
		if err != nil {
			t.Fatalf("DetectConfigDrift failed: %v", err)
		}

		if len(drift) != 2 {
			t.Errorf("expected 2 drifted plugins, got %d", len(drift))
		}

		// Check that drifted plugins are from project scope
		for _, d := range drift {
			if d.Scope != ScopeProject {
				t.Errorf("expected drift from project scope, got %s", d.Scope)
			}
			if d.PluginName != "plugin-b@marketplace" && d.PluginName != "plugin-c@marketplace" {
				t.Errorf("unexpected drifted plugin: %s", d.PluginName)
			}
		}
	})

	t.Run("detects drift from local settings", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "claudeup-test-*")
		if err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tempDir)
		// Create .claude directory
		claudeDir := filepath.Join(tempDir, ".claude")
		if err := os.MkdirAll(claudeDir, 0755); err != nil {
			t.Fatalf("failed to create .claude dir: %v", err)
		}

		// Create local settings with enabled plugins
		localSettings := &claude.Settings{
			EnabledPlugins: map[string]bool{
				"local-plugin-a@marketplace": true, // NOT installed (drift)
			},
		}
		settingsPath := filepath.Join(claudeDir, "settings.local.json")
		data, err := json.MarshalIndent(localSettings, "", "  ")
		if err != nil {
			t.Fatalf("failed to marshal settings: %v", err)
		}
		if err := os.WriteFile(settingsPath, data, 0644); err != nil {
			t.Fatalf("failed to write settings: %v", err)
		}

		// Create mock registry with no plugins installed
		mockRegistry := &MockPluginRegistry{
			plugins: map[string]bool{},
		}

		profilesDir := filepath.Join(tempDir, "profiles")
		drift, err := DetectConfigDrift(profilesDir, tempDir, tempDir, mockRegistry)
		if err != nil {
			t.Fatalf("DetectConfigDrift failed: %v", err)
		}

		if len(drift) != 1 {
			t.Errorf("expected 1 drifted plugin, got %d", len(drift))
		}

		if drift[0].Scope != ScopeLocal {
			t.Errorf("expected drift from local scope, got %s", drift[0].Scope)
		}
		if drift[0].PluginName != "local-plugin-a@marketplace" {
			t.Errorf("expected local-plugin-a@marketplace, got %s", drift[0].PluginName)
		}
	})

	t.Run("detects drift from both scopes", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "claudeup-test-*")
		if err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tempDir)

		// Create profiles directory and test profile
		profilesDir := filepath.Join(tempDir, "profiles")
		if err := os.MkdirAll(profilesDir, 0755); err != nil {
			t.Fatalf("failed to create profiles dir: %v", err)
		}

		testProfile := &Profile{
			Name: "test-profile",
			Plugins: []string{
				"project-plugin@marketplace",  // NOT installed (drift)
			},
		}
		if err := Save(profilesDir, testProfile); err != nil {
			t.Fatalf("failed to save test profile: %v", err)
		}

		// Create project config
		projectCfg := &ProjectConfig{
			Profile: "test-profile",
		}
		if err := SaveProjectConfig(tempDir, projectCfg); err != nil {
			t.Fatalf("SaveProjectConfig failed: %v", err)
		}

		// Create .claude directory
		claudeDir := filepath.Join(tempDir, ".claude")
		if err := os.MkdirAll(claudeDir, 0755); err != nil {
			t.Fatalf("failed to create .claude dir: %v", err)
		}

		// Create local settings with enabled plugins
		localSettings := &claude.Settings{
			EnabledPlugins: map[string]bool{
				"local-plugin@marketplace": true, // NOT installed (drift)
			},
		}
		settingsPath := filepath.Join(claudeDir, "settings.local.json")
		data, err := json.MarshalIndent(localSettings, "", "  ")
		if err != nil {
			t.Fatalf("failed to marshal settings: %v", err)
		}
		if err := os.WriteFile(settingsPath, data, 0644); err != nil {
			t.Fatalf("failed to write settings: %v", err)
		}

		// Create mock registry with no plugins installed
		mockRegistry := &MockPluginRegistry{
			plugins: map[string]bool{},
		}

		drift, err := DetectConfigDrift(profilesDir, tempDir, tempDir, mockRegistry)
		if err != nil {
			t.Fatalf("DetectConfigDrift failed: %v", err)
		}

		if len(drift) != 2 {
			t.Errorf("expected 2 drifted plugins, got %d", len(drift))
		}

		// Check we have one from each scope
		scopeCounts := make(map[Scope]int)
		for _, d := range drift {
			scopeCounts[d.Scope]++
		}

		if scopeCounts[ScopeProject] != 1 {
			t.Errorf("expected 1 drift from project scope, got %d", scopeCounts[ScopeProject])
		}
		if scopeCounts[ScopeLocal] != 1 {
			t.Errorf("expected 1 drift from local scope, got %d", scopeCounts[ScopeLocal])
		}
	})

	t.Run("no drift when all plugins are installed", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "claudeup-test-*")
		if err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tempDir)

		// Create profiles directory and test profile
		profilesDir := filepath.Join(tempDir, "profiles")
		if err := os.MkdirAll(profilesDir, 0755); err != nil {
			t.Fatalf("failed to create profiles dir: %v", err)
		}

		testProfile := &Profile{
			Name: "test-profile",
			Plugins: []string{
				"plugin-a@marketplace",
				"plugin-b@marketplace",
			},
		}
		if err := Save(profilesDir, testProfile); err != nil {
			t.Fatalf("failed to save test profile: %v", err)
		}

		// Create project config
		projectCfg := &ProjectConfig{
			Profile: "test-profile",
		}
		if err := SaveProjectConfig(tempDir, projectCfg); err != nil {
			t.Fatalf("SaveProjectConfig failed: %v", err)
		}

		// Create mock registry with all plugins installed
		mockRegistry := &MockPluginRegistry{
			plugins: map[string]bool{
				"plugin-a@marketplace": true,
				"plugin-b@marketplace": true,
			},
		}

		drift, err := DetectConfigDrift(profilesDir, tempDir, tempDir, mockRegistry)
		if err != nil {
			t.Fatalf("DetectConfigDrift failed: %v", err)
		}

		if len(drift) != 0 {
			t.Errorf("expected no drift, got %d drifted plugins", len(drift))
		}
	})
}

// MockPluginRegistry implements PluginChecker interface for testing
type MockPluginRegistry struct {
	plugins map[string]bool
}

func (m *MockPluginRegistry) IsPluginInstalled(name string) bool {
	return m.plugins[name]
}

func TestDetectProfileFromProject(t *testing.T) {
	t.Run("returns profile name when .claudeup.json exists", func(t *testing.T) {
		tempDir := t.TempDir()
		cfg := &ProjectConfig{Profile: "my-profile"}
		if err := SaveProjectConfig(tempDir, cfg); err != nil {
			t.Fatalf("failed to save config: %v", err)
		}

		name, err := DetectProfileFromProject(tempDir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if name != "my-profile" {
			t.Errorf("expected 'my-profile', got '%s'", name)
		}
	})

	t.Run("returns empty string when no .claudeup.json exists", func(t *testing.T) {
		tempDir := t.TempDir()

		name, err := DetectProfileFromProject(tempDir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if name != "" {
			t.Errorf("expected empty string, got '%s'", name)
		}
	})

	t.Run("returns error for malformed .claudeup.json", func(t *testing.T) {
		tempDir := t.TempDir()
		path := filepath.Join(tempDir, ".claudeup.json")
		if err := os.WriteFile(path, []byte("not json"), 0644); err != nil {
			t.Fatalf("failed to write file: %v", err)
		}

		_, err := DetectProfileFromProject(tempDir)
		if err == nil {
			t.Error("expected error for malformed JSON")
		}
		if !strings.Contains(err.Error(), "invalid") {
			t.Errorf("expected error to contain 'invalid', got: %v", err)
		}
	})
}
