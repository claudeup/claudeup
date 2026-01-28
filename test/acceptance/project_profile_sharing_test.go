// ABOUTME: Acceptance tests for project-local profile sharing
// ABOUTME: Tests the complete Alice/Bob team workflow

package acceptance

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/claudeup/claudeup/v3/internal/profile"
)

// Section 1: Directory structure and resolution order
func TestProjectProfileDirectoryStructure(t *testing.T) {
	env := SetupAcceptanceTestEnv(t)
	defer env.Cleanup()

	projectDir := env.ProjectDir()

	// Create a profile and save to project scope
	p := &profile.Profile{
		Name:        "backend-go",
		Description: "Go backend development",
		Plugins:     []string{"backend-development@claude-code-workflows"},
	}

	// Save to project
	if err := profile.SaveToProject(projectDir, p); err != nil {
		t.Fatalf("Failed to save profile to project: %v", err)
	}

	// Verify directory structure
	expectedDir := filepath.Join(projectDir, ".claudeup", "profiles")
	if _, err := os.Stat(expectedDir); os.IsNotExist(err) {
		t.Errorf("Expected .claudeup/profiles/ directory to exist at %s", expectedDir)
	}

	expectedFile := filepath.Join(expectedDir, "backend-go.json")
	if _, err := os.Stat(expectedFile); os.IsNotExist(err) {
		t.Errorf("Expected backend-go.json to exist at %s", expectedFile)
	}
}

func TestProfileResolutionOrder_ProjectFirst(t *testing.T) {
	env := SetupAcceptanceTestEnv(t)
	defer env.Cleanup()

	projectDir := env.ProjectDir()

	// Create same profile in both locations with different content
	userProfile := &profile.Profile{
		Name:        "shared-name",
		Description: "User version - should be overridden",
		Plugins:     []string{"user-plugin@marketplace"},
	}
	projectProfile := &profile.Profile{
		Name:        "shared-name",
		Description: "Project version - should win",
		Plugins:     []string{"project-plugin@marketplace"},
	}

	// Save to user profiles
	if err := profile.Save(env.ProfilesDir, userProfile); err != nil {
		t.Fatalf("Failed to save user profile: %v", err)
	}

	// Save to project profiles
	if err := profile.SaveToProject(projectDir, projectProfile); err != nil {
		t.Fatalf("Failed to save project profile: %v", err)
	}

	// Load with fallback - should get project version
	loaded, source, err := profile.LoadWithFallback(env.ProfilesDir, projectDir, "shared-name")
	if err != nil {
		t.Fatalf("LoadWithFallback failed: %v", err)
	}

	if source != "project" {
		t.Errorf("Expected source 'project', got %q", source)
	}

	if loaded.Description != "Project version - should win" {
		t.Errorf("Expected project description, got %q", loaded.Description)
	}

	if len(loaded.Plugins) != 1 || loaded.Plugins[0] != "project-plugin@marketplace" {
		t.Errorf("Expected project plugins, got %v", loaded.Plugins)
	}
}

func TestProfileResolutionOrder_FallbackToUser(t *testing.T) {
	env := SetupAcceptanceTestEnv(t)
	defer env.Cleanup()

	projectDir := env.ProjectDir()

	// Create profile only in user location
	userProfile := &profile.Profile{
		Name:        "user-only",
		Description: "Only exists in user profiles",
		Plugins:     []string{"my-plugin@marketplace"},
	}

	if err := profile.Save(env.ProfilesDir, userProfile); err != nil {
		t.Fatalf("Failed to save user profile: %v", err)
	}

	// Load with fallback - should fall back to user
	loaded, source, err := profile.LoadWithFallback(env.ProfilesDir, projectDir, "user-only")
	if err != nil {
		t.Fatalf("LoadWithFallback failed: %v", err)
	}

	if source != "user" {
		t.Errorf("Expected source 'user', got %q", source)
	}

	if loaded.Description != "Only exists in user profiles" {
		t.Errorf("Expected user description, got %q", loaded.Description)
	}
}

