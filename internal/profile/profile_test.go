// ABOUTME: Tests for Profile struct and Load/Save functionality
// ABOUTME: Validates profile serialization, loading, and listing
package profile

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestProfileRoundTrip(t *testing.T) {
	// Create temp directory for test
	tmpDir := t.TempDir()
	profilesDir := filepath.Join(tmpDir, "profiles")

	// Create a profile
	p := &Profile{
		Name:        "test-profile",
		Description: "A test profile",
		MCPServers: []MCPServer{
			{
				Name:    "context7",
				Command: "npx",
				Args:    []string{"-y", "@context7/mcp"},
				Scope:   "user",
			},
		},
		Marketplaces: []Marketplace{
			{Source: "github", Repo: "anthropics/claude-code-plugins"},
		},
		Plugins: []string{"superpowers@superpowers-marketplace"},
	}

	// Save it
	err := Save(profilesDir, p)
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Verify file exists
	profilePath := filepath.Join(profilesDir, "test-profile.json")
	if _, err := os.Stat(profilePath); os.IsNotExist(err) {
		t.Fatal("Profile file was not created")
	}

	// Load it back
	loaded, err := Load(profilesDir, "test-profile")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Verify fields
	if loaded.Name != p.Name {
		t.Errorf("Name mismatch: got %q, want %q", loaded.Name, p.Name)
	}
	if loaded.Description != p.Description {
		t.Errorf("Description mismatch: got %q, want %q", loaded.Description, p.Description)
	}
	if len(loaded.MCPServers) != 1 {
		t.Errorf("MCPServers count mismatch: got %d, want 1", len(loaded.MCPServers))
	}
	if len(loaded.Marketplaces) != 1 {
		t.Errorf("Marketplaces count mismatch: got %d, want 1", len(loaded.Marketplaces))
	}
	if len(loaded.Plugins) != 1 {
		t.Errorf("Plugins count mismatch: got %d, want 1", len(loaded.Plugins))
	}
}

func TestLoadNonexistent(t *testing.T) {
	tmpDir := t.TempDir()
	profilesDir := filepath.Join(tmpDir, "profiles")

	_, err := Load(profilesDir, "does-not-exist")
	if err == nil {
		t.Error("Expected error loading nonexistent profile, got nil")
	}
}

func TestLoadSetsNameFromFilename(t *testing.T) {
	tmpDir := t.TempDir()
	profilesDir := filepath.Join(tmpDir, "profiles")

	// Create directory
	if err := os.MkdirAll(profilesDir, 0755); err != nil {
		t.Fatalf("Failed to create profiles dir: %v", err)
	}

	// Write a profile file without a name field
	profileJSON := `{
		"description": "Profile without name field",
		"plugins": ["plugin1@marketplace", "plugin2@marketplace"]
	}`
	profilePath := filepath.Join(profilesDir, "my-profile.json")
	if err := os.WriteFile(profilePath, []byte(profileJSON), 0644); err != nil {
		t.Fatalf("Failed to write profile file: %v", err)
	}

	// Load the profile
	loaded, err := Load(profilesDir, "my-profile")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Verify name was set from filename
	if loaded.Name != "my-profile" {
		t.Errorf("Name should be set from filename: got %q, want %q", loaded.Name, "my-profile")
	}

	// Verify other fields loaded correctly
	if loaded.Description != "Profile without name field" {
		t.Errorf("Description mismatch: got %q", loaded.Description)
	}
	if len(loaded.Plugins) != 2 {
		t.Errorf("Plugins count mismatch: got %d, want 2", len(loaded.Plugins))
	}
}

func TestLoadPreservesJSONNameOverFilename(t *testing.T) {
	tmpDir := t.TempDir()
	profilesDir := filepath.Join(tmpDir, "profiles")

	if err := os.MkdirAll(profilesDir, 0755); err != nil {
		t.Fatalf("Failed to create profiles dir: %v", err)
	}

	// Write a profile file where JSON name differs from filename
	// This tests that JSON name takes precedence (explicit user intent)
	profileJSON := `{
		"name": "json-specified-name",
		"description": "Profile with explicit name in JSON"
	}`
	profilePath := filepath.Join(profilesDir, "different-filename.json")
	if err := os.WriteFile(profilePath, []byte(profileJSON), 0644); err != nil {
		t.Fatalf("Failed to write profile file: %v", err)
	}

	// Load using the filename
	loaded, err := Load(profilesDir, "different-filename")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// JSON name should take precedence over filename
	if loaded.Name != "json-specified-name" {
		t.Errorf("JSON name should take precedence: got %q, want %q", loaded.Name, "json-specified-name")
	}
}

func TestList(t *testing.T) {
	tmpDir := t.TempDir()
	profilesDir := filepath.Join(tmpDir, "profiles")

	// Create a few profiles
	profiles := []*Profile{
		{Name: "alpha", Description: "First profile"},
		{Name: "beta", Description: "Second profile"},
		{Name: "gamma", Description: "Third profile"},
	}

	for _, p := range profiles {
		if err := Save(profilesDir, p); err != nil {
			t.Fatalf("Failed to save profile %s: %v", p.Name, err)
		}
	}

	// List them
	listed, err := List(profilesDir)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(listed) != 3 {
		t.Errorf("Expected 3 profiles, got %d", len(listed))
	}

	// Verify names (should be sorted)
	expectedNames := []string{"alpha", "beta", "gamma"}
	for i, name := range expectedNames {
		if listed[i].Name != name {
			t.Errorf("Profile %d: got %q, want %q", i, listed[i].Name, name)
		}
	}
}

func TestListEmptyDir(t *testing.T) {
	tmpDir := t.TempDir()
	profilesDir := filepath.Join(tmpDir, "profiles")

	// List from nonexistent directory should return empty, not error
	listed, err := List(profilesDir)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(listed) != 0 {
		t.Errorf("Expected 0 profiles, got %d", len(listed))
	}
}

func TestSecretSourcesInProfile(t *testing.T) {
	tmpDir := t.TempDir()
	profilesDir := filepath.Join(tmpDir, "profiles")

	p := &Profile{
		Name: "with-secrets",
		MCPServers: []MCPServer{
			{
				Name:    "github-mcp",
				Command: "npx",
				Args:    []string{"-y", "@anthropic/github-mcp", "$GITHUB_TOKEN"},
				Secrets: map[string]SecretRef{
					"GITHUB_TOKEN": {
						Description: "GitHub personal access token",
						Sources: []SecretSource{
							{Type: "env", Key: "GITHUB_TOKEN"},
							{Type: "1password", Ref: "op://Private/GitHub/token"},
						},
					},
				},
			},
		},
	}

	if err := Save(profilesDir, p); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	loaded, err := Load(profilesDir, "with-secrets")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if len(loaded.MCPServers) != 1 {
		t.Fatal("Expected 1 MCP server")
	}

	secrets := loaded.MCPServers[0].Secrets
	if len(secrets) != 1 {
		t.Fatal("Expected 1 secret")
	}

	ref, ok := secrets["GITHUB_TOKEN"]
	if !ok {
		t.Fatal("GITHUB_TOKEN secret not found")
	}

	if len(ref.Sources) != 2 {
		t.Errorf("Expected 2 sources, got %d", len(ref.Sources))
	}
}

func TestProfile_Clone(t *testing.T) {
	original := &Profile{
		Name:        "original",
		Description: "Original description",
		MCPServers: []MCPServer{
			{Name: "server1", Command: "cmd1", Args: []string{"arg1"}},
		},
		Marketplaces: []Marketplace{
			{Source: "github", Repo: "org/repo"},
		},
		Plugins: []string{"plugin1", "plugin2"},
	}

	cloned := original.Clone("cloned")

	// Verify name changed
	if cloned.Name != "cloned" {
		t.Errorf("Expected cloned name 'cloned', got %q", cloned.Name)
	}

	// Verify description copied
	if cloned.Description != original.Description {
		t.Errorf("Expected description %q, got %q", original.Description, cloned.Description)
	}

	// Verify deep copy - modifying clone doesn't affect original
	cloned.Plugins[0] = "modified"
	if original.Plugins[0] == "modified" {
		t.Error("Clone should be a deep copy, but modifying clone affected original")
	}

	cloned.MCPServers[0].Name = "modified"
	if original.MCPServers[0].Name == "modified" {
		t.Error("Clone should deep copy MCPServers")
	}
}

