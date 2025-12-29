package profile

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Helper to create a test profile with plugins and return profilesDir
func setupTestProfile(t *testing.T, projectDir string, plugins []string, marketplaces []Marketplace) string {
	profilesDir := filepath.Join(projectDir, "profiles")
	if err := os.MkdirAll(profilesDir, 0755); err != nil {
		t.Fatalf("failed to create profiles dir: %v", err)
	}

	testProfile := &Profile{
		Name:         "test-profile",
		Plugins:      plugins,
		Marketplaces: marketplaces,
	}
	if err := Save(profilesDir, testProfile); err != nil {
		t.Fatalf("failed to save test profile: %v", err)
	}

	// Create .claudeup.json referencing the profile
	cfg := &ProjectConfig{
		Version: "1",
		Profile: "test-profile",
	}
	if err := SaveProjectConfig(projectDir, cfg); err != nil {
		t.Fatalf("failed to save project config: %v", err)
	}

	return profilesDir
}

// syncMockExecutor records commands for testing sync operations
type syncMockExecutor struct {
	commands      [][]string
	failOn        map[string]bool   // command prefixes that should fail
	outputFor     map[string]string // custom output for specific commands
	alreadyExists map[string]bool   // plugins that should report "already installed"
}

func newSyncMockExecutor() *syncMockExecutor {
	return &syncMockExecutor{
		commands:      [][]string{},
		failOn:        make(map[string]bool),
		outputFor:     make(map[string]string),
		alreadyExists: make(map[string]bool),
	}
}

func (m *syncMockExecutor) Run(args ...string) error {
	m.commands = append(m.commands, args)
	if len(args) > 0 && m.failOn != nil {
		key := strings.Join(args[:min(3, len(args))], " ")
		if m.failOn[key] {
			return errMockFailure(key)
		}
	}
	return nil
}

func (m *syncMockExecutor) RunWithOutput(args ...string) (string, error) {
	m.commands = append(m.commands, args)

	// Check for custom output
	key := strings.Join(args, " ")
	if output, ok := m.outputFor[key]; ok {
		return output, nil
	}

	// Check for "already installed" plugins - returns error with output
	if len(args) >= 4 && args[0] == "plugin" && args[1] == "install" {
		plugin := args[len(args)-1]
		if m.alreadyExists[plugin] {
			return "already installed", errMockFailure("already installed")
		}
	}

	// Check for failures
	if len(args) > 0 && m.failOn != nil {
		shortKey := strings.Join(args[:min(3, len(args))], " ")
		if m.failOn[shortKey] {
			return "mock failure output", errMockFailure(shortKey)
		}
	}

	return "", nil
}

func errMockFailure(key string) error {
	return &mockError{key: key}
}

type mockError struct {
	key string
}

func (e *mockError) Error() string {
	return "mock failure for: " + e.key
}

func TestSyncWithExecutor_InstallsPlugins(t *testing.T) {
	// Setup temp directories
	projectDir := t.TempDir()
	claudeDir := filepath.Join(t.TempDir(), ".claude")
	pluginsDir := filepath.Join(claudeDir, "plugins")
	os.MkdirAll(pluginsDir, 0755)

	// Create test profile with plugins
	profilesDir := setupTestProfile(t, projectDir,
		[]string{"plugin-a@test-plugins", "plugin-b@test-plugins"},
		[]Marketplace{{Source: "github", Repo: "test/plugins"}})

	// Create empty installed_plugins.json (no plugins installed)
	emptyPlugins := map[string]interface{}{
		"version": 2,
		"plugins": map[string]interface{}{},
	}
	pluginsData, _ := json.Marshal(emptyPlugins)
	os.WriteFile(filepath.Join(pluginsDir, "installed_plugins.json"), pluginsData, 0644)

	// Run sync
	executor := newSyncMockExecutor()
	result, err := SyncWithExecutor(profilesDir, projectDir, claudeDir, SyncOptions{}, executor)
	if err != nil {
		t.Fatalf("SyncWithExecutor failed: %v", err)
	}

	// Verify results
	if result.PluginsInstalled != 2 {
		t.Errorf("PluginsInstalled = %d, want 2", result.PluginsInstalled)
	}
	if result.PluginsSkipped != 0 {
		t.Errorf("PluginsSkipped = %d, want 0", result.PluginsSkipped)
	}
	if result.MarketplacesAdded != 1 {
		t.Errorf("MarketplacesAdded = %d, want 1", result.MarketplacesAdded)
	}

	// Verify commands were called
	foundMarketplace := false
	pluginsInstalled := 0
	for _, cmd := range executor.commands {
		if len(cmd) >= 4 && cmd[0] == "plugin" && cmd[1] == "marketplace" && cmd[2] == "add" {
			foundMarketplace = true
		}
		if len(cmd) >= 4 && cmd[0] == "plugin" && cmd[1] == "install" && cmd[2] == "--scope" && cmd[3] == "project" {
			pluginsInstalled++
		}
	}
	if !foundMarketplace {
		t.Error("marketplace add command not found")
	}
	if pluginsInstalled != 2 {
		t.Errorf("plugin install commands = %d, want 2", pluginsInstalled)
	}
}

