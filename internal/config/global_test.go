// ABOUTME: Unit tests for global configuration management
// ABOUTME: Tests config loading, saving, and preferences
package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg == nil {
		t.Fatal("DefaultConfig should not return nil")
	}
}

func TestSaveAndLoad(t *testing.T) {
	// Create temp directory for test
	tempDir, err := os.MkdirTemp("", "claudeup-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create config
	cfg := DefaultConfig()

	// Save to temp file
	configFile := filepath.Join(tempDir, "config.json")
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(configFile, data, 0644); err != nil {
		t.Fatal(err)
	}

	// Load it back
	loadedData, err := os.ReadFile(configFile)
	if err != nil {
		t.Fatal(err)
	}

	var loadedCfg GlobalConfig
	if err := json.Unmarshal(loadedData, &loadedCfg); err != nil {
		t.Fatal(err)
	}
}
