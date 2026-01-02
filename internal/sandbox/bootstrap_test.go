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
}