func TestSyncWithExecutor_SkipsExistingPlugins(t *testing.T) {
	projectDir := t.TempDir()
	claudeDir := filepath.Join(t.TempDir(), ".claude")
	pluginsDir := filepath.Join(claudeDir, "plugins")
	os.MkdirAll(pluginsDir, 0755)

	// Create test profile
	profilesDir := setupTestProfile(t, projectDir,
		[]string{"existing-plugin@test", "new-plugin@test"},
		nil)

	// Create installed_plugins.json with one plugin already installed
	installedPlugins := map[string]interface{}{
		"version": 2,
		"plugins": map[string]interface{}{
			"existing-plugin@test": []map[string]interface{}{{"scope": "project", "version": "1.0"}},
		},
	}
	pluginsData, _ := json.Marshal(installedPlugins)
	os.WriteFile(filepath.Join(pluginsDir, "installed_plugins.json"), pluginsData, 0644)

	executor := newSyncMockExecutor()
	result, err := SyncWithExecutor(profilesDir, projectDir, claudeDir, SyncOptions{}, executor)
	if err != nil {
		t.Fatalf("SyncWithExecutor failed: %v", err)
	}

	if result.PluginsInstalled != 1 {
		t.Errorf("PluginsInstalled = %d, want 1", result.PluginsInstalled)
	}
	if result.PluginsSkipped != 1 {
		t.Errorf("PluginsSkipped = %d, want 1", result.PluginsSkipped)
	}
}

func TestSyncWithExecutor_DryRun(t *testing.T) {
	projectDir := t.TempDir()
	claudeDir := filepath.Join(t.TempDir(), ".claude")
	pluginsDir := filepath.Join(claudeDir, "plugins")
	os.MkdirAll(pluginsDir, 0755)

	// Create test profile
	profilesDir := setupTestProfile(t, projectDir,
		[]string{"plugin-a@test", "plugin-b@test"},
		[]Marketplace{{Source: "github", Repo: "test/plugins"}})

	// Empty installed plugins
	emptyPlugins := map[string]interface{}{"version": 2, "plugins": map[string]interface{}{}}
	pluginsData, _ := json.Marshal(emptyPlugins)
	os.WriteFile(filepath.Join(pluginsDir, "installed_plugins.json"), pluginsData, 0644)

	executor := newSyncMockExecutor()
	result, err := SyncWithExecutor(profilesDir, projectDir, claudeDir, SyncOptions{DryRun: true}, executor)
	if err != nil {
		t.Fatalf("SyncWithExecutor failed: %v", err)
	}

	// Dry run should report what would be installed
	if result.PluginsInstalled != 2 {
		t.Errorf("PluginsInstalled = %d, want 2", result.PluginsInstalled)
	}
	if result.MarketplacesAdded != 1 {
		t.Errorf("MarketplacesAdded = %d, want 1", result.MarketplacesAdded)
	}

	// But no commands should have been executed
	if len(executor.commands) != 0 {
		t.Errorf("Commands executed in dry run: %d, want 0", len(executor.commands))
	}
}

