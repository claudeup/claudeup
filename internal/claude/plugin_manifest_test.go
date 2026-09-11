// ABOUTME: Unit tests for reading a plugin's .claude-plugin/plugin.json
// ABOUTME: Covers a present, absent, malformed and unreadable manifest
package claude

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeManifest(t *testing.T, pluginDir, content string) {
	t.Helper()
	manifestDir := filepath.Join(pluginDir, ".claude-plugin")
	if err := os.MkdirAll(manifestDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(manifestDir, "plugin.json"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestLoadPluginManifestReadsNameAndVersion(t *testing.T) {
	pluginDir := t.TempDir()
	writeManifest(t, pluginDir, `{"name":"hookify","version":"1.2.3","description":"ignored"}`)

	manifest, err := LoadPluginManifest(pluginDir)
	if err != nil {
		t.Fatalf("LoadPluginManifest returned error: %v", err)
	}
	if manifest == nil {
		t.Fatal("expected a manifest, got nil")
	}
	if manifest.Name != "hookify" {
		t.Errorf("expected name hookify, got %q", manifest.Name)
	}
	if manifest.Version != "1.2.3" {
		t.Errorf("expected version 1.2.3, got %q", manifest.Version)
	}
}

func TestLoadPluginManifestAbsentIsNotAnError(t *testing.T) {
	// A plugin need not carry a manifest; the marketplace entry then describes it.
	manifest, err := LoadPluginManifest(t.TempDir())
	if err != nil {
		t.Fatalf("an absent manifest is not an error, got: %v", err)
	}
	if manifest != nil {
		t.Errorf("expected nil for an absent manifest, got %+v", manifest)
	}
}

func TestLoadPluginManifestVersionUnsetIsEmpty(t *testing.T) {
	pluginDir := t.TempDir()
	writeManifest(t, pluginDir, `{"name":"hookify"}`)

	manifest, err := LoadPluginManifest(pluginDir)
	if err != nil {
		t.Fatalf("LoadPluginManifest returned error: %v", err)
	}
	if manifest == nil {
		t.Fatal("expected a manifest, got nil")
	}
	if manifest.Version != "" {
		t.Errorf("expected an empty version, got %q", manifest.Version)
	}
}

func TestLoadPluginManifestMalformed(t *testing.T) {
	pluginDir := t.TempDir()
	writeManifest(t, pluginDir, `{"name":`)

	_, err := LoadPluginManifest(pluginDir)
	if err == nil {
		t.Fatal("expected an error for a malformed manifest")
	}
	if !strings.Contains(err.Error(), "plugin.json") {
		t.Errorf("error should name the file, got: %v", err)
	}
}

func TestLoadPluginManifestUnreadable(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("permission checks do not apply when running as root")
	}
	pluginDir := t.TempDir()
	writeManifest(t, pluginDir, `{"name":"hookify","version":"1.0.0"}`)
	manifestPath := filepath.Join(pluginDir, ".claude-plugin", "plugin.json")
	if err := os.Chmod(manifestPath, 0000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(manifestPath, 0644) })

	_, err := LoadPluginManifest(pluginDir)
	if err == nil {
		t.Fatal("expected an error for an unreadable manifest")
	}
	if !strings.Contains(err.Error(), "plugin.json") {
		t.Errorf("error should name the file, got: %v", err)
	}
}