func TestProfile_GenerateDescription(t *testing.T) {
	tests := []struct {
		name     string
		profile  *Profile
		expected string
	}{
		{
			name: "all components",
			profile: &Profile{
				Name:         "test",
				Marketplaces: []Marketplace{{Source: "github", Repo: "org/repo"}},
				Plugins:      []string{"plugin1", "plugin2", "plugin3"},
				MCPServers:   []MCPServer{{Name: "server1"}, {Name: "server2"}},
			},
			expected: "3 plugins, 1 marketplace, 2 MCP servers",
		},
		{
			name: "multiple marketplaces",
			profile: &Profile{
				Name:         "test",
				Marketplaces: []Marketplace{{Source: "github"}, {Source: "github"}},
				Plugins:      []string{"plugin1"},
				MCPServers:   []MCPServer{{Name: "server1"}},
			},
			expected: "1 plugin, 2 marketplaces, 1 MCP server",
		},
		{
			name: "only plugins",
			profile: &Profile{
				Name:    "test",
				Plugins: []string{"plugin1", "plugin2"},
			},
			expected: "2 plugins",
		},
		{
			name: "only MCP servers",
			profile: &Profile{
				Name:       "test",
				MCPServers: []MCPServer{{Name: "server1"}},
			},
			expected: "1 MCP server",
		},
		{
			name:     "empty profile",
			profile:  &Profile{Name: "test"},
			expected: "Empty profile",
		},
		{
			name: "multi-scope with user and project plugins",
			profile: &Profile{
				Name: "test",
				PerScope: &PerScopeSettings{
					User: &ScopeSettings{
						Plugins: []string{"a@m", "b@m", "c@m", "d@m", "e@m"},
					},
					Project: &ScopeSettings{
						Plugins: []string{"f@m", "g@m", "h@m"},
					},
				},
				Marketplaces: []Marketplace{
					{Source: "github", Repo: "test/marketplace"},
				},
			},
			expected: "5 user plugins, 3 project plugins, 1 marketplace",
		},
		{
			name: "multi-scope single plugin per scope",
			profile: &Profile{
				Name: "test",
				PerScope: &PerScopeSettings{
					User: &ScopeSettings{
						Plugins: []string{"a@m"},
					},
					Project: &ScopeSettings{
						Plugins: []string{"b@m"},
					},
				},
				Marketplaces: []Marketplace{
					{Source: "github", Repo: "test/m1"},
					{Source: "github", Repo: "test/m2"},
				},
			},
			expected: "1 user plugin, 1 project plugin, 2 marketplaces",
		},
		{
			name: "multi-scope project-only",
			profile: &Profile{
				Name: "test",
				PerScope: &PerScopeSettings{
					Project: &ScopeSettings{
						Plugins: []string{"a@m", "b@m", "c@m"},
					},
				},
				Marketplaces: []Marketplace{
					{Source: "github", Repo: "test/marketplace"},
				},
			},
			expected: "3 project plugins, 1 marketplace",
		},
		{
			name: "multi-scope with all three scopes",
			profile: &Profile{
				Name: "test",
				PerScope: &PerScopeSettings{
					User: &ScopeSettings{
						Plugins: []string{"a@m", "b@m"},
					},
					Project: &ScopeSettings{
						Plugins: []string{"c@m"},
					},
					Local: &ScopeSettings{
						Plugins: []string{"d@m", "e@m", "f@m"},
					},
				},
			},
			expected: "2 user plugins, 1 project plugin, 3 local plugins",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.profile.GenerateDescription()
			if got != tt.expected {
				t.Errorf("GenerateDescription() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestSaveToProject(t *testing.T) {
	tmpDir := t.TempDir()
	projectDir := filepath.Join(tmpDir, "project")

	p := &Profile{Name: "team-profile", Description: "Team shared profile"}

	err := SaveToProject(projectDir, p)
	if err != nil {
		t.Fatalf("SaveToProject failed: %v", err)
	}

	// Verify file exists in correct location
	expectedPath := filepath.Join(projectDir, ".claudeup", "profiles", "team-profile.json")
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Errorf("Profile not saved to expected path: %s", expectedPath)
	}

	// Verify it can be loaded
	loaded, err := Load(filepath.Join(projectDir, ".claudeup", "profiles"), "team-profile")
	if err != nil {
		t.Fatalf("Failed to load saved profile: %v", err)
	}
	if loaded.Description != "Team shared profile" {
		t.Errorf("Description mismatch: got %q", loaded.Description)
	}
}

func TestSave_NestedProfileName(t *testing.T) {
	tmpDir := t.TempDir()
	profilesDir := filepath.Join(tmpDir, "profiles")

	p := &Profile{Name: "projects/claudeup", Description: "Nested profile"}

	err := Save(profilesDir, p)
	if err != nil {
		t.Fatalf("Save with nested name failed: %v", err)
	}

	// Verify file exists in nested location
	expectedPath := filepath.Join(profilesDir, "projects", "claudeup.json")
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Errorf("Nested profile not saved to expected path: %s", expectedPath)
	}

	// Verify it can be loaded by name
	loaded, err := Load(profilesDir, "projects/claudeup")
	if err != nil {
		t.Fatalf("Failed to load nested profile: %v", err)
	}
	if loaded.Description != "Nested profile" {
		t.Errorf("Description mismatch: got %q", loaded.Description)
	}
}

func TestListAll(t *testing.T) {
	tmpDir := t.TempDir()
	userProfilesDir := filepath.Join(tmpDir, "user-profiles")
	projectDir := filepath.Join(tmpDir, "project")
	projectProfilesDir := filepath.Join(projectDir, ".claudeup", "profiles")

	// Create profiles in user directory
	userProfiles := []*Profile{
		{Name: "alpha", Description: "user alpha"},
		{Name: "beta", Description: "user beta"},
	}
	for _, p := range userProfiles {
		if err := Save(userProfilesDir, p); err != nil {
			t.Fatalf("Failed to save user profile %s: %v", p.Name, err)
		}
	}

	// Create profiles in project directory (one with same name as user)
	projectProfiles := []*Profile{
		{Name: "alpha", Description: "project alpha"}, // shadows user alpha
		{Name: "gamma", Description: "project gamma"},
	}
	for _, p := range projectProfiles {
		if err := Save(projectProfilesDir, p); err != nil {
			t.Fatalf("Failed to save project profile %s: %v", p.Name, err)
		}
	}

	// List all profiles
	all, err := ListAll(userProfilesDir, projectDir)
	if err != nil {
		t.Fatalf("ListAll failed: %v", err)
	}

	// Should have 3 profiles: alpha (project), beta (user), gamma (project)
	if len(all) != 3 {
		t.Errorf("Expected 3 profiles, got %d", len(all))
	}

	// Verify sorted order and sources
	expected := []struct {
		name   string
		source string
		desc   string
	}{
		{"alpha", "project", "project alpha"}, // project shadows user
		{"beta", "user", "user beta"},
		{"gamma", "project", "project gamma"},
	}

	for i, exp := range expected {
		if all[i].Name != exp.name {
			t.Errorf("Profile %d: expected name %q, got %q", i, exp.name, all[i].Name)
		}
		if all[i].Source != exp.source {
			t.Errorf("Profile %d (%s): expected source %q, got %q", i, exp.name, exp.source, all[i].Source)
		}
		if all[i].Description != exp.desc {
			t.Errorf("Profile %d (%s): expected desc %q, got %q", i, exp.name, exp.desc, all[i].Description)
		}
	}
}

func TestListAll_EmptyDirectories(t *testing.T) {
	tmpDir := t.TempDir()
	userProfilesDir := filepath.Join(tmpDir, "user-profiles")
	projectDir := filepath.Join(tmpDir, "project")

	// List from nonexistent directories should return empty, not error
	all, err := ListAll(userProfilesDir, projectDir)
	if err != nil {
		t.Fatalf("ListAll failed: %v", err)
	}

	if len(all) != 0 {
		t.Errorf("Expected 0 profiles, got %d", len(all))
	}
}

func TestListAll_OnlyUserProfiles(t *testing.T) {
	tmpDir := t.TempDir()
	userProfilesDir := filepath.Join(tmpDir, "user-profiles")
	projectDir := filepath.Join(tmpDir, "project")

	// Create profile only in user directory
	p := &Profile{Name: "myprofile", Description: "user profile"}
	if err := Save(userProfilesDir, p); err != nil {
		t.Fatalf("Failed to save profile: %v", err)
	}

	all, err := ListAll(userProfilesDir, projectDir)
	if err != nil {
		t.Fatalf("ListAll failed: %v", err)
	}

	if len(all) != 1 {
		t.Errorf("Expected 1 profile, got %d", len(all))
	}
	if all[0].Source != "user" {
		t.Errorf("Expected source 'user', got %q", all[0].Source)
	}
}

func TestListAll_OnlyProjectProfiles(t *testing.T) {
	tmpDir := t.TempDir()
	userProfilesDir := filepath.Join(tmpDir, "user-profiles")
	projectDir := filepath.Join(tmpDir, "project")

	// Create profile only in project directory
	p := &Profile{Name: "myprofile", Description: "project profile"}
	if err := SaveToProject(projectDir, p); err != nil {
		t.Fatalf("Failed to save profile: %v", err)
	}

	all, err := ListAll(userProfilesDir, projectDir)
	if err != nil {
		t.Fatalf("ListAll failed: %v", err)
	}

	if len(all) != 1 {
		t.Errorf("Expected 1 profile, got %d", len(all))
	}
	if all[0].Source != "project" {
		t.Errorf("Expected source 'project', got %q", all[0].Source)
	}
}

func TestProfile_Equal_IdenticalProfiles(t *testing.T) {
	p1 := &Profile{
		Name:        "test",
		Description: "Test profile",
		Marketplaces: []Marketplace{
			{Source: "github", Repo: "anthropics/claude-code"},
		},
		Plugins: []string{"plugin1@marketplace", "plugin2@marketplace"},
	}
	p2 := &Profile{
		Name:        "test",
		Description: "Test profile",
		Marketplaces: []Marketplace{
			{Source: "github", Repo: "anthropics/claude-code"},
		},
		Plugins: []string{"plugin1@marketplace", "plugin2@marketplace"},
	}

	if !p1.Equal(p2) {
		t.Error("Identical profiles should be equal")
	}
}

func TestProfile_Equal_DifferentDescription(t *testing.T) {
	p1 := &Profile{
		Name:        "test",
		Description: "Test profile v1",
	}
	p2 := &Profile{
		Name:        "test",
		Description: "Test profile v2",
	}

	if p1.Equal(p2) {
		t.Error("Profiles with different descriptions should not be equal")
	}
}

func TestProfile_Equal_DifferentPlugins(t *testing.T) {
	p1 := &Profile{
		Name:    "test",
		Plugins: []string{"plugin1@marketplace"},
	}
	p2 := &Profile{
		Name:    "test",
		Plugins: []string{"plugin2@marketplace"},
	}

	if p1.Equal(p2) {
		t.Error("Profiles with different plugins should not be equal")
	}
}

func TestProfile_Equal_DifferentPluginOrder(t *testing.T) {
	// Order should matter for plugins (it affects behavior)
	p1 := &Profile{
		Name:    "test",
		Plugins: []string{"plugin1@marketplace", "plugin2@marketplace"},
	}
	p2 := &Profile{
		Name:    "test",
		Plugins: []string{"plugin2@marketplace", "plugin1@marketplace"},
	}

	if p1.Equal(p2) {
		t.Error("Profiles with different plugin order should not be equal")
	}
}

func TestProfile_Equal_DifferentMarketplaces(t *testing.T) {
	p1 := &Profile{
		Name:         "test",
		Marketplaces: []Marketplace{{Source: "github", Repo: "org1/repo"}},
	}
	p2 := &Profile{
		Name:         "test",
		Marketplaces: []Marketplace{{Source: "github", Repo: "org2/repo"}},
	}

	if p1.Equal(p2) {
		t.Error("Profiles with different marketplaces should not be equal")
	}
}

func TestProfile_Equal_DifferentMCPServers(t *testing.T) {
	p1 := &Profile{
		Name:       "test",
		MCPServers: []MCPServer{{Name: "server1", Command: "cmd1"}},
	}
	p2 := &Profile{
		Name:       "test",
		MCPServers: []MCPServer{{Name: "server2", Command: "cmd2"}},
	}

	if p1.Equal(p2) {
		t.Error("Profiles with different MCP servers should not be equal")
	}
}

func TestProfile_Equal_EmptyVsNilSlices(t *testing.T) {
	// Empty and nil slices should be considered equal
	p1 := &Profile{
		Name:         "test",
		Plugins:      nil,
		Marketplaces: nil,
		MCPServers:   nil,
	}
	p2 := &Profile{
		Name:         "test",
		Plugins:      []string{},
		Marketplaces: []Marketplace{},
		MCPServers:   []MCPServer{},
	}

	if !p1.Equal(p2) {
		t.Error("Profiles with nil vs empty slices should be equal")
	}
}

func TestProfile_Equal_IgnoresName(t *testing.T) {
	// Names are identifiers, not content - two profiles with different names
	// but same content should be considered equal for content comparison
	p1 := &Profile{
		Name:        "profile-a",
		Description: "Same content",
		Plugins:     []string{"plugin1@marketplace"},
	}
	p2 := &Profile{
		Name:        "profile-b",
		Description: "Same content",
		Plugins:     []string{"plugin1@marketplace"},
	}

	if !p1.Equal(p2) {
		t.Error("Profiles with same content but different names should be equal (name is identity, not content)")
	}
}

func TestProfile_Equal_DifferentDetectRules(t *testing.T) {
	p1 := &Profile{
		Name:   "test",
		Detect: DetectRules{Files: []string{"package.json"}},
	}
	p2 := &Profile{
		Name:   "test",
		Detect: DetectRules{Files: []string{"Cargo.toml"}},
	}

	if p1.Equal(p2) {
		t.Error("Profiles with different detect rules should not be equal")
	}
}

func TestProfile_Equal_DifferentMarketplaceOrder(t *testing.T) {
	// Order should matter for marketplaces (it affects plugin resolution)
	p1 := &Profile{
		Name: "test",
		Marketplaces: []Marketplace{
			{Source: "github", Repo: "org1/repo"},
			{Source: "github", Repo: "org2/repo"},
		},
	}
	p2 := &Profile{
		Name: "test",
		Marketplaces: []Marketplace{
			{Source: "github", Repo: "org2/repo"},
			{Source: "github", Repo: "org1/repo"},
		},
	}

	if p1.Equal(p2) {
		t.Error("Profiles with different marketplace order should not be equal")
	}
}

func TestProfile_Equal_PostApplyHook_BothNil(t *testing.T) {
	p1 := &Profile{Name: "test", PostApply: nil}
	p2 := &Profile{Name: "test", PostApply: nil}

	if !p1.Equal(p2) {
		t.Error("Profiles with both nil PostApply hooks should be equal")
	}
}

func TestProfile_Equal_PostApplyHook_OneNil(t *testing.T) {
	p1 := &Profile{Name: "test", PostApply: nil}
	p2 := &Profile{Name: "test", PostApply: &PostApplyHook{Command: "echo test"}}

	if p1.Equal(p2) {
		t.Error("Profiles with one nil PostApply hook should not be equal")
	}
}

func TestProfile_Equal_PostApplyHook_DifferentScript(t *testing.T) {
	p1 := &Profile{Name: "test", PostApply: &PostApplyHook{Script: "setup.sh"}}
	p2 := &Profile{Name: "test", PostApply: &PostApplyHook{Script: "init.sh"}}

	if p1.Equal(p2) {
		t.Error("Profiles with different PostApply scripts should not be equal")
	}
}

func TestProfile_Equal_PostApplyHook_DifferentCommand(t *testing.T) {
	p1 := &Profile{Name: "test", PostApply: &PostApplyHook{Command: "echo hello"}}
	p2 := &Profile{Name: "test", PostApply: &PostApplyHook{Command: "echo world"}}

	if p1.Equal(p2) {
		t.Error("Profiles with different PostApply commands should not be equal")
	}
}

func TestProfile_Equal_PostApplyHook_DifferentCondition(t *testing.T) {
	p1 := &Profile{Name: "test", PostApply: &PostApplyHook{Command: "echo", Condition: "always"}}
	p2 := &Profile{Name: "test", PostApply: &PostApplyHook{Command: "echo", Condition: "first-run"}}

	if p1.Equal(p2) {
		t.Error("Profiles with different PostApply conditions should not be equal")
	}
}

func TestProfile_Equal_PostApplyHook_Identical(t *testing.T) {
	p1 := &Profile{Name: "test", PostApply: &PostApplyHook{
		Script:    "setup.sh",
		Command:   "echo fallback",
		Condition: "first-run",
	}}
	p2 := &Profile{Name: "test", PostApply: &PostApplyHook{
		Script:    "setup.sh",
		Command:   "echo fallback",
		Condition: "first-run",
	}}

	if !p1.Equal(p2) {
		t.Error("Profiles with identical PostApply hooks should be equal")
	}
}

func TestProfile_Equal_NilVsEmptyMaps(t *testing.T) {
	// Nil and empty maps should be considered equal (consistent with slice behavior)
	p1 := &Profile{
		Name:   "test",
		Detect: DetectRules{Contains: nil},
	}
	p2 := &Profile{
		Name:   "test",
		Detect: DetectRules{Contains: map[string]string{}},
	}

	if !p1.Equal(p2) {
		t.Error("Profiles with nil vs empty maps should be equal")
	}
}

func TestSaveProfileTrailingNewline(t *testing.T) {
	tmpDir := t.TempDir()
	profilesDir := filepath.Join(tmpDir, "profiles")

	p := &Profile{
		Name:    "test",
		Plugins: []string{"plugin1@marketplace"},
	}

	if err := Save(profilesDir, p); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(profilesDir, "test.json"))
	if err != nil {
		t.Fatalf("Failed to read saved profile: %v", err)
	}

	if len(data) == 0 {
		t.Fatal("Saved profile is empty")
	}
	if data[len(data)-1] != '\n' {
		t.Error("Saved profile should end with a trailing newline")
	}
}

func TestPreserveFromExisting(t *testing.T) {
	// When overwriting an existing profile, extensions should be preserved
	// from the original, not re-snapshotted.
	existing := &Profile{
		Extensions: &ExtensionSettings{
			Agents: []string{"original-agent"},
		},
	}

	// Fresh snapshot picked up extra stuff from the environment
	fresh := &Profile{
		Extensions: &ExtensionSettings{
			Agents: []string{"original-agent", "extra-agent"},
		},
	}

	fresh.PreserveFrom(existing)

	if len(fresh.Extensions.Agents) != 1 {
		t.Errorf("Expected 1 agent (preserved), got %d", len(fresh.Extensions.Agents))
	}
	if fresh.Extensions.Agents[0] != "original-agent" {
		t.Errorf("Expected original agent, got %q", fresh.Extensions.Agents[0])
	}
}

func TestPreserveFromExistingNilFields(t *testing.T) {
	// When existing profile has no extensions, fresh should keep them nil
	existing := &Profile{}

	fresh := &Profile{
		Extensions: &ExtensionSettings{
			Agents: []string{"extra-agent"},
		},
	}

	fresh.PreserveFrom(existing)

	if fresh.Extensions != nil {
		t.Errorf("Expected nil extensions (existing had none), got %v", fresh.Extensions)
	}
}

func TestProfileWithExtensions(t *testing.T) {
	tmpDir := t.TempDir()
	profilesDir := filepath.Join(tmpDir, "profiles")

	p := &Profile{
		Name:        "gsd-profile",
		Description: "Get Shit Done workflow",
		Extensions: &ExtensionSettings{
			Agents:   []string{"gsd-*"},
			Commands: []string{"gsd/*"},
			Hooks:    []string{"gsd-check-update.js"},
		},
		SettingsHooks: map[string][]HookEntry{
			"SessionStart": {
				{Type: "command", Command: "node ~/.claude/hooks/gsd-check-update.js"},
			},
		},
	}

	// Save
	err := Save(profilesDir, p)
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Load
	loaded, err := Load(profilesDir, "gsd-profile")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Verify Extensions
	if loaded.Extensions == nil {
		t.Fatal("Extensions is nil")
	}
	if len(loaded.Extensions.Agents) != 1 || loaded.Extensions.Agents[0] != "gsd-*" {
		t.Errorf("Extensions.Agents = %v, want [gsd-*]", loaded.Extensions.Agents)
	}
	if len(loaded.Extensions.Commands) != 1 || loaded.Extensions.Commands[0] != "gsd/*" {
		t.Errorf("Extensions.Commands = %v, want [gsd/*]", loaded.Extensions.Commands)
	}
	if len(loaded.Extensions.Hooks) != 1 || loaded.Extensions.Hooks[0] != "gsd-check-update.js" {
		t.Errorf("Extensions.Hooks = %v, want [gsd-check-update.js]", loaded.Extensions.Hooks)
	}

	// Verify SettingsHooks
	if loaded.SettingsHooks == nil {
		t.Fatal("SettingsHooks is nil")
	}
	if len(loaded.SettingsHooks["SessionStart"]) != 1 {
		t.Errorf("SettingsHooks[SessionStart] = %v, want 1 entry", loaded.SettingsHooks["SessionStart"])
	}
	hook := loaded.SettingsHooks["SessionStart"][0]
	if hook.Type != "command" {
		t.Errorf("Hook.Type = %q, want 'command'", hook.Type)
	}
	if hook.Command != "node ~/.claude/hooks/gsd-check-update.js" {
		t.Errorf("Hook.Command = %q, want expected command", hook.Command)
	}
}

// ---------------------------------------------------------------------------
// List with nested profiles tests
// ---------------------------------------------------------------------------

func TestList_IncludesNestedProfiles(t *testing.T) {
	tmpDir := t.TempDir()
	profilesDir := filepath.Join(tmpDir, "profiles")

	// Create flat profile
	if err := Save(profilesDir, &Profile{Name: "alpha", Description: "Alpha profile"}); err != nil {
		t.Fatalf("Failed to save alpha: %v", err)
	}

	// Create nested profile
	backendDir := filepath.Join(profilesDir, "backend")
	if err := os.MkdirAll(backendDir, 0755); err != nil {
		t.Fatalf("Failed to create backend dir: %v", err)
	}
	data, err := json.Marshal(&Profile{Name: "api", Description: "API profile"})
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}
	if err := os.WriteFile(filepath.Join(backendDir, "api.json"), data, 0644); err != nil {
		t.Fatalf("Failed to write nested profile: %v", err)
	}

	listed, err := List(profilesDir)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(listed) != 2 {
		t.Fatalf("Expected 2 profiles, got %d", len(listed))
	}

	// Sorted by name: alpha, api
	if listed[0].Name != "alpha" {
		t.Errorf("Profile 0: got %q, want %q", listed[0].Name, "alpha")
	}
	if listed[1].Name != "api" {
		t.Errorf("Profile 1: got %q, want %q", listed[1].Name, "api")
	}
}

