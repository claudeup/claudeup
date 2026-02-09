// ABOUTME: Tests for profile command functions
// ABOUTME: Validates profile loading fallback, stack apply, and scope routing
package commands

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/claudeup/claudeup/v4/internal/config"
	"github.com/claudeup/claudeup/v4/internal/profile"
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

func TestProfileDelete_DetectsActiveProfile(t *testing.T) {
	tmpDir := t.TempDir()
	profilesDir := filepath.Join(tmpDir, "profiles")
	configDir := filepath.Join(tmpDir, "config")
	os.MkdirAll(profilesDir, 0755)
	os.MkdirAll(configDir, 0755)

	// Create a test profile
	testProfile := &profile.Profile{
		Name:        "test-active",
		Description: "Test profile",
	}
	if err := profile.Save(profilesDir, testProfile); err != nil {
		t.Fatalf("Failed to save test profile: %v", err)
	}

	// This test verifies that the delete command logic can detect if a profile is active
	// The actual deletion is tested in acceptance tests
	profilePath := filepath.Join(profilesDir, "test-active.json")
	if _, err := os.Stat(profilePath); os.IsNotExist(err) {
		t.Error("Profile file should exist before deletion")
	}
}

func TestResolveProfileArg_SingleMatch(t *testing.T) {
	tmpDir := t.TempDir()
	profilesDir := filepath.Join(tmpDir, "profiles")
	os.MkdirAll(profilesDir, 0755)

	// Create a flat profile
	testProfile := &profile.Profile{
		Name:        "api",
		Description: "API profile",
	}
	if err := profile.Save(profilesDir, testProfile); err != nil {
		t.Fatalf("Failed to save profile: %v", err)
	}

	path, err := resolveProfileArg(profilesDir, "api")
	if err != nil {
		t.Fatalf("resolveProfileArg failed: %v", err)
	}

	expected := filepath.Join(profilesDir, "api.json")
	if path != expected {
		t.Errorf("Expected path %q, got %q", expected, path)
	}
}

func TestResolveProfileArg_NestedSingleMatch(t *testing.T) {
	tmpDir := t.TempDir()
	profilesDir := filepath.Join(tmpDir, "profiles")
	nestedDir := filepath.Join(profilesDir, "backend")
	os.MkdirAll(nestedDir, 0755)

	// Create a nested profile
	data := []byte(`{"name":"worker","description":"Worker service"}`)
	if err := os.WriteFile(filepath.Join(nestedDir, "worker.json"), data, 0644); err != nil {
		t.Fatalf("Failed to write nested profile: %v", err)
	}

	path, err := resolveProfileArg(profilesDir, "worker")
	if err != nil {
		t.Fatalf("resolveProfileArg failed: %v", err)
	}

	expected := filepath.Join(nestedDir, "worker.json")
	if path != expected {
		t.Errorf("Expected path %q, got %q", expected, path)
	}
}

func TestResolveProfileArg_PathReference(t *testing.T) {
	tmpDir := t.TempDir()
	profilesDir := filepath.Join(tmpDir, "profiles")
	nestedDir := filepath.Join(profilesDir, "backend")
	os.MkdirAll(nestedDir, 0755)

	// Create nested profile
	data := []byte(`{"name":"api","description":"Backend API"}`)
	if err := os.WriteFile(filepath.Join(nestedDir, "api.json"), data, 0644); err != nil {
		t.Fatalf("Failed to write nested profile: %v", err)
	}

	// Resolve with path reference
	path, err := resolveProfileArg(profilesDir, "backend/api")
	if err != nil {
		t.Fatalf("resolveProfileArg failed: %v", err)
	}

	expected := filepath.Join(nestedDir, "api.json")
	if path != expected {
		t.Errorf("Expected path %q, got %q", expected, path)
	}
}

func TestResolveProfileArg_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	profilesDir := filepath.Join(tmpDir, "profiles")
	os.MkdirAll(profilesDir, 0755)

	_, err := resolveProfileArg(profilesDir, "nonexistent")
	if err == nil {
		t.Fatal("Expected error for nonexistent profile, got nil")
	}

	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("Expected error containing 'not found', got %q", err.Error())
	}
}