// Section 2: Saving profiles with --scope project
func TestSaveToProject_CreatesCorrectFiles(t *testing.T) {
	env := SetupAcceptanceTestEnv(t)
	defer env.Cleanup()

	projectDir := env.ProjectDir()

	// Setup: Create Claude installation with plugins
	env.CreateMarketplace("superpowers-marketplace", "github.com/superpowers-marketplace/superpowers")
	env.CreatePlugin("tdd-workflows", "superpowers-marketplace", "1.0.0", nil)
	env.CreatePlugin("backend-development", "superpowers-marketplace", "1.0.0", nil)

	// Create a profile with real plugin data
	p := &profile.Profile{
		Name:        "backend-go",
		Description: "Go backend development profile",
		Marketplaces: []profile.Marketplace{
			{Source: "github", Repo: "github.com/superpowers-marketplace/superpowers"},
		},
		Plugins: []string{
			"tdd-workflows@superpowers-marketplace",
			"backend-development@superpowers-marketplace",
		},
		MCPServers: []profile.MCPServer{},
	}

	// Save to project
	if err := profile.SaveToProject(projectDir, p); err != nil {
		t.Fatalf("Failed to save profile to project: %v", err)
	}

	// Verify profile exists in correct location
	profilePath := filepath.Join(projectDir, ".claudeup", "profiles", "backend-go.json")
	if _, err := os.Stat(profilePath); os.IsNotExist(err) {
		t.Fatalf("Profile not saved to project location: %s", profilePath)
	}

	// Load and verify contents
	loaded, err := profile.Load(filepath.Join(projectDir, ".claudeup", "profiles"), "backend-go")
	if err != nil {
		t.Fatalf("Failed to load saved profile: %v", err)
	}

	if loaded.Name != "backend-go" {
		t.Errorf("Expected name 'backend-go', got %q", loaded.Name)
	}

	if len(loaded.Marketplaces) != 1 {
		t.Errorf("Expected 1 marketplace, got %d", len(loaded.Marketplaces))
	}

	if len(loaded.Plugins) != 2 {
		t.Errorf("Expected 2 plugins, got %d", len(loaded.Plugins))
	}
}

// Section 3: ListAll shows profiles from both sources with correct labels
func TestListAllWithSources(t *testing.T) {
	env := SetupAcceptanceTestEnv(t)
	defer env.Cleanup()

	projectDir := env.ProjectDir()

	// Create user profile
	userProfile := &profile.Profile{
		Name:        "base-tools",
		Description: "Personal toolkit",
	}
	if err := profile.Save(env.ProfilesDir, userProfile); err != nil {
		t.Fatalf("Failed to save user profile: %v", err)
	}

	// Create project profile
	projectProfile := &profile.Profile{
		Name:        "team-config",
		Description: "Team configuration",
	}
	if err := profile.SaveToProject(projectDir, projectProfile); err != nil {
		t.Fatalf("Failed to save project profile: %v", err)
	}

	// List all profiles
	all, err := profile.ListAll(env.ProfilesDir, projectDir)
	if err != nil {
		t.Fatalf("ListAll failed: %v", err)
	}

	if len(all) != 2 {
		t.Fatalf("Expected 2 profiles, got %d", len(all))
	}

	// Verify sources (sorted alphabetically: base-tools, team-config)
	found := map[string]string{}
	for _, p := range all {
		found[p.Name] = p.Source
	}

	if found["base-tools"] != "user" {
		t.Errorf("Expected base-tools source 'user', got %q", found["base-tools"])
	}
	if found["team-config"] != "project" {
		t.Errorf("Expected team-config source 'project', got %q", found["team-config"])
	}
}

