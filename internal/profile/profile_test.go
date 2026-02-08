// ABOUTME: Tests for Profile struct and Load/Save functionality
// ABOUTME: Validates profile serialization, loading, and listing
package profile

import (
	"encoding/json"
	"os"
	"path/filepath"
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
			expected: "1 marketplace, 3 plugins, 2 MCP servers",
		},
		{
			name: "multiple marketplaces",
			profile: &Profile{
				Name:         "test",
				Marketplaces: []Marketplace{{Source: "github"}, {Source: "github"}},
				Plugins:      []string{"plugin1"},
				MCPServers:   []MCPServer{{Name: "server1"}},
			},
			expected: "2 marketplaces, 1 plugin, 1 MCP server",
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
	// When overwriting an existing profile, localItems should be preserved
	// from the original, not re-snapshotted.
	existing := &Profile{
		LocalItems: &LocalItemSettings{
			Agents: []string{"original-agent"},
		},
	}

	// Fresh snapshot picked up extra stuff from the environment
	fresh := &Profile{
		LocalItems: &LocalItemSettings{
			Agents: []string{"original-agent", "extra-agent"},
		},
	}

	fresh.PreserveFrom(existing)

	if len(fresh.LocalItems.Agents) != 1 {
		t.Errorf("Expected 1 agent (preserved), got %d", len(fresh.LocalItems.Agents))
	}
	if fresh.LocalItems.Agents[0] != "original-agent" {
		t.Errorf("Expected original agent, got %q", fresh.LocalItems.Agents[0])
	}
}

func TestPreserveFromExistingNilFields(t *testing.T) {
	// When existing profile has no localItems, fresh should keep them nil
	existing := &Profile{}

	fresh := &Profile{
		LocalItems: &LocalItemSettings{
			Agents: []string{"extra-agent"},
		},
	}

	fresh.PreserveFrom(existing)

	if fresh.LocalItems != nil {
		t.Errorf("Expected nil localItems (existing had none), got %v", fresh.LocalItems)
	}
}

func TestProfileWithLocalItems(t *testing.T) {
	tmpDir := t.TempDir()
	profilesDir := filepath.Join(tmpDir, "profiles")

	p := &Profile{
		Name:        "gsd-profile",
		Description: "Get Shit Done workflow",
		LocalItems: &LocalItemSettings{
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

	// Verify LocalItems
	if loaded.LocalItems == nil {
		t.Fatal("LocalItems is nil")
	}
	if len(loaded.LocalItems.Agents) != 1 || loaded.LocalItems.Agents[0] != "gsd-*" {
		t.Errorf("LocalItems.Agents = %v, want [gsd-*]", loaded.LocalItems.Agents)
	}
	if len(loaded.LocalItems.Commands) != 1 || loaded.LocalItems.Commands[0] != "gsd/*" {
		t.Errorf("LocalItems.Commands = %v, want [gsd/*]", loaded.LocalItems.Commands)
	}
	if len(loaded.LocalItems.Hooks) != 1 || loaded.LocalItems.Hooks[0] != "gsd-check-update.js" {
		t.Errorf("LocalItems.Hooks = %v, want [gsd-check-update.js]", loaded.LocalItems.Hooks)
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
