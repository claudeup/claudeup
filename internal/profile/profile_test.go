// ABOUTME: Tests for Profile struct and Load/Save functionality
// ABOUTME: Validates profile serialization, loading, and listing
package profile

import (
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