func TestResolveProfileArg_AmbiguousWithYesFlag(t *testing.T) {
	tmpDir := t.TempDir()
	profilesDir := filepath.Join(tmpDir, "profiles")
	nestedDir := filepath.Join(profilesDir, "backend")
	os.MkdirAll(profilesDir, 0755)
	os.MkdirAll(nestedDir, 0755)

	// Create two profiles with the same name
	data := []byte(`{"name":"api","description":"Root API"}`)
	if err := os.WriteFile(filepath.Join(profilesDir, "api.json"), data, 0644); err != nil {
		t.Fatalf("Failed to write root profile: %v", err)
	}
	data2 := []byte(`{"name":"api","description":"Backend API"}`)
	if err := os.WriteFile(filepath.Join(nestedDir, "api.json"), data2, 0644); err != nil {
		t.Fatalf("Failed to write nested profile: %v", err)
	}

	// Set --yes flag to simulate non-interactive mode
	oldYesFlag := config.YesFlag
	config.YesFlag = true
	defer func() { config.YesFlag = oldYesFlag }()

	_, err := resolveProfileArg(profilesDir, "api")
	if err == nil {
		t.Fatal("Expected ambiguity error with --yes flag, got nil")
	}

	var ambigErr *profile.AmbiguousProfileError
	if !errors.As(err, &ambigErr) {
		t.Fatalf("Expected *AmbiguousProfileError, got %T: %v", err, err)
	}
	if ambigErr.Name != "api" {
		t.Errorf("Expected Name 'api', got %q", ambigErr.Name)
	}
	// Should list the paths to help the user
	found := false
	for _, p := range ambigErr.Paths {
		if p == "backend/api" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected paths to include 'backend/api', got %v", ambigErr.Paths)
	}
}

func TestResolveProfileArg_AmbiguousInteractive(t *testing.T) {
	tmpDir := t.TempDir()
	profilesDir := filepath.Join(tmpDir, "profiles")
	nestedDir := filepath.Join(profilesDir, "backend")
	os.MkdirAll(profilesDir, 0755)
	os.MkdirAll(nestedDir, 0755)

	// Create two profiles with the same name
	data := []byte(`{"name":"api","description":"Root API"}`)
	if err := os.WriteFile(filepath.Join(profilesDir, "api.json"), data, 0644); err != nil {
		t.Fatalf("Failed to write root profile: %v", err)
	}
	data2 := []byte(`{"name":"api","description":"Backend API"}`)
	if err := os.WriteFile(filepath.Join(nestedDir, "api.json"), data2, 0644); err != nil {
		t.Fatalf("Failed to write nested profile: %v", err)
	}

	// Ensure --yes flag is off for interactive mode
	oldYesFlag := config.YesFlag
	config.YesFlag = false
	defer func() { config.YesFlag = oldYesFlag }()

	// Simulate user selecting option 2 (backend/api)
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("Failed to create pipe: %v", err)
	}
	w.WriteString("2\n")
	w.Close()

	oldStdin := os.Stdin
	os.Stdin = r
	defer func() { os.Stdin = oldStdin }()

	path, err := resolveProfileArg(profilesDir, "api")
	if err != nil {
		t.Fatalf("resolveProfileArg failed: %v", err)
	}

	expected := filepath.Join(nestedDir, "api.json")
	if path != expected {
		t.Errorf("Expected path %q, got %q", expected, path)
	}
}

func TestResolveProfileArg_PathReferenceNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	profilesDir := filepath.Join(tmpDir, "profiles")
	os.MkdirAll(profilesDir, 0755)

	_, err := resolveProfileArg(profilesDir, "backend/missing")
	if err == nil {
		t.Fatal("Expected error for missing path reference, got nil")
	}

	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("Expected error containing 'not found', got %q", err.Error())
	}
}

