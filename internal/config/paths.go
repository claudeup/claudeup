// ABOUTME: Centralized path resolution for claudeup directories
// ABOUTME: Respects CLAUDEUP_HOME environment variable for isolation

package config

import (
	"os"
	"path/filepath"
)

// MustClaudeupHome returns the claudeup home directory.
// Checks CLAUDEUP_HOME env var first, falls back to ~/.claudeup.
// Panics if home directory cannot be determined.
func MustClaudeupHome() string {
	if home := os.Getenv("CLAUDEUP_HOME"); home != "" {
		return home
	}
	homeDir, err := os.UserHomeDir()
	if err != nil {
		panic("cannot determine home directory: " + err.Error())
	}
	return filepath.Join(homeDir, ".claudeup")
}