// Section 4: User base profile + project profile coexist
func TestUserAndProjectProfilesCoexist(t *testing.T) {
	env := SetupAcceptanceTestEnv(t)
	defer env.Cleanup()

	projectDir := env.ProjectDir()

	// Create user base profile
	baseProfile := &profile.Profile{
		Name:        "base-tools",
		Description: "TDD and personal tools",
		Plugins: []string{
			"superpowers@superpowers-marketplace",
			"tdd-workflows@claude-code-workflows",
		},
	}
	if err := profile.Save(env.ProfilesDir, baseProfile); err != nil {
		t.Fatalf("Failed to save base profile: %v", err)
	}

	// Create project profile
	goProfile := &profile.Profile{
		Name:        "backend-go",
		Description: "Go-specific plugins",
		Plugins: []string{
			"backend-development@claude-code-workflows",
			"security-scanning@claude-code-workflows",
		},
	}
	if err := profile.SaveToProject(projectDir, goProfile); err != nil {
		t.Fatalf("Failed to save project profile: %v", err)
	}

	// Both should be loadable
	all, err := profile.ListAll(env.ProfilesDir, projectDir)
	if err != nil {
		t.Fatalf("ListAll failed: %v", err)
	}

	if len(all) != 2 {
		t.Errorf("Expected 2 profiles (user + project), got %d", len(all))
	}

	// Verify each can be loaded correctly
	base, baseSource, err := profile.LoadWithFallback(env.ProfilesDir, projectDir, "base-tools")
	if err != nil {
		t.Fatalf("Failed to load base-tools: %v", err)
	}
	if baseSource != "user" {
		t.Errorf("Expected base-tools from 'user', got %q", baseSource)
	}
	if len(base.Plugins) != 2 {
		t.Errorf("Expected 2 plugins in base-tools, got %d", len(base.Plugins))
	}

	backend, backendSource, err := profile.LoadWithFallback(env.ProfilesDir, projectDir, "backend-go")
	if err != nil {
		t.Fatalf("Failed to load backend-go: %v", err)
	}
	if backendSource != "project" {
		t.Errorf("Expected backend-go from 'project', got %q", backendSource)
	}
	if len(backend.Plugins) != 2 {
		t.Errorf("Expected 2 plugins in backend-go, got %d", len(backend.Plugins))
	}
}