func TestList_ProfileEntryRelPath(t *testing.T) {
	tmpDir := t.TempDir()
	profilesDir := filepath.Join(tmpDir, "profiles")

	// Create flat and nested profiles
	if err := Save(profilesDir, &Profile{Name: "flat"}); err != nil {
		t.Fatalf("Failed to save flat: %v", err)
	}
	backendDir := filepath.Join(profilesDir, "backend")
	if err := os.MkdirAll(backendDir, 0755); err != nil {
		t.Fatalf("Failed to create backend dir: %v", err)
	}
	data, err := json.Marshal(&Profile{Name: "nested"})
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}
	if err := os.WriteFile(filepath.Join(backendDir, "nested.json"), data, 0644); err != nil {
		t.Fatalf("Failed to write: %v", err)
	}

	listed, err := List(profilesDir)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(listed) != 2 {
		t.Fatalf("Expected 2, got %d", len(listed))
	}

	// Find each by name and check RelPath
	relPaths := make(map[string]string)
	for _, e := range listed {
		relPaths[e.Name] = e.RelPath
	}

	if relPaths["flat"] != "flat.json" {
		t.Errorf("flat RelPath = %q, want %q", relPaths["flat"], "flat.json")
	}
	if relPaths["nested"] != "backend/nested.json" {
		t.Errorf("nested RelPath = %q, want %q", relPaths["nested"], "backend/nested.json")
	}
}

