// ABOUTME: Centralized path resolution for claudeup directories
// ABOUTME: Respects CLAUDEUP_HOME and CLAUDE_CONFIG_DIR environment variables for isolation

package config

import (
	"os"
	"path/filepath"
	"strings"
)

// MustClaudeupHome returns the claudeup home directory.
// Checks CLAUDEUP_HOME env var first, falls back to ~/.claudeup.
// Panics if CLAUDEUP_HOME is set but invalid (whitespace-only or relative path).
// Panics if home directory cannot be determined.
func MustClaudeupHome() string {
	if home := os.Getenv("CLAUDEUP_HOME"); home != "" {
		home = strings.TrimSpace(home)
		if home == "" {
			panic("CLAUDEUP_HOME is set but contains only whitespace")
		}
		if !filepath.IsAbs(home) {
			panic("CLAUDEUP_HOME must be an absolute path: " + home)
		}
		return home
	}
	homeDir, err := os.UserHomeDir()
	if err != nil {
		panic("cannot determine home directory: " + err.Error())
	}
	return filepath.Join(homeDir, ".claudeup")
}

// MustClaudeDir returns the Claude configuration directory.
// Checks CLAUDE_CONFIG_DIR env var first, falls back to ~/.claude.
// Panics if home directory cannot be determined.
func MustClaudeDir() string {
	if dir := os.Getenv("CLAUDE_CONFIG_DIR"); dir != "" {
		return dir
	}
	homeDir, err := os.UserHomeDir()
	if err != nil {
		panic("cannot determine home directory: " + err.Error())
	}
	return filepath.Join(homeDir, ".claude")
}
