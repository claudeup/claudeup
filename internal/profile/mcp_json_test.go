package profile

import (
	"encoding/json"
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
	if _, err := os.Stat(path); os.IsNotExist(err) {
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

func TestWriteMCPJSON_EmptyServers(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "claudeup-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	if err := WriteMCPJSON(tempDir, []MCPServer{}); err != nil {
		t.Fatalf("WriteMCPJSON failed: %v", err)
	}

	cfg, err := LoadMCPJSON(tempDir)
	if err != nil {
		t.Fatalf("LoadMCPJSON failed: %v", err)
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

func TestLoadMCPJSON(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "claudeup-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Write a file
	servers := []MCPServer{
		{Name: "test", Command: "echo", Args: []string{"hello"}},
	}
	if err := WriteMCPJSON(tempDir, servers); err != nil {
		t.Fatalf("WriteMCPJSON failed: %v", err)
	}

	// Load it back
	cfg, err := LoadMCPJSON(tempDir)
	if err != nil {
		t.Fatalf("LoadMCPJSON failed: %v", err)
	}

	if len(cfg.MCPServers) != 1 {
		t.Errorf("len(MCPServers) = %d, want 1", len(cfg.MCPServers))
	}

	server, ok := cfg.MCPServers["test"]
	if !ok {
		t.Fatal("test server not found")
	}
	if server.Command != "echo" {
		t.Errorf("Command = %q, want %q", server.Command, "echo")
	}
}

func TestLoadMCPJSON_NotFound(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "claudeup-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	_, err = LoadMCPJSON(tempDir)
	if err == nil {
		t.Error("LoadMCPJSON should fail when file doesn't exist")
	}
}