// Section 5: End-to-end Alice creates, Bob syncs
func TestAliceBobWorkflow(t *testing.T) {
	// This test simulates the complete workflow:
	// Alice: Saves profile to project → commits
	// Bob: Pulls → syncs → has the profile available

	env := SetupAcceptanceTestEnv(t)
	defer env.Cleanup()

	// === ALICE'S ENVIRONMENT ===
	aliceProjectDir := filepath.Join(env.HomeDir, "projects", "alice-project")
	if err := os.MkdirAll(aliceProjectDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Alice creates and saves a team profile
	teamProfile := &profile.Profile{
		Name:        "backend-go",
		Description: "Go backend development",
		Marketplaces: []profile.Marketplace{
			{Source: "github", Repo: "github.com/claude-code-workflows/claude-code-workflows"},
		},
		Plugins: []string{
			"tdd-workflows@claude-code-workflows",
			"backend-development@claude-code-workflows",
		},
	}

	// Alice saves to project scope
	if err := profile.SaveToProject(aliceProjectDir, teamProfile); err != nil {
		t.Fatalf("Alice: Failed to save profile to project: %v", err)
	}

	// Verify Alice's project now has the profile
	aliceProfilePath := filepath.Join(aliceProjectDir, ".claudeup", "profiles", "backend-go.json")
	if _, err := os.Stat(aliceProfilePath); os.IsNotExist(err) {
		t.Fatalf("Alice: Profile not saved to project: %s", aliceProfilePath)
	}

	// === SIMULATE GIT PUSH/PULL ===
	// In real world: git add .claudeup && git commit && git push
	// Bob: git clone / git pull

	// Simulate Bob getting Alice's files by copying .claudeup directory
	bobProjectDir := filepath.Join(env.HomeDir, "projects", "bob-project")
	bobClaudeupDir := filepath.Join(bobProjectDir, ".claudeup")

	// Copy Alice's .claudeup to Bob's project (simulates git pull)
	aliceClaudeupDir := filepath.Join(aliceProjectDir, ".claudeup")
	if err := copyDir(aliceClaudeupDir, bobClaudeupDir); err != nil {
		t.Fatalf("Failed to simulate git pull: %v", err)
	}

	// === BOB'S ENVIRONMENT ===
	// Bob has an empty user profiles directory (fresh setup)
	bobUserProfilesDir := filepath.Join(env.HomeDir, ".claudeup", "profiles")

	// Bob can now see the profile from the project
	all, err := profile.ListAll(bobUserProfilesDir, bobProjectDir)
	if err != nil {
		t.Fatalf("Bob: ListAll failed: %v", err)
	}

	if len(all) != 1 {
		t.Errorf("Bob: Expected 1 profile from project, got %d", len(all))
	}

	if all[0].Name != "backend-go" {
		t.Errorf("Bob: Expected profile 'backend-go', got %q", all[0].Name)
	}

	if all[0].Source != "project" {
		t.Errorf("Bob: Expected source 'project', got %q", all[0].Source)
	}

	// Bob loads the profile
	loaded, source, err := profile.LoadWithFallback(bobUserProfilesDir, bobProjectDir, "backend-go")
	if err != nil {
		t.Fatalf("Bob: Failed to load profile: %v", err)
	}

	if source != "project" {
		t.Errorf("Bob: Expected source 'project', got %q", source)
	}

	if len(loaded.Plugins) != 2 {
		t.Errorf("Bob: Expected 2 plugins, got %d", len(loaded.Plugins))
	}

	// Verify Bob has access to the plugins list for syncing
	expectedPlugins := map[string]bool{
		"tdd-workflows@claude-code-workflows":      true,
		"backend-development@claude-code-workflows": true,
	}

	for _, plugin := range loaded.Plugins {
		if !expectedPlugins[plugin] {
			t.Errorf("Bob: Unexpected plugin %q", plugin)
		}
	}
}

// TestProjectProfileShadowsUser verifies that project profiles override user profiles
func TestProjectProfileShadowsUser(t *testing.T) {
	env := SetupAcceptanceTestEnv(t)
	defer env.Cleanup()

	projectDir := env.ProjectDir()

	// Create user profile named "alpha"
	userAlpha := &profile.Profile{
		Name:        "alpha",
		Description: "User alpha - should be shadowed",
	}
	if err := profile.Save(env.ProfilesDir, userAlpha); err != nil {
		t.Fatalf("Failed to save user alpha: %v", err)
	}

	// Create project profile also named "alpha"
	projectAlpha := &profile.Profile{
		Name:        "alpha",
		Description: "Project alpha - should win",
	}
	if err := profile.SaveToProject(projectDir, projectAlpha); err != nil {
		t.Fatalf("Failed to save project alpha: %v", err)
	}

	// ListAll should only show one "alpha" from project
	all, err := profile.ListAll(env.ProfilesDir, projectDir)
	if err != nil {
		t.Fatalf("ListAll failed: %v", err)
	}

	alphaCount := 0
	for _, p := range all {
		if p.Name == "alpha" {
			alphaCount++
			if p.Source != "project" {
				t.Errorf("Expected alpha source 'project', got %q", p.Source)
			}
			if p.Description != "Project alpha - should win" {
				t.Errorf("Expected project description, got %q", p.Description)
			}
		}
	}

	if alphaCount != 1 {
		t.Errorf("Expected exactly 1 alpha profile (shadowed), got %d", alphaCount)
	}
}

// copyDir recursively copies a directory
func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		dstPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(dstPath, info.Mode())
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		return os.WriteFile(dstPath, data, info.Mode())
	})
}
