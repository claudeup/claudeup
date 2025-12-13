// ABOUTME: Integration tests for profile Snapshot functionality
// ABOUTME: Validates profile creation and display with various marketplace types
package integration

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/claudeup/claudeup/internal/profile"
)

func TestSnapshotCapturesGitURLMarketplace(t *testing.T) {
	env := setupSnapshotTestEnv(t)
	defer env.cleanup()

	// Create marketplace registry with both github and git URL sources
	marketplaces := map[string]interface{}{
		"claude-code-plugins": map[string]interface{}{
			"source": map[string]interface{}{
				"source": "github",
				"repo":   "anthropics/claude-code",
			},
			"installLocation": "/path/to/claude-code-plugins",
		},
		"every-marketplace": map[string]interface{}{
			"source": map[string]interface{}{
				"source": "git",
				"url":    "https://github.com/EveryInc/compound-engineering-plugin.git",
			},
			"installLocation": "/path/to/every-marketplace",
		},
	}
	env.createMarketplaceRegistry(marketplaces)

	// Create empty plugin and MCP registries
	env.createPluginRegistry(map[string]interface{}{})
	env.createClaudeJSON(map[string]interface{}{"mcpServers": map[string]interface{}{}})

	// Create snapshot
	p, err := profile.Snapshot("test-snapshot", env.claudeDir, env.claudeJSON)
	if err != nil {
		t.Fatalf("Snapshot failed: %v", err)
	}

	// Verify both marketplaces captured
	if len(p.Marketplaces) != 2 {
		t.Fatalf("Expected 2 marketplaces, got %d", len(p.Marketplaces))
	}

	// Verify DisplayName works for both
	foundGithub := false
	foundGitURL := false
	for _, m := range p.Marketplaces {
		displayName := m.DisplayName()
		if displayName == "" {
			t.Errorf("Empty display name for marketplace: source=%s", m.Source)
		}

		if m.Source == "github" && m.Repo == "anthropics/claude-code" {
			foundGithub = true
			if displayName != "anthropics/claude-code" {
				t.Errorf("GitHub marketplace display name wrong: got %q", displayName)
			}
		}

		if m.Source == "git" && m.URL == "https://github.com/EveryInc/compound-engineering-plugin.git" {
			foundGitURL = true
			if displayName != "https://github.com/EveryInc/compound-engineering-plugin.git" {
				t.Errorf("Git URL marketplace display name wrong: got %q", displayName)
			}
		}
	}

	if !foundGithub {
		t.Error("GitHub marketplace not found in snapshot")
	}
	if !foundGitURL {
		t.Error("Git URL marketplace not found in snapshot")
	}
}

func TestSnapshotSaveLoadRoundTrip(t *testing.T) {
	env := setupSnapshotTestEnv(t)
	defer env.cleanup()

	// Create marketplace registry with git URL source
	marketplaces := map[string]interface{}{
		"git-marketplace": map[string]interface{}{
			"source": map[string]interface{}{
				"source": "git",
				"url":    "https://example.com/plugin.git",
			},
		},
	}
	env.createMarketplaceRegistry(marketplaces)
	env.createPluginRegistry(map[string]interface{}{})
	env.createClaudeJSON(map[string]interface{}{"mcpServers": map[string]interface{}{}})

	// Create snapshot
	p, err := profile.Snapshot("roundtrip-test", env.claudeDir, env.claudeJSON)
	if err != nil {
		t.Fatalf("Snapshot failed: %v", err)
	}

	// Save the profile
	profilesDir := filepath.Join(env.claudeDir, "profiles")
	if err := profile.Save(profilesDir, p); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Load it back
	loaded, err := profile.Load(profilesDir, "roundtrip-test")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Verify the git URL marketplace survived the round trip
	if len(loaded.Marketplaces) != 1 {
		t.Fatalf("Expected 1 marketplace after load, got %d", len(loaded.Marketplaces))
	}

	m := loaded.Marketplaces[0]
	if m.Source != "git" {
		t.Errorf("Expected source 'git', got %q", m.Source)
	}
	if m.URL != "https://example.com/plugin.git" {
		t.Errorf("Expected URL preserved, got %q", m.URL)
	}
	if m.DisplayName() != "https://example.com/plugin.git" {
		t.Errorf("DisplayName() wrong after load: got %q", m.DisplayName())
	}
}

// Test environment helpers

type snapshotTestEnv struct {
	claudeDir  string
	claudeJSON string
	t          *testing.T
}

func setupSnapshotTestEnv(t *testing.T) *snapshotTestEnv {
	t.Helper()
	tmpDir := t.TempDir()

	claudeDir := filepath.Join(tmpDir, ".claude")
	pluginsDir := filepath.Join(claudeDir, "plugins")
	os.MkdirAll(pluginsDir, 0755)

	claudeJSON := filepath.Join(tmpDir, ".claude.json")

	return &snapshotTestEnv{
		claudeDir:  claudeDir,
		claudeJSON: claudeJSON,
		t:          t,
	}
}

func (e *snapshotTestEnv) cleanup() {
	// t.TempDir() handles cleanup
}

func (e *snapshotTestEnv) createPluginRegistry(plugins map[string]interface{}) {
	e.t.Helper()
	pluginsV2 := make(map[string]interface{})
	for name, meta := range plugins {
		metaMap, ok := meta.(map[string]interface{})
		if !ok {
			metaMap = make(map[string]interface{})
		}
		if _, hasScope := metaMap["scope"]; !hasScope {
			metaMap["scope"] = "user"
		}
		pluginsV2[name] = []interface{}{metaMap}
	}
	data := map[string]interface{}{
		"version": 2,
		"plugins": pluginsV2,
	}
	e.writeJSON(filepath.Join(e.claudeDir, "plugins", "installed_plugins.json"), data)
}

func (e *snapshotTestEnv) createMarketplaceRegistry(marketplaces map[string]interface{}) {
	e.t.Helper()
	e.writeJSON(filepath.Join(e.claudeDir, "plugins", "known_marketplaces.json"), marketplaces)
}

func (e *snapshotTestEnv) createClaudeJSON(data map[string]interface{}) {
	e.t.Helper()
	e.writeJSON(e.claudeJSON, data)
}

func (e *snapshotTestEnv) writeJSON(path string, data interface{}) {
	e.t.Helper()
	bytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		e.t.Fatal(err)
	}
	if err := os.WriteFile(path, bytes, 0644); err != nil {
		e.t.Fatal(err)
	}
}
