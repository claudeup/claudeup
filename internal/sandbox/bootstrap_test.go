// ABOUTME: Unit tests for profile bootstrap functionality.
// ABOUTME: Tests first-run detection, config writing, and sentinel management.
package sandbox

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/claudeup/claudeup/internal/profile"
)

func TestIsFirstRun(t *testing.T) {
	t.Run("empty directory is first run", func(t *testing.T) {
		stateDir := t.TempDir()
		if !IsFirstRun(stateDir) {
			t.Error("expected first run for empty directory")
		}
	})

	t.Run("directory with sentinel is not first run", func(t *testing.T) {
		stateDir := t.TempDir()
		sentinel := filepath.Join(stateDir, ".bootstrapped")
		if err := os.WriteFile(sentinel, []byte("2026-01-01"), 0644); err != nil {
			t.Fatal(err)
		}
		if IsFirstRun(stateDir) {
			t.Error("expected not first run when sentinel exists")
		}
	})

	t.Run("directory with other files but no sentinel is first run", func(t *testing.T) {
		stateDir := t.TempDir()
		otherFile := filepath.Join(stateDir, "settings.json")
		if err := os.WriteFile(otherFile, []byte("{}"), 0644); err != nil {
			t.Fatal(err)
		}
		if !IsFirstRun(stateDir) {
			t.Error("expected first run when no sentinel")
		}
	})
}

func TestBootstrapFromProfile(t *testing.T) {
	t.Run("writes marketplaces config", func(t *testing.T) {
		stateDir := t.TempDir()
		p := &profile.Profile{
			Name: "test",
			Marketplaces: []profile.Marketplace{
				{Source: "github", Repo: "obra/superpowers-marketplace"},
			},
		}

		if err := BootstrapFromProfile(p, stateDir); err != nil {
			t.Fatalf("BootstrapFromProfile failed: %v", err)
		}

		// Check marketplaces.json was created
		data, err := os.ReadFile(filepath.Join(stateDir, "marketplaces.json"))
		if err != nil {
			t.Fatalf("failed to read marketplaces.json: %v", err)
		}

		var marketplaces []map[string]interface{}
		if err := json.Unmarshal(data, &marketplaces); err != nil {
			t.Fatalf("failed to parse marketplaces.json: %v", err)
		}

		if len(marketplaces) != 1 {
			t.Errorf("expected 1 marketplace, got %d", len(marketplaces))
		}
	})

	t.Run("writes plugins to settings.json", func(t *testing.T) {
		stateDir := t.TempDir()
		p := &profile.Profile{
			Name:    "test",
			Plugins: []string{"superpowers@superpowers-marketplace"},
		}

		if err := BootstrapFromProfile(p, stateDir); err != nil {
			t.Fatalf("BootstrapFromProfile failed: %v", err)
		}

		data, err := os.ReadFile(filepath.Join(stateDir, "settings.json"))
		if err != nil {
			t.Fatalf("failed to read settings.json: %v", err)
		}

		var settings map[string]interface{}
		if err := json.Unmarshal(data, &settings); err != nil {
			t.Fatalf("failed to parse settings.json: %v", err)
		}

		plugins, ok := settings["enabledPlugins"].([]interface{})
		if !ok {
			t.Fatal("enabledPlugins not found or wrong type")
		}
		if len(plugins) != 1 {
			t.Errorf("expected 1 plugin, got %d", len(plugins))
		}
	})

	t.Run("creates sentinel file", func(t *testing.T) {
		stateDir := t.TempDir()
		p := &profile.Profile{Name: "test"}

		if err := BootstrapFromProfile(p, stateDir); err != nil {
			t.Fatalf("BootstrapFromProfile failed: %v", err)
		}

		if IsFirstRun(stateDir) {
			t.Error("should not be first run after bootstrap")
		}
	})

	t.Run("empty profile still creates sentinel", func(t *testing.T) {
		stateDir := t.TempDir()
		p := &profile.Profile{Name: "empty"}

		if err := BootstrapFromProfile(p, stateDir); err != nil {
			t.Fatalf("BootstrapFromProfile failed: %v", err)
		}

		if IsFirstRun(stateDir) {
			t.Error("should not be first run after bootstrap")
		}
	})

	t.Run("sync updates plugins while preserving other settings", func(t *testing.T) {
		stateDir := t.TempDir()

		// First bootstrap with plugin1
		p1 := &profile.Profile{
			Name:    "test",
			Plugins: []string{"plugin1@marketplace"},
		}
		if err := BootstrapFromProfile(p1, stateDir); err != nil {
			t.Fatalf("first bootstrap failed: %v", err)
		}

		// Simulate user adding custom settings
		settingsPath := filepath.Join(stateDir, "settings.json")
		customSettings := map[string]interface{}{
			"enabledPlugins": []string{"plugin1@marketplace"},
			"theme":          "dark",
			"fontSize":       14,
		}
		customData, _ := json.MarshalIndent(customSettings, "", "  ")
		if err := os.WriteFile(settingsPath, customData, 0644); err != nil {
			t.Fatalf("failed to write custom settings: %v", err)
		}

		// Re-bootstrap with plugin2 (simulating --sync)
		p2 := &profile.Profile{
			Name:    "test",
			Plugins: []string{"plugin2@marketplace"},
		}
		if err := BootstrapFromProfile(p2, stateDir); err != nil {
			t.Fatalf("sync bootstrap failed: %v", err)
		}

		// Verify settings updated but customizations preserved
		data, err := os.ReadFile(settingsPath)
		if err != nil {
			t.Fatalf("failed to read settings.json: %v", err)
		}

		var settings map[string]interface{}
		if err := json.Unmarshal(data, &settings); err != nil {
			t.Fatalf("failed to parse settings.json: %v", err)
		}

		// Check plugins updated
		plugins, ok := settings["enabledPlugins"].([]interface{})
		if !ok {
			t.Fatal("enabledPlugins not found or wrong type")
		}
		if len(plugins) != 1 || plugins[0] != "plugin2@marketplace" {
			t.Errorf("expected plugin2@marketplace, got %v", plugins)
		}

		// Check custom settings preserved
		if settings["theme"] != "dark" {
			t.Errorf("expected theme=dark to be preserved, got %v", settings["theme"])
		}
		if settings["fontSize"] != float64(14) {
			t.Errorf("expected fontSize=14 to be preserved, got %v", settings["fontSize"])
		}
	})
}