func TestSyncWithExecutor_NoProjectConfig(t *testing.T) {
	projectDir := t.TempDir()
	claudeDir := t.TempDir()

	executor := newSyncMockExecutor()
	profilesDir := filepath.Join(projectDir, "profiles")
	_, err := SyncWithExecutor(profilesDir, projectDir, claudeDir, SyncOptions{}, executor)
	if err == nil {
		t.Error("expected error for missing .claudeup.json")
	}
	if !strings.Contains(err.Error(), ProjectConfigFile) {
		t.Errorf("error should mention %s: %v", ProjectConfigFile, err)
	}
}

func TestSyncWithExecutor_HandlesPluginInstallFailure(t *testing.T) {
	projectDir := t.TempDir()
	claudeDir := filepath.Join(t.TempDir(), ".claude")
	pluginsDir := filepath.Join(claudeDir, "plugins")
	os.MkdirAll(pluginsDir, 0755)

	profilesDir := setupTestProfile(t, projectDir,
		[]string{"good-plugin@test", "bad-plugin@test"},
		nil)

	emptyPlugins := map[string]interface{}{"version": 2, "plugins": map[string]interface{}{}}
	pluginsData, _ := json.Marshal(emptyPlugins)
	os.WriteFile(filepath.Join(pluginsDir, "installed_plugins.json"), pluginsData, 0644)

	executor := newSyncMockExecutor()
	// Make bad-plugin fail
	executor.failOn["plugin install --scope"] = true

	result, err := SyncWithExecutor(profilesDir, projectDir, claudeDir, SyncOptions{}, executor)
	if err != nil {
		t.Fatalf("SyncWithExecutor failed: %v", err)
	}

	// Should have errors for failed plugins
	if len(result.Errors) == 0 {
		t.Error("expected errors for failed plugin installs")
	}
}

func TestSyncWithExecutor_HandlesAlreadyInstalledOutput(t *testing.T) {
	projectDir := t.TempDir()
	claudeDir := filepath.Join(t.TempDir(), ".claude")
	pluginsDir := filepath.Join(claudeDir, "plugins")
	os.MkdirAll(pluginsDir, 0755)

	profilesDir := setupTestProfile(t, projectDir,
		[]string{"plugin@test"},
		nil)

	// Plugin not in installed_plugins.json but CLI says "already installed"
	emptyPlugins := map[string]interface{}{"version": 2, "plugins": map[string]interface{}{}}
	pluginsData, _ := json.Marshal(emptyPlugins)
	os.WriteFile(filepath.Join(pluginsDir, "installed_plugins.json"), pluginsData, 0644)

	executor := newSyncMockExecutor()
	executor.alreadyExists["plugin@test"] = true

	result, err := SyncWithExecutor(profilesDir, projectDir, claudeDir, SyncOptions{}, executor)
	if err != nil {
		t.Fatalf("SyncWithExecutor failed: %v", err)
	}

	// Should count as skipped, not installed
	if result.PluginsSkipped != 1 {
		t.Errorf("PluginsSkipped = %d, want 1", result.PluginsSkipped)
	}
	if result.PluginsInstalled != 0 {
		t.Errorf("PluginsInstalled = %d, want 0", result.PluginsInstalled)
	}
}

func TestSyncWithExecutor_Idempotent(t *testing.T) {
	projectDir := t.TempDir()
	claudeDir := filepath.Join(t.TempDir(), ".claude")
	pluginsDir := filepath.Join(claudeDir, "plugins")
	os.MkdirAll(pluginsDir, 0755)

	profilesDir := setupTestProfile(t, projectDir,
		[]string{"plugin@test"},
		[]Marketplace{{Source: "github", Repo: "test/plugins"}})

	// Plugin already installed
	installedPlugins := map[string]interface{}{
		"version": 2,
		"plugins": map[string]interface{}{
			"plugin@test": []map[string]interface{}{{"scope": "project", "version": "1.0"}},
		},
	}
	pluginsData, _ := json.Marshal(installedPlugins)
	os.WriteFile(filepath.Join(pluginsDir, "installed_plugins.json"), pluginsData, 0644)

	// Run sync twice
	executor := newSyncMockExecutor()
	result1, _ := SyncWithExecutor(profilesDir, projectDir, claudeDir, SyncOptions{}, executor)

	executor2 := newSyncMockExecutor()
	result2, _ := SyncWithExecutor(profilesDir, projectDir, claudeDir, SyncOptions{}, executor2)

	// Both runs should have same result - nothing new to install
	if result1.PluginsSkipped != result2.PluginsSkipped {
		t.Errorf("Sync not idempotent: first run skipped %d, second run skipped %d",
			result1.PluginsSkipped, result2.PluginsSkipped)
	}
	if result1.PluginsInstalled != result2.PluginsInstalled {
		t.Errorf("Sync not idempotent: first run installed %d, second run installed %d",
			result1.PluginsInstalled, result2.PluginsInstalled)
	}
}