func TestList_DuplicateNamesBothReturned(t *testing.T) {
	tmpDir := t.TempDir()
	profilesDir := filepath.Join(tmpDir, "profiles")

	// Create api.json at root
	if err := Save(profilesDir, &Profile{Name: "api", Description: "Root API"}); err != nil {
		t.Fatalf("Failed to save root api: %v", err)
	}

	// Create api.json in backend/
	backendDir := filepath.Join(profilesDir, "backend")
	if err := os.MkdirAll(backendDir, 0755); err != nil {
		t.Fatalf("Failed to create dir: %v", err)
	}
	data, err := json.Marshal(&Profile{Name: "api", Description: "Backend API"})
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}
	if err := os.WriteFile(filepath.Join(backendDir, "api.json"), data, 0644); err != nil {
		t.Fatalf("Failed to write: %v", err)
	}

	listed, err := List(profilesDir)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(listed) != 2 {
		t.Fatalf("Expected 2 profiles (duplicate names included), got %d", len(listed))
	}

	// Both should be named "api" with different RelPaths
	for _, e := range listed {
		if e.Name != "api" {
			t.Errorf("Expected name 'api', got %q", e.Name)
		}
	}
}

func TestList_SortedByNameThenRelPath(t *testing.T) {
	tmpDir := t.TempDir()
	profilesDir := filepath.Join(tmpDir, "profiles")

	// Create profiles: alpha, backend/api, api (root)
	if err := Save(profilesDir, &Profile{Name: "alpha"}); err != nil {
		t.Fatalf("Failed: %v", err)
	}
	if err := Save(profilesDir, &Profile{Name: "api", Description: "Root API"}); err != nil {
		t.Fatalf("Failed: %v", err)
	}
	backendDir := filepath.Join(profilesDir, "backend")
	if err := os.MkdirAll(backendDir, 0755); err != nil {
		t.Fatalf("Failed: %v", err)
	}
	data, err := json.Marshal(&Profile{Name: "api", Description: "Backend API"})
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}
	if err := os.WriteFile(filepath.Join(backendDir, "api.json"), data, 0644); err != nil {
		t.Fatalf("Failed: %v", err)
	}

	listed, err := List(profilesDir)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(listed) != 3 {
		t.Fatalf("Expected 3, got %d", len(listed))
	}

	// alpha, then api (root -- "api.json" < "backend/api.json"), then backend/api
	if listed[0].Name != "alpha" {
		t.Errorf("listed[0].Name = %q, want 'alpha'", listed[0].Name)
	}
	if listed[1].RelPath != "api.json" {
		t.Errorf("listed[1].RelPath = %q, want 'api.json'", listed[1].RelPath)
	}
	if listed[2].RelPath != "backend/api.json" {
		t.Errorf("listed[2].RelPath = %q, want 'backend/api.json'", listed[2].RelPath)
	}
}

