// ABOUTME: Unit tests for marketplace registry management
// ABOUTME: Tests loading and marketplace operations
package claude

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadMarketplacesNonExistent(t *testing.T) {
	// A non-existent plugins directory is treated as a fresh install (no marketplaces yet).
	// The stat check is intentionally absent; ReadFile handles the missing path.
	registry, err := LoadMarketplaces("/non/existent/path")
	if err != nil {
		t.Errorf("LoadMarketplaces should return empty registry for non-existent path, got error: %v", err)
	}
	if registry == nil {
		t.Error("Registry should be initialized, not nil")
	}
	if len(registry) != 0 {
		t.Errorf("Expected 0 marketplaces for non-existent path, got %d", len(registry))
	}
}

func TestLoadMarketplacesFreshInstall(t *testing.T) {
	// Create temp directory with plugins subdirectory but no known_marketplaces.json
	// This simulates a fresh Claude Code install that hasn't added any marketplaces yet
	tempDir, err := os.MkdirTemp("", "claudeup-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create plugins directory (Claude creates this on install)
	pluginsDir := filepath.Join(tempDir, "plugins")
	if err := os.MkdirAll(pluginsDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Load marketplaces from fresh install (no known_marketplaces.json yet)
	registry, err := LoadMarketplaces(tempDir)
	if err != nil {
		t.Fatalf("LoadMarketplaces should not error on fresh install, got: %v", err)
	}

	// Should return empty registry
	if registry == nil {
		t.Error("Registry should be initialized, not nil")
	}

	if len(registry) != 0 {
		t.Errorf("Expected 0 marketplaces in fresh install, got %d", len(registry))
	}
}

func TestLoadMarketplacesPermissionError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("permission checks do not apply when running as root")
	}
	tempDir := t.TempDir()
	pluginsDir := filepath.Join(tempDir, "plugins")
	if err := os.MkdirAll(pluginsDir, 0755); err != nil {
		t.Fatal(err)
	}
	// Create an unreadable known_marketplaces.json
	marketplacesPath := filepath.Join(pluginsDir, "known_marketplaces.json")
	if err := os.WriteFile(marketplacesPath, []byte(`{}`), 0000); err != nil {
		t.Fatal(err)
	}

	_, err := LoadMarketplaces(tempDir)
	if err == nil {
		t.Fatal("LoadMarketplaces should return error for unreadable file")
	}
	if !strings.Contains(err.Error(), "cannot read marketplaces from") {
		t.Errorf("expected context-wrapped error, got: %v", err)
	}
	// Should NOT be a not-found error -- it's a permission error
	if errors.Is(err, fs.ErrNotExist) {
		t.Errorf("expected permission error, not ErrNotExist, got: %v", err)
	}
}

func TestLoadMarketplacesCorruptJSON(t *testing.T) {
	tempDir := t.TempDir()
	pluginsDir := filepath.Join(tempDir, "plugins")
	if err := os.MkdirAll(pluginsDir, 0755); err != nil {
		t.Fatal(err)
	}
	marketplacesPath := filepath.Join(pluginsDir, "known_marketplaces.json")
	if err := os.WriteFile(marketplacesPath, []byte(`{not valid json`), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := LoadMarketplaces(tempDir)
	if err == nil {
		t.Fatal("LoadMarketplaces should return error for corrupt JSON")
	}
	if !strings.Contains(err.Error(), "failed to parse marketplaces JSON from") {
		t.Errorf("expected parse error with path context, got: %v", err)
	}
}

func TestMarketplaceRegistryJSONMarshaling(t *testing.T) {
	registry := MarketplaceRegistry{
		"marketplace-1": MarketplaceMetadata{
			Source: MarketplaceSource{
				Source: "github",
				Repo:   "org/repo1",
			},
			InstallLocation: "/path/1",
			LastUpdated:     "2024-01-01T00:00:00Z",
		},
		"marketplace-2": MarketplaceMetadata{
			Source: MarketplaceSource{
				Source: "git",
				Repo:   "org/repo2",
			},
			InstallLocation: "/path/2",
			LastUpdated:     "2024-01-02T00:00:00Z",
		},
	}

	// Marshal to JSON
	data, err := json.MarshalIndent(registry, "", "  ")
	if err != nil {
		t.Fatal(err)
	}

	// Unmarshal from JSON
	var loaded MarketplaceRegistry
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatal(err)
	}

	// Verify data integrity
	if len(loaded) != len(registry) {
		t.Error("Marketplace count mismatch after JSON round-trip")
	}

	m1 := loaded["marketplace-1"]
	if m1.Source.Repo != "org/repo1" {
		t.Error("Marketplace-1 repo mismatch after JSON round-trip")
	}

	m2 := loaded["marketplace-2"]
	if m2.Source.Repo != "org/repo2" {
		t.Error("Marketplace-2 repo mismatch after JSON round-trip")
	}
}