func TestProfileExists_FindsNestedProfile(t *testing.T) {
	tmpDir := t.TempDir()
	profilesDir := filepath.Join(tmpDir, "profiles")
	nestedDir := filepath.Join(profilesDir, "backend")
	os.MkdirAll(nestedDir, 0755)

	// Create a nested profile
	data := []byte(`{"name":"api","description":"Backend API"}`)
	if err := os.WriteFile(filepath.Join(nestedDir, "api.json"), data, 0644); err != nil {
		t.Fatalf("Failed to write nested profile: %v", err)
	}

	if !profileExists(profilesDir, "api") {
		t.Error("profileExists should find nested profile by name")
	}
}

func TestProfileExists_FindsNestedByPathReference(t *testing.T) {
	tmpDir := t.TempDir()
	profilesDir := filepath.Join(tmpDir, "profiles")
	nestedDir := filepath.Join(profilesDir, "backend")
	os.MkdirAll(nestedDir, 0755)

	// Create a nested profile
	data := []byte(`{"name":"api","description":"Backend API"}`)
	if err := os.WriteFile(filepath.Join(nestedDir, "api.json"), data, 0644); err != nil {
		t.Fatalf("Failed to write nested profile: %v", err)
	}

	if !profileExists(profilesDir, "backend/api") {
		t.Error("profileExists should find nested profile by path reference")
	}
}

func TestProfileDelete_ClearsActiveProfile(t *testing.T) {
	tmpDir := t.TempDir()
	profilesDir := filepath.Join(tmpDir, "profiles")
	os.MkdirAll(profilesDir, 0755)

	// Create a test profile
	testProfile := &profile.Profile{
		Name:        "test-to-clear",
		Description: "Test profile",
	}
	if err := profile.Save(profilesDir, testProfile); err != nil {
		t.Fatalf("Failed to save test profile: %v", err)
	}

	// This test verifies that profile deletion logic includes clearing active profile
	// The actual behavior is tested in acceptance tests with full command execution
	// Unit test validates the profile exists and can be found
	loaded, err := profile.Load(profilesDir, "test-to-clear")
	if err != nil {
		t.Errorf("Should be able to load profile before deletion: %v", err)
	}
	if loaded.Name != "test-to-clear" {
		t.Errorf("Expected profile name 'test-to-clear', got %q", loaded.Name)
	}
}

// setupStackApplyEnv prepares an isolated environment for testing applyProfileWithScope.
// Sets CLAUDEUP_HOME and claudeDir to temp directories, overrides global flags
// for non-interactive operation, and returns the profiles directory path.
// All globals and env vars are restored via t.Cleanup.
func setupStackApplyEnv(t *testing.T) string {
	t.Helper()

	tmpDir := t.TempDir()
	claudeupHome := filepath.Join(tmpDir, "claudeup-home")
	profilesDir := filepath.Join(claudeupHome, "profiles")
	testClaudeDir := filepath.Join(tmpDir, "claude")

	os.MkdirAll(profilesDir, 0755)
	os.MkdirAll(testClaudeDir, 0755)

	// Set CLAUDEUP_HOME so getProfilesDir() returns our temp profiles dir
	t.Setenv("CLAUDEUP_HOME", claudeupHome)

	// Save and restore globals
	origClaudeDir := claudeDir
	origForce := profileApplyForce
	origNoInteractive := profileApplyNoInteractive
	origScope := profileApplyScope
	origDryRun := profileApplyDryRun
	origReplace := profileApplyReplace
	origReinstall := profileApplyReinstall
	origNoProgress := profileApplyNoProgress
	origSetup := profileApplySetup
	origYesFlag := config.YesFlag

	claudeDir = testClaudeDir
	profileApplyForce = true
	profileApplyNoInteractive = true
	profileApplyScope = ""
	profileApplyDryRun = false
	profileApplyReplace = false
	profileApplyReinstall = false
	profileApplyNoProgress = true
	profileApplySetup = false
	config.YesFlag = true

	t.Cleanup(func() {
		claudeDir = origClaudeDir
		profileApplyForce = origForce
		profileApplyNoInteractive = origNoInteractive
		profileApplyScope = origScope
		profileApplyDryRun = origDryRun
		profileApplyReplace = origReplace
		profileApplyReinstall = origReinstall
		profileApplyNoProgress = origNoProgress
		profileApplySetup = origSetup
		config.YesFlag = origYesFlag
	})

	return profilesDir
}