func TestList_SkipsEmptySubdirs(t *testing.T) {
	tmpDir := t.TempDir()
	profilesDir := filepath.Join(tmpDir, "profiles")

	if err := Save(profilesDir, &Profile{Name: "solo"}); err != nil {
		t.Fatalf("Failed: %v", err)
	}
	// Create empty subdir
	emptyDir := filepath.Join(profilesDir, "empty")
	if err := os.MkdirAll(emptyDir, 0755); err != nil {
		t.Fatalf("Failed: %v", err)
	}

	listed, err := List(profilesDir)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(listed) != 1 {
		t.Errorf("Expected 1 profile, got %d", len(listed))
	}
}

func TestList_SkipsNonJSONInSubdirs(t *testing.T) {
	tmpDir := t.TempDir()
	profilesDir := filepath.Join(tmpDir, "profiles")

	if err := Save(profilesDir, &Profile{Name: "real"}); err != nil {
		t.Fatalf("Failed: %v", err)
	}
	// Create non-JSON file in subdir
	subDir := filepath.Join(profilesDir, "backend")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("Failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(subDir, "readme.md"), []byte("# Profiles"), 0644); err != nil {
		t.Fatalf("Failed: %v", err)
	}

	listed, err := List(profilesDir)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(listed) != 1 {
		t.Errorf("Expected 1 profile, got %d", len(listed))
	}
}

func TestProfileEntry_DisplayName(t *testing.T) {
	tests := []struct {
		relPath  string
		expected string
	}{
		{"mobile.json", "mobile"},
		{"backend/api.json", "backend/api"},
		{"team/backend/worker.json", "team/backend/worker"},
	}
	for _, tt := range tests {
		e := ProfileEntry{Profile: &Profile{Name: "test"}, RelPath: tt.relPath}
		if got := e.DisplayName(); got != tt.expected {
			t.Errorf("DisplayName(%q) = %q, want %q", tt.relPath, got, tt.expected)
		}
	}
}

// ---------------------------------------------------------------------------
// Load with nested profile support tests
// ---------------------------------------------------------------------------

func TestLoad_FindsNestedProfile(t *testing.T) {
	tmpDir := t.TempDir()
	profilesDir := filepath.Join(tmpDir, "profiles")

	// Create only a nested profile
	backendDir := filepath.Join(profilesDir, "backend")
	if err := os.MkdirAll(backendDir, 0755); err != nil {
		t.Fatalf("Failed to create dir: %v", err)
	}
	data, err := json.Marshal(&Profile{Name: "api", Description: "Backend API"})
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}
	if err := os.WriteFile(filepath.Join(backendDir, "api.json"), data, 0644); err != nil {
		t.Fatalf("Failed to write: %v", err)
	}

	loaded, err := Load(profilesDir, "api")
	if err != nil {
		t.Fatalf("Load should find nested profile, got error: %v", err)
	}

	if loaded.Name != "api" {
		t.Errorf("Name = %q, want %q", loaded.Name, "api")
	}
	if loaded.Description != "Backend API" {
		t.Errorf("Description = %q, want %q", loaded.Description, "Backend API")
	}
}

func TestLoad_AmbiguousNameReturnsError(t *testing.T) {
	tmpDir := t.TempDir()
	profilesDir := filepath.Join(tmpDir, "profiles")

	// Create api.json at root and in backend/
	if err := Save(profilesDir, &Profile{Name: "api", Description: "Root API"}); err != nil {
		t.Fatalf("Failed to save: %v", err)
	}
	backendDir := filepath.Join(profilesDir, "backend")
	if err := os.MkdirAll(backendDir, 0755); err != nil {
		t.Fatalf("Failed to create dir: %v", err)
	}
	data, err := json.Marshal(&Profile{Name: "api", Description: "Backend API"})
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}
	if err := os.WriteFile(filepath.Join(backendDir, "api.json"), data, 0644); err != nil {
		t.Fatalf("Failed to write: %v", err)
	}

	_, loadErr := Load(profilesDir, "api")
	if loadErr == nil {
		t.Fatal("Load should return an error for ambiguous profile name")
	}

	// Error should be the typed AmbiguousProfileError
	var ambigErr *AmbiguousProfileError
	if !errors.As(loadErr, &ambigErr) {
		t.Fatalf("Expected *AmbiguousProfileError, got %T: %v", loadErr, loadErr)
	}
	if ambigErr.Name != "api" {
		t.Errorf("Expected Name 'api', got %q", ambigErr.Name)
	}
	if len(ambigErr.Paths) != 2 {
		t.Errorf("Expected 2 paths, got %d: %v", len(ambigErr.Paths), ambigErr.Paths)
	}
}

func TestLoad_PathReferenceResolvesDirectly(t *testing.T) {
	tmpDir := t.TempDir()
	profilesDir := filepath.Join(tmpDir, "profiles")

	// Create root api.json and backend/api.json
	if err := Save(profilesDir, &Profile{Name: "api", Description: "Root API"}); err != nil {
		t.Fatalf("Failed to save: %v", err)
	}
	backendDir := filepath.Join(profilesDir, "backend")
	if err := os.MkdirAll(backendDir, 0755); err != nil {
		t.Fatalf("Failed to create dir: %v", err)
	}
	data, err := json.Marshal(&Profile{Name: "api", Description: "Backend API"})
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}
	if err := os.WriteFile(filepath.Join(backendDir, "api.json"), data, 0644); err != nil {
		t.Fatalf("Failed to write: %v", err)
	}

	// Using path reference should bypass ambiguity
	loaded, err := Load(profilesDir, "backend/api")
	if err != nil {
		t.Fatalf("Load with path reference should succeed, got: %v", err)
	}

	if loaded.Description != "Backend API" {
		t.Errorf("Description = %q, want %q", loaded.Description, "Backend API")
	}
}