func TestSyncWithExecutor_EmptyPluginsList(t *testing.T) {
	projectDir := t.TempDir()
	claudeDir := filepath.Join(t.TempDir(), ".claude")
	pluginsDir := filepath.Join(claudeDir, "plugins")
	os.MkdirAll(pluginsDir, 0755)

	// Create test profile with empty plugins list
	profilesDir := setupTestProfile(t, projectDir,
		[]string{},
		nil)

	emptyPlugins := map[string]interface{}{"version": 2, "plugins": map[string]interface{}{}}
	pluginsData, _ := json.Marshal(emptyPlugins)
	os.WriteFile(filepath.Join(pluginsDir, "installed_plugins.json"), pluginsData, 0644)

	executor := newSyncMockExecutor()
	result, err := SyncWithExecutor(profilesDir, projectDir, claudeDir, SyncOptions{}, executor)
	if err != nil {
		t.Fatalf("SyncWithExecutor failed: %v", err)
	}

	if result.PluginsInstalled != 0 {
		t.Errorf("PluginsInstalled = %d, want 0", result.PluginsInstalled)
	}
	if result.PluginsSkipped != 0 {
		t.Errorf("PluginsSkipped = %d, want 0", result.PluginsSkipped)
	}
}

func TestSyncWithExecutor_LoadsFromProject(t *testing.T) {
	tmpDir := t.TempDir()
	userProfilesDir := filepath.Join(tmpDir, "user-profiles")
	projectDir := filepath.Join(tmpDir, "project")
	claudeDir := filepath.Join(tmpDir, "claude")
	projectProfilesDir := filepath.Join(projectDir, ".claudeup", "profiles")
	pluginsDir := filepath.Join(claudeDir, "plugins")

	// Create directories
	os.MkdirAll(userProfilesDir, 0755)
	os.MkdirAll(projectProfilesDir, 0755)
	os.MkdirAll(pluginsDir, 0755)

	// Create project profile with a specific plugin
	projectProfile := &Profile{
		Name:    "team-config",
		Plugins: []string{"project-plugin@marketplace"},
	}
	if err := Save(projectProfilesDir, projectProfile); err != nil {
		t.Fatal(err)
	}

	// Create .claudeup.json pointing to the profile
	cfg := &ProjectConfig{Profile: "team-config"}
	if err := SaveProjectConfig(projectDir, cfg); err != nil {
		t.Fatal(err)
	}

	// Create empty installed_plugins.json
	emptyPlugins := map[string]interface{}{"version": 2, "plugins": map[string]interface{}{}}
	pluginsData, _ := json.Marshal(emptyPlugins)
	os.WriteFile(filepath.Join(pluginsDir, "installed_plugins.json"), pluginsData, 0644)

	// Create mock executor that records what plugins are installed
	executor := newSyncMockExecutor()

	result, err := SyncWithExecutor(userProfilesDir, projectDir, claudeDir, SyncOptions{}, executor)
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}

	if result.PluginsInstalled != 1 {
		t.Errorf("Expected 1 plugin installed, got %d", result.PluginsInstalled)
	}

	// Verify the correct plugin was installed
	foundPlugin := false
	for _, cmd := range executor.commands {
		if len(cmd) >= 5 && cmd[0] == "plugin" && cmd[1] == "install" && cmd[4] == "project-plugin@marketplace" {
			foundPlugin = true
			break
		}
	}
	if !foundPlugin {
		t.Errorf("Expected project-plugin@marketplace to be installed, commands: %v", executor.commands)
	}
}