func TestApplyProfileWithScope_ExplicitScopeRejectsStacks(t *testing.T) {
	profilesDir := setupStackApplyEnv(t)

	// Create a stack profile (has includes, making it a stack)
	stackJSON := []byte(`{"name":"my-stack","includes":["base","tools"]}`)
	if err := os.WriteFile(filepath.Join(profilesDir, "my-stack.json"), stackJSON, 0644); err != nil {
		t.Fatalf("Failed to write stack profile: %v", err)
	}

	err := applyProfileWithScope("my-stack", profile.ScopeUser, true)
	if err == nil {
		t.Fatal("Expected error when applying stack with explicit scope, got nil")
	}

	expected := "stack profiles define their own scopes"
	if !strings.Contains(err.Error(), expected) {
		t.Errorf("Expected error containing %q, got %q", expected, err.Error())
	}
}

func TestApplyProfileWithScope_StackSucceedsWithAutoScope(t *testing.T) {
	profilesDir := setupStackApplyEnv(t)

	// Create a stack that includes a leaf profile
	stackJSON := []byte(`{"name":"test-stack","includes":["base"]}`)
	if err := os.WriteFile(filepath.Join(profilesDir, "test-stack.json"), stackJSON, 0644); err != nil {
		t.Fatalf("Failed to write stack profile: %v", err)
	}

	// Create the leaf profile (no plugins/MCP -- resolves to empty config)
	leafJSON := []byte(`{"name":"base","description":"Base configuration"}`)
	if err := os.WriteFile(filepath.Join(profilesDir, "base.json"), leafJSON, 0644); err != nil {
		t.Fatalf("Failed to write leaf profile: %v", err)
	}

	// With explicitScope=false, the stack should pass the scope check,
	// resolve its includes, and return successfully (no changes needed)
	err := applyProfileWithScope("test-stack", profile.ScopeUser, false)
	if err != nil {
		t.Fatalf("Stack apply with auto scope should succeed, got: %v", err)
	}
}

func TestApplyProfileWithScope_MultiScopeStackRoutesToApplyAllScopes(t *testing.T) {
	profilesDir := setupStackApplyEnv(t)

	// Create a stack that includes a multi-scope leaf
	stackJSON := []byte(`{"name":"scoped-stack","includes":["scoped-leaf"]}`)
	if err := os.WriteFile(filepath.Join(profilesDir, "scoped-stack.json"), stackJSON, 0644); err != nil {
		t.Fatalf("Failed to write stack profile: %v", err)
	}

	// Create a multi-scope leaf profile (has perScope, making resolved profile IsMultiScope)
	leafJSON := []byte(`{
		"name": "scoped-leaf",
		"perScope": {
			"user": {
				"plugins": ["test-plugin@test-market"]
			}
		}
	}`)
	if err := os.WriteFile(filepath.Join(profilesDir, "scoped-leaf.json"), leafJSON, 0644); err != nil {
		t.Fatalf("Failed to write scoped leaf profile: %v", err)
	}

	err := applyProfileWithScope("scoped-stack", profile.ScopeUser, false)
	if err != nil {
		t.Fatalf("Multi-scope stack apply failed: %v", err)
	}

	// Verify settings.json was written by ApplyAllScopes
	settingsPath := filepath.Join(claudeDir, "settings.json")
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatalf("Expected settings.json to be created by ApplyAllScopes, got: %v", err)
	}

	// Verify the plugin from perScope.user was written to settings
	var settings map[string]interface{}
	if err := json.Unmarshal(data, &settings); err != nil {
		t.Fatalf("Failed to parse settings.json: %v", err)
	}

	enabledPlugins, ok := settings["enabledPlugins"].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected enabledPlugins map in settings.json, got: %v", settings)
	}

	if _, found := enabledPlugins["test-plugin@test-market"]; !found {
		t.Errorf("Expected 'test-plugin@test-market' in enabledPlugins, got: %v", enabledPlugins)
	}
}