func TestLoad_StillWorksFlatProfiles(t *testing.T) {
	tmpDir := t.TempDir()
	profilesDir := filepath.Join(tmpDir, "profiles")

	// Flat profile at root (existing behavior)
	if err := Save(profilesDir, &Profile{Name: "mobile", Description: "Mobile profile"}); err != nil {
		t.Fatalf("Failed to save: %v", err)
	}

	loaded, err := Load(profilesDir, "mobile")
	if err != nil {
		t.Fatalf("Load should work for flat profiles: %v", err)
	}

	if loaded.Name != "mobile" {
		t.Errorf("Name = %q, want %q", loaded.Name, "mobile")
	}
}

// ---------------------------------------------------------------------------
// FindProfilePaths tests
// ---------------------------------------------------------------------------

func TestFindProfilePaths_FlatProfile(t *testing.T) {
	tmpDir := t.TempDir()
	profilesDir := filepath.Join(tmpDir, "profiles")

	// Create a flat profile: profiles/mobile.json
	if err := os.MkdirAll(profilesDir, 0755); err != nil {
		t.Fatalf("Failed to create profiles dir: %v", err)
	}
	p := Profile{Name: "mobile", Description: "Mobile profile"}
	data, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("Failed to marshal profile: %v", err)
	}
	if err := os.WriteFile(filepath.Join(profilesDir, "mobile.json"), data, 0644); err != nil {
		t.Fatalf("Failed to write profile: %v", err)
	}

	paths, err := FindProfilePaths(profilesDir, "mobile")
	if err != nil {
		t.Fatalf("FindProfilePaths returned error: %v", err)
	}

	if len(paths) != 1 {
		t.Fatalf("Expected 1 path, got %d: %v", len(paths), paths)
	}

	expected := filepath.Join(profilesDir, "mobile.json")
	if paths[0] != expected {
		t.Errorf("Expected path %q, got %q", expected, paths[0])
	}
}

func TestFindProfilePaths_NestedProfile(t *testing.T) {
	tmpDir := t.TempDir()
	profilesDir := filepath.Join(tmpDir, "profiles")

	// Create a nested profile: profiles/backend/api.json
	nestedDir := filepath.Join(profilesDir, "backend")
	if err := os.MkdirAll(nestedDir, 0755); err != nil {
		t.Fatalf("Failed to create nested dir: %v", err)
	}
	p := Profile{Name: "api", Description: "Backend API profile"}
	data, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("Failed to marshal profile: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nestedDir, "api.json"), data, 0644); err != nil {
		t.Fatalf("Failed to write profile: %v", err)
	}

	paths, err := FindProfilePaths(profilesDir, "api")
	if err != nil {
		t.Fatalf("FindProfilePaths returned error: %v", err)
	}

	if len(paths) != 1 {
		t.Fatalf("Expected 1 path, got %d: %v", len(paths), paths)
	}

	expected := filepath.Join(profilesDir, "backend", "api.json")
	if paths[0] != expected {
		t.Errorf("Expected path %q, got %q", expected, paths[0])
	}
}

func TestFindProfilePaths_DeeplyNested(t *testing.T) {
	tmpDir := t.TempDir()
	profilesDir := filepath.Join(tmpDir, "profiles")

	// Create a deeply nested profile: profiles/team/backend/worker.json
	deepDir := filepath.Join(profilesDir, "team", "backend")
	if err := os.MkdirAll(deepDir, 0755); err != nil {
		t.Fatalf("Failed to create deep dir: %v", err)
	}
	p := Profile{Name: "worker", Description: "Worker profile"}
	data, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("Failed to marshal profile: %v", err)
	}
	if err := os.WriteFile(filepath.Join(deepDir, "worker.json"), data, 0644); err != nil {
		t.Fatalf("Failed to write profile: %v", err)
	}

	paths, err := FindProfilePaths(profilesDir, "worker")
	if err != nil {
		t.Fatalf("FindProfilePaths returned error: %v", err)
	}

	if len(paths) != 1 {
		t.Fatalf("Expected 1 path, got %d: %v", len(paths), paths)
	}

	expected := filepath.Join(profilesDir, "team", "backend", "worker.json")
	if paths[0] != expected {
		t.Errorf("Expected path %q, got %q", expected, paths[0])
	}
}

func TestFindProfilePaths_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	profilesDir := filepath.Join(tmpDir, "profiles")

	// Create directory with a different profile
	if err := os.MkdirAll(profilesDir, 0755); err != nil {
		t.Fatalf("Failed to create profiles dir: %v", err)
	}
	p := Profile{Name: "other"}
	data, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("Failed to marshal profile: %v", err)
	}
	if err := os.WriteFile(filepath.Join(profilesDir, "other.json"), data, 0644); err != nil {
		t.Fatalf("Failed to write profile: %v", err)
	}

	paths, err := FindProfilePaths(profilesDir, "nonexistent")
	if err != nil {
		t.Fatalf("FindProfilePaths returned error: %v", err)
	}

	if len(paths) != 0 {
		t.Errorf("Expected empty slice, got %v", paths)
	}
}

func TestFindProfilePaths_MultipleMatches(t *testing.T) {
	tmpDir := t.TempDir()
	profilesDir := filepath.Join(tmpDir, "profiles")

	// Create api.json at root and in backend/ subdir
	if err := os.MkdirAll(profilesDir, 0755); err != nil {
		t.Fatalf("Failed to create profiles dir: %v", err)
	}
	backendDir := filepath.Join(profilesDir, "backend")
	if err := os.MkdirAll(backendDir, 0755); err != nil {
		t.Fatalf("Failed to create backend dir: %v", err)
	}

	rootProfile := Profile{Name: "api", Description: "Root API"}
	data, err := json.Marshal(rootProfile)
	if err != nil {
		t.Fatalf("Failed to marshal profile: %v", err)
	}
	if err := os.WriteFile(filepath.Join(profilesDir, "api.json"), data, 0644); err != nil {
		t.Fatalf("Failed to write root profile: %v", err)
	}

	nestedProfile := Profile{Name: "api", Description: "Backend API"}
	data, err = json.Marshal(nestedProfile)
	if err != nil {
		t.Fatalf("Failed to marshal profile: %v", err)
	}
	if err := os.WriteFile(filepath.Join(backendDir, "api.json"), data, 0644); err != nil {
		t.Fatalf("Failed to write nested profile: %v", err)
	}

	paths, err := FindProfilePaths(profilesDir, "api")
	if err != nil {
		t.Fatalf("FindProfilePaths returned error: %v", err)
	}

	if len(paths) != 2 {
		t.Fatalf("Expected 2 paths, got %d: %v", len(paths), paths)
	}

	// Verify both expected paths are present
	expectedRoot := filepath.Join(profilesDir, "api.json")
	expectedNested := filepath.Join(profilesDir, "backend", "api.json")
	foundRoot, foundNested := false, false
	for _, p := range paths {
		if p == expectedRoot {
			foundRoot = true
		}
		if p == expectedNested {
			foundNested = true
		}
	}
	if !foundRoot {
		t.Errorf("Missing root path %q in results %v", expectedRoot, paths)
	}
	if !foundNested {
		t.Errorf("Missing nested path %q in results %v", expectedNested, paths)
	}
}

func TestFindProfilePaths_MissingDir(t *testing.T) {
	tmpDir := t.TempDir()
	profilesDir := filepath.Join(tmpDir, "does-not-exist")

	paths, err := FindProfilePaths(profilesDir, "anything")
	if err != nil {
		t.Fatalf("FindProfilePaths should not return error for missing dir, got: %v", err)
	}

	if len(paths) != 0 {
		t.Errorf("Expected empty slice for missing dir, got %v", paths)
	}
}