func TestMarketplaceExists(t *testing.T) {
	registry := MarketplaceRegistry{
		"claude-mem": MarketplaceMetadata{
			Source: MarketplaceSource{
				Source: "github",
				Repo:   "thedotmack/claude-mem",
			},
		},
		"superpowers": MarketplaceMetadata{
			Source: MarketplaceSource{
				Source: "git",
				URL:    "https://github.com/obra/superpowers-marketplace.git",
			},
		},
	}

	// Test exists by repo
	if !registry.MarketplaceExists("thedotmack/claude-mem") {
		t.Error("Should find marketplace by repo")
	}

	// Test exists by URL
	if !registry.MarketplaceExists("https://github.com/obra/superpowers-marketplace.git") {
		t.Error("Should find marketplace by URL")
	}

	// Test not found
	if registry.MarketplaceExists("nonexistent/repo") {
		t.Error("Should not find nonexistent marketplace")
	}
}

func TestPluginSourceUnmarshalJSON(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantSource string
		wantURL    string
		wantErr    bool
	}{
		{
			name:       "string source (relative path)",
			input:      `"./plugins/hookify"`,
			wantSource: "./plugins/hookify",
			wantURL:    "",
		},
		{
			name:       "object source with url",
			input:      `{"source":"git","url":"https://github.com/org/repo"}`,
			wantSource: "git",
			wantURL:    "https://github.com/org/repo",
		},
		{
			name:       "object source without url",
			input:      `{"source":"./local/path"}`,
			wantSource: "./local/path",
			wantURL:    "",
		},
		{
			name:    "invalid JSON (number)",
			input:   `123`,
			wantErr: true,
		},
		{
			name:    "invalid JSON (array)",
			input:   `[1,2,3]`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var ps PluginSource
			err := json.Unmarshal([]byte(tt.input), &ps)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error for input %s, got nil", tt.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if ps.Source != tt.wantSource {
				t.Errorf("Source = %q, want %q", ps.Source, tt.wantSource)
			}
			if ps.URL != tt.wantURL {
				t.Errorf("URL = %q, want %q", ps.URL, tt.wantURL)
			}
		})
	}
}

func TestPluginSourceIsRelativePath(t *testing.T) {
	tests := []struct {
		name   string
		source PluginSource
		want   bool
	}{
		{"relative path", PluginSource{Source: "./plugins/hookify"}, true},
		{"url source", PluginSource{Source: "git", URL: "https://github.com/org/repo"}, false},
		{"empty source", PluginSource{}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.source.IsRelativePath(); got != tt.want {
				t.Errorf("IsRelativePath() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPluginSourceIsURL(t *testing.T) {
	tests := []struct {
		name   string
		source PluginSource
		want   bool
	}{
		{"url source", PluginSource{Source: "git", URL: "https://github.com/org/repo"}, true},
		{"relative path", PluginSource{Source: "./plugins/hookify"}, false},
		{"empty source", PluginSource{}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.source.IsURL(); got != tt.want {
				t.Errorf("IsURL() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetMarketplaceByRepo(t *testing.T) {
	registry := MarketplaceRegistry{
		"claude-mem": MarketplaceMetadata{
			Source: MarketplaceSource{
				Source: "github",
				Repo:   "thedotmack/claude-mem",
			},
		},
	}

	// Test found
	name := registry.GetMarketplaceByRepo("thedotmack/claude-mem")
	if name != "claude-mem" {
		t.Errorf("Expected 'claude-mem', got '%s'", name)
	}

	// Test not found
	name = registry.GetMarketplaceByRepo("nonexistent/repo")
	if name != "" {
		t.Errorf("Expected empty string for not found, got '%s'", name)
	}
}
