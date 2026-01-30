// ABOUTME: Writes .mcp.json files in Claude Code's native format
// ABOUTME: Enables project-scoped MCP servers that Claude auto-loads
package profile

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/claudeup/claudeup/v4/internal/events"
)

// MCPConfigFile is the filename for Claude's native MCP configuration
const MCPConfigFile = ".mcp.json"

// MCPJSONConfig represents Claude Code's native .mcp.json format
type MCPJSONConfig struct {
	MCPServers map[string]MCPJSONServer `json:"mcpServers"`
}

// MCPJSONServer represents an MCP server in Claude's .mcp.json format
type MCPJSONServer struct {
	Command string            `json:"command"`
	Args    []string          `json:"args,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
}

// WriteMCPJSON writes a .mcp.json file to the given directory
func WriteMCPJSON(projectDir string, servers []MCPServer) error {
	cfg := MCPJSONConfig{
		MCPServers: make(map[string]MCPJSONServer),
	}

	for _, s := range servers {
		env := make(map[string]string)

		// Convert secrets to ${VAR} format for .mcp.json
		// Claude Code expands these at runtime
		for envVar := range s.Secrets {
			env[envVar] = "${" + envVar + "}"
		}

		// Only include env if there are entries
		var envPtr map[string]string
		if len(env) > 0 {
			envPtr = env
		}

		cfg.MCPServers[s.Name] = MCPJSONServer{
			Command: s.Command,
			Args:    s.Args,
			Env:     envPtr,
		}
	}

	path := filepath.Join(projectDir, MCPConfigFile)
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	// Add trailing newline
	data = append(data, '\n')

	// Wrap file write with event tracking
	return events.GlobalTracker().RecordFileWrite(
		"mcp config write",
		path,
		"project",
		func() error {
			return os.WriteFile(path, data, 0644)
		},
	)
}

// MCPJSONExists returns true if a .mcp.json file exists in the directory
func MCPJSONExists(projectDir string) bool {
	path := filepath.Join(projectDir, MCPConfigFile)
	_, err := os.Stat(path)
	return err == nil
}