func TestFindProfilePaths_IgnoresNonJSON(t *testing.T) {
	tmpDir := t.TempDir()
	profilesDir := filepath.Join(tmpDir, "profiles")

	if err := os.MkdirAll(profilesDir, 0755); err != nil {
		t.Fatalf("Failed to create profiles dir: %v", err)
	}

	// Create non-JSON files with the same stem
	if err := os.WriteFile(filepath.Join(profilesDir, "api.txt"), []byte("not json"), 0644); err != nil {
		t.Fatalf("Failed to write txt file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(profilesDir, "api.yaml"), []byte("not json"), 0644); err != nil {
		t.Fatalf("Failed to write yaml file: %v", err)
	}

	paths, err := FindProfilePaths(profilesDir, "api")
	if err != nil {
		t.Fatalf("FindProfilePaths returned error: %v", err)
	}

	if len(paths) != 0 {
		t.Errorf("Expected empty slice (non-JSON files should be ignored), got %v", paths)
	}
}

func TestFindProfilePaths_PathReference(t *testing.T) {
	tmpDir := t.TempDir()
	profilesDir := filepath.Join(tmpDir, "profiles")

	// Create profiles/backend/api.json
	backendDir := filepath.Join(profilesDir, "backend")
	if err := os.MkdirAll(backendDir, 0755); err != nil {
		t.Fatalf("Failed to create backend dir: %v", err)
	}
	p := Profile{Name: "api", Description: "Backend API"}
	data, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("Failed to marshal profile: %v", err)
	}
	if err := os.WriteFile(filepath.Join(backendDir, "api.json"), data, 0644); err != nil {
		t.Fatalf("Failed to write profile: %v", err)
	}

	// Use path reference: "backend/api" should resolve directly
	paths, err := FindProfilePaths(profilesDir, "backend/api")
	if err != nil {
		t.Fatalf("FindProfilePaths returned error: %v", err)
	}

	if len(paths) != 1 {
		t.Fatalf("Expected 1 path for path reference, got %d: %v", len(paths), paths)
	}

	expected := filepath.Join(profilesDir, "backend", "api.json")
	if paths[0] != expected {
		t.Errorf("Expected path %q, got %q", expected, paths[0])
	}
}

func TestFindProfilePaths_PathTraversalBlocked(t *testing.T) {
	tmpDir := t.TempDir()
	profilesDir := filepath.Join(tmpDir, "profiles")

	if err := os.MkdirAll(profilesDir, 0755); err != nil {
		t.Fatalf("Failed to create profiles dir: %v", err)
	}

	// Create a file outside profilesDir to ensure it can't be reached
	outsidePath := filepath.Join(tmpDir, "secret.json")
	if err := os.WriteFile(outsidePath, []byte(`{"name":"secret"}`), 0644); err != nil {
		t.Fatalf("Failed to write outside file: %v", err)
	}

	// Attempt traversal with "../"
	_, err := FindProfilePaths(profilesDir, "../secret")
	if err == nil {
		t.Fatal("Expected error for path traversal attempt, got nil")
	}

	if !strings.Contains(err.Error(), "escapes profiles directory") {
		t.Errorf("Expected 'escapes profiles directory' error, got: %q", err.Error())
	}
}

func TestFindProfilePaths_PathReferenceNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	profilesDir := filepath.Join(tmpDir, "profiles")

	// Create the backend dir but not the target profile
	backendDir := filepath.Join(profilesDir, "backend")
	if err := os.MkdirAll(backendDir, 0755); err != nil {
		t.Fatalf("Failed to create backend dir: %v", err)
	}

	paths, err := FindProfilePaths(profilesDir, "backend/missing")
	if err != nil {
		t.Fatalf("FindProfilePaths returned error: %v", err)
	}

	if len(paths) != 0 {
		t.Errorf("Expected empty slice for missing path reference, got %v", paths)
	}
}

// ---------------------------------------------------------------------------
// LoadFromPath tests
// ---------------------------------------------------------------------------

func TestLoadFromPath_LoadsProfile(t *testing.T) {
	tmpDir := t.TempDir()

	p := Profile{
		Name:        "myprofile",
		Description: "A test profile",
		Plugins:     []string{"plugin1@marketplace"},
		MCPServers: []MCPServer{
			{Name: "server1", Command: "cmd1", Args: []string{"arg1"}},
		},
	}
	data, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("Failed to marshal profile: %v", err)
	}

	profilePath := filepath.Join(tmpDir, "myprofile.json")
	if err := os.WriteFile(profilePath, data, 0644); err != nil {
		t.Fatalf("Failed to write profile: %v", err)
	}

	loaded, err := LoadFromPath(profilePath)
	if err != nil {
		t.Fatalf("LoadFromPath returned error: %v", err)
	}

	if loaded.Name != "myprofile" {
		t.Errorf("Name = %q, want %q", loaded.Name, "myprofile")
	}
	if loaded.Description != "A test profile" {
		t.Errorf("Description = %q, want %q", loaded.Description, "A test profile")
	}
	if len(loaded.Plugins) != 1 || loaded.Plugins[0] != "plugin1@marketplace" {
		t.Errorf("Plugins = %v, want [plugin1@marketplace]", loaded.Plugins)
	}
	if len(loaded.MCPServers) != 1 || loaded.MCPServers[0].Name != "server1" {
		t.Errorf("MCPServers = %v, want [{server1 ...}]", loaded.MCPServers)
	}
}

func TestLoadFromPath_SetsNameFromFilename(t *testing.T) {
	tmpDir := t.TempDir()

	// Profile JSON without a name field
	profileJSON := `{
		"description": "No name in JSON",
		"plugins": ["plugin1@marketplace"]
	}`
	profilePath := filepath.Join(tmpDir, "derived-name.json")
	if err := os.WriteFile(profilePath, []byte(profileJSON), 0644); err != nil {
		t.Fatalf("Failed to write profile: %v", err)
	}

	loaded, err := LoadFromPath(profilePath)
	if err != nil {
		t.Fatalf("LoadFromPath returned error: %v", err)
	}

	if loaded.Name != "derived-name" {
		t.Errorf("Name should be derived from filename: got %q, want %q", loaded.Name, "derived-name")
	}
}

func TestLoadFromPath_PreservesJSONName(t *testing.T) {
	tmpDir := t.TempDir()

	// Profile JSON with an explicit name field that differs from filename
	profileJSON := `{
		"name": "explicit-name",
		"description": "Name in JSON"
	}`
	profilePath := filepath.Join(tmpDir, "different-filename.json")
	if err := os.WriteFile(profilePath, []byte(profileJSON), 0644); err != nil {
		t.Fatalf("Failed to write profile: %v", err)
	}

	loaded, err := LoadFromPath(profilePath)
	if err != nil {
		t.Fatalf("LoadFromPath returned error: %v", err)
	}

	if loaded.Name != "explicit-name" {
		t.Errorf("JSON name should be preserved: got %q, want %q", loaded.Name, "explicit-name")
	}
}

func TestLoadFromPath_NonexistentPath(t *testing.T) {
	_, err := LoadFromPath("/tmp/absolutely-does-not-exist-profile.json")
	if err == nil {
		t.Error("Expected error for nonexistent path, got nil")
	}
}

func TestLoadFromPath_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()

	profilePath := filepath.Join(tmpDir, "broken.json")
	if err := os.WriteFile(profilePath, []byte("{this is not valid json}"), 0644); err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}

	_, err := LoadFromPath(profilePath)
	if err == nil {
		t.Error("Expected error for invalid JSON, got nil")
	}
}

func TestClone_IncludesDeepCopy(t *testing.T) {
	original := &Profile{
		Name:     "original",
		Includes: []string{"a", "b", "c"},
	}

	clone := original.Clone("clone")

	// Verify includes were copied
	if len(clone.Includes) != 3 {
		t.Fatalf("clone includes: got %d, want 3", len(clone.Includes))
	}

	// Modify original includes -- clone should not be affected
	original.Includes[0] = "modified"
	if clone.Includes[0] != "a" {
		t.Errorf("clone includes[0] changed after modifying original: got %q, want %q", clone.Includes[0], "a")
	}
}

