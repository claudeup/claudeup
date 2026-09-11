// ABOUTME: Reads a plugin's own .claude-plugin/plugin.json manifest
// ABOUTME: Only the fields claudeup acts on are retained
package claude

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// PluginManifest is the .claude-plugin/plugin.json a plugin carries. Only the
// fields claudeup acts on are retained. The version matters because Claude Code
// records it in preference to the version the marketplace entry declares, and
// falls back to the marketplace's only when the manifest sets none.
type PluginManifest struct {
	Name    string `json:"name"`
	Version string `json:"version,omitempty"`
}

// LoadPluginManifest reads the manifest of the plugin whose content is in
// pluginDir. A plugin need not carry one, so an absent manifest yields nil with
// no error; the marketplace entry then describes the plugin. A manifest that is
// present but cannot be read or parsed is an error, because anything decided
// from its absence would be wrong.
func LoadPluginManifest(pluginDir string) (*PluginManifest, error) {
	manifestPath := filepath.Join(pluginDir, ".claude-plugin", "plugin.json")

	data, err := os.ReadFile(manifestPath)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("cannot read plugin manifest %s: %w", manifestPath, err)
	}

	var manifest PluginManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("failed to parse plugin manifest %s: %w", manifestPath, err)
	}
	return &manifest, nil
}
