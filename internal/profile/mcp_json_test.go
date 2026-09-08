package profile

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

func TestWriteMCPJSON(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "claudeup-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	servers := []MCPServer{
		{
			Name:    "filesystem",
			Command: "npx",
			Args:    []string{"-y", "@anthropic-ai/mcp-server-filesystem", "."},
		},
		{
			Name:    "github",
			Command: "npx",
			Args:    []string{"-y", "@anthropic-ai/mcp-server-github"},
			Secrets: map[string]SecretRef{
				"GITHUB_TOKEN": {
					Description: "GitHub personal access token",
					Sources:     []SecretSource{{Type: "env", Key: "GITHUB_TOKEN"}},
				},
			},
		},
	}

	if err := WriteMCPJSON(tempDir, servers); err != nil {
		t.Fatalf("WriteMCPJSON failed: %v", err)
	}

	// Verify file exists
	path := filepath.Join(tempDir, MCPConfigFile)
	if _, err := os.Stat(path); errors.Is(err, fs.ErrNotExist) {
		t.Fatal(".mcp.json was not created")
	}

	// Read and parse
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}

	var cfg MCPJSONConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	// Verify filesystem server
	fs, ok := cfg.MCPServers["filesystem"]
	if !ok {
		t.Fatal("filesystem server not found")
	}
	if fs.Command != "npx" {
		t.Errorf("filesystem.Command = %q, want %q", fs.Command, "npx")
	}
	if len(fs.Args) != 3 {
		t.Errorf("len(filesystem.Args) = %d, want 3", len(fs.Args))
	}
	if fs.Env != nil {
		t.Error("filesystem.Env should be nil (no secrets)")
	}

	// Verify github server with secret
	gh, ok := cfg.MCPServers["github"]
	if !ok {
		t.Fatal("github server not found")
	}
	if gh.Env == nil {
		t.Fatal("github.Env should not be nil")
	}
	if gh.Env["GITHUB_TOKEN"] != "${GITHUB_TOKEN}" {
		t.Errorf("GITHUB_TOKEN = %q, want %q", gh.Env["GITHUB_TOKEN"], "${GITHUB_TOKEN}")
	}
}

// Claude Code expands ${VAR} in .mcp.json args but leaves a bare $VAR alone,
// so secret references in args must be written in the braced form.
func TestWriteMCPJSON_ArgPlaceholders(t *testing.T) {
	tempDir := t.TempDir()

	servers := []MCPServer{
		{
			Name:    "api",
			Command: "npx",
			Args:    []string{"-y", "@my/mcp", "--token", "$API_TOKEN", "${ALREADY_BRACED}", "$", "literal$notref"},
			Secrets: map[string]SecretRef{
				"API_TOKEN": {Sources: []SecretSource{{Type: "env", Key: "API_TOKEN"}}},
			},
		},
	}

	if err := WriteMCPJSON(tempDir, servers); err != nil {
		t.Fatalf("WriteMCPJSON failed: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(tempDir, MCPConfigFile))
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}
	var cfg MCPJSONConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	want := []string{"-y", "@my/mcp", "--token", "${API_TOKEN}", "${ALREADY_BRACED}", "$", "literal$notref"}
	got := cfg.MCPServers["api"].Args
	if len(got) != len(want) {
		t.Fatalf("args = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("arg %d = %q, want %q", i, got[i], want[i])
		}
	}

	// The caller's slice must not be rewritten in place.
	if servers[0].Args[3] != "$API_TOKEN" {
		t.Errorf("WriteMCPJSON mutated caller args: %v", servers[0].Args)
	}
}

func TestWriteMCPJSON_EmptyServers(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "claudeup-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	if err := WriteMCPJSON(tempDir, []MCPServer{}); err != nil {
		t.Fatalf("WriteMCPJSON failed: %v", err)
	}

	// Read and verify directly
	path := filepath.Join(tempDir, MCPConfigFile)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}

	var cfg MCPJSONConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	if len(cfg.MCPServers) != 0 {
		t.Errorf("len(MCPServers) = %d, want 0", len(cfg.MCPServers))
	}
}

func TestMCPJSONExists(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "claudeup-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Should not exist initially
	if MCPJSONExists(tempDir) {
		t.Error("MCPJSONExists should return false for empty directory")
	}

	// Create file
	if err := WriteMCPJSON(tempDir, []MCPServer{}); err != nil {
		t.Fatalf("WriteMCPJSON failed: %v", err)
	}

	// Should exist now
	if !MCPJSONExists(tempDir) {
		t.Error("MCPJSONExists should return true after writing")
	}
}