func TestEqual_IncludesComparison(t *testing.T) {
	a := &Profile{
		Name:     "a",
		Includes: []string{"x", "y"},
	}
	b := &Profile{
		Name:     "b",
		Includes: []string{"x", "y"},
	}
	if !a.Equal(b) {
		t.Error("profiles with same includes should be equal")
	}

	c := &Profile{
		Name:     "c",
		Includes: []string{"x", "z"},
	}
	if a.Equal(c) {
		t.Error("profiles with different includes should not be equal")
	}

	d := &Profile{
		Name:     "d",
		Includes: nil,
	}
	e := &Profile{
		Name:     "e",
		Includes: []string{},
	}
	if !d.Equal(e) {
		t.Error("nil and empty includes should be equal")
	}
}

func TestPreserveFrom_DoesNotCopyIncludes(t *testing.T) {
	existing := &Profile{
		Includes: []string{"saved-include"},
		Extensions: &ExtensionSettings{
			Agents: []string{"saved-agent"},
		},
	}

	p := &Profile{
		Name:    "new-save",
		Plugins: []string{"plugin-a"},
	}

	p.PreserveFrom(existing)

	// Includes should NOT be preserved -- re-saving a snapshot over a stack
	// would produce an invalid profile with both includes and config fields
	if len(p.Includes) != 0 {
		t.Errorf("includes should not be preserved: got %v", p.Includes)
	}
	if p.Extensions == nil || len(p.Extensions.Agents) != 1 {
		t.Error("extensions not preserved")
	}
}

func TestGenerateDescription_Stack(t *testing.T) {
	tests := []struct {
		name     string
		profile  *Profile
		expected string
	}{
		{
			"single include",
			&Profile{Includes: []string{"a"}},
			"stack: 1 include",
		},
		{
			"multiple includes",
			&Profile{Includes: []string{"a", "b", "c"}},
			"stack: 3 includes",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.profile.GenerateDescription()
			if got != tt.expected {
				t.Errorf("GenerateDescription(): got %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestProfileRoundTrip_WithIncludes(t *testing.T) {
	tmpDir := t.TempDir()
	profilesDir := filepath.Join(tmpDir, "profiles")

	p := &Profile{
		Name:        "my-stack",
		Description: "A composable stack",
		Includes:    []string{"base", "languages/go", "testing"},
	}

	if err := Save(profilesDir, p); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	loaded, err := Load(profilesDir, "my-stack")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if len(loaded.Includes) != 3 {
		t.Fatalf("includes: got %d, want 3", len(loaded.Includes))
	}
	if loaded.Includes[0] != "base" || loaded.Includes[1] != "languages/go" || loaded.Includes[2] != "testing" {
		t.Errorf("includes: got %v, want [base languages/go testing]", loaded.Includes)
	}
	if !loaded.IsStack() {
		t.Error("loaded profile should be a stack")
	}
}

func TestLoadProfileWithLegacyLocalItemsField(t *testing.T) {
	// Profiles saved before the rename used "localItems" in JSON.
	// Verify they load correctly into the Extensions field.
	jsonData := []byte(`{
		"name": "legacy",
		"localItems": {
			"agents": ["planner.md"],
			"rules": ["coding.md"]
		}
	}`)

	var p Profile
	if err := json.Unmarshal(jsonData, &p); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if p.Extensions == nil {
		t.Fatal("Extensions should be populated from legacy localItems field")
	}
	if len(p.Extensions.Agents) != 1 || p.Extensions.Agents[0] != "planner.md" {
		t.Errorf("Agents = %v, want [planner.md]", p.Extensions.Agents)
	}
	if len(p.Extensions.Rules) != 1 || p.Extensions.Rules[0] != "coding.md" {
		t.Errorf("Rules = %v, want [coding.md]", p.Extensions.Rules)
	}
}

func TestLoadProfileWithLegacyLocalItemsInPerScope(t *testing.T) {
	// Per-scope settings also used "localItems" before the rename.
	jsonData := []byte(`{
		"name": "legacy-scoped",
		"perScope": {
			"user": {
				"localItems": {
					"agents": ["user-agent.md"]
				}
			},
			"project": {
				"localItems": {
					"rules": ["project-rule.md"]
				}
			}
		}
	}`)

	var p Profile
	if err := json.Unmarshal(jsonData, &p); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if p.PerScope == nil {
		t.Fatal("PerScope should be populated")
	}
	if p.PerScope.User == nil || p.PerScope.User.Extensions == nil {
		t.Fatal("User scope Extensions should be populated from legacy localItems")
	}
	if p.PerScope.User.Extensions.Agents[0] != "user-agent.md" {
		t.Errorf("User agents = %v, want [user-agent.md]", p.PerScope.User.Extensions.Agents)
	}
	if p.PerScope.Project == nil || p.PerScope.Project.Extensions == nil {
		t.Fatal("Project scope Extensions should be populated from legacy localItems")
	}
	if p.PerScope.Project.Extensions.Rules[0] != "project-rule.md" {
		t.Errorf("Project rules = %v, want [project-rule.md]", p.PerScope.Project.Extensions.Rules)
	}
}

func TestNewFieldTakesPrecedenceOverLegacy(t *testing.T) {
	// If both "extensions" and "localItems" are present, "extensions" wins.
	jsonData := []byte(`{
		"name": "both",
		"extensions": {
			"agents": ["new-agent.md"]
		},
		"localItems": {
			"agents": ["old-agent.md"]
		}
	}`)

	var p Profile
	if err := json.Unmarshal(jsonData, &p); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if p.Extensions == nil {
		t.Fatal("Extensions should be populated")
	}
	if p.Extensions.Agents[0] != "new-agent.md" {
		t.Errorf("Agents = %v, want [new-agent.md] (new field should take precedence)", p.Extensions.Agents)
	}
}

func TestValidateMarketplaceRefs(t *testing.T) {
	t.Run("flat profile with matching marketplace passes", func(t *testing.T) {
		p := &Profile{
			Plugins: []string{"my-tool@claude-code-plugins"},
			Marketplaces: []Marketplace{
				{Source: "github", Repo: "anthropics/claude-code-plugins"},
			},
		}
		err := p.ValidateMarketplaceRefs(nil)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("flat profile with unresolvable plugin fails", func(t *testing.T) {
		p := &Profile{
			Plugins: []string{"my-tool@nonexistent"},
			Marketplaces: []Marketplace{
				{Source: "github", Repo: "anthropics/claude-code-plugins"},
			},
		}
		err := p.ValidateMarketplaceRefs(nil)
		if err == nil {
			t.Error("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "my-tool@nonexistent") {
			t.Errorf("error should mention the plugin, got: %v", err)
		}
	})

	t.Run("scoped profile aggregates plugins from all scopes", func(t *testing.T) {
		p := &Profile{
			Marketplaces: []Marketplace{
				{Source: "github", Repo: "anthropics/claude-code-plugins"},
			},
			PerScope: &PerScopeSettings{
				User: &ScopeSettings{
					Plugins: []string{"user-tool@claude-code-plugins"},
				},
				Project: &ScopeSettings{
					Plugins: []string{"proj-tool@nonexistent"},
				},
			},
		}
		err := p.ValidateMarketplaceRefs(nil)
		if err == nil {
			t.Error("expected error for scoped profile with unresolvable project plugin")
		}
		if !strings.Contains(err.Error(), "proj-tool@nonexistent") {
			t.Errorf("error should mention the unresolvable plugin, got: %v", err)
		}
	})

	t.Run("scoped profile with all matching passes", func(t *testing.T) {
		p := &Profile{
			Marketplaces: []Marketplace{
				{Source: "github", Repo: "anthropics/claude-code-plugins"},
			},
			PerScope: &PerScopeSettings{
				User: &ScopeSettings{
					Plugins: []string{"user-tool@claude-code-plugins"},
				},
				Local: &ScopeSettings{
					Plugins: []string{"local-tool@claude-code-plugins"},
				},
			},
		}
		err := p.ValidateMarketplaceRefs(nil)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("registry key resolves plugin ref", func(t *testing.T) {
		p := &Profile{
			Plugins: []string{"my-tool@custom-marketplace"},
		}
		err := p.ValidateMarketplaceRefs([]string{"custom-marketplace"})
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("nil profile returns no error", func(t *testing.T) {
		var p *Profile
		err := p.ValidateMarketplaceRefs(nil)
		if err != nil {
			t.Errorf("unexpected error for nil profile: %v", err)
		}
	})
}
