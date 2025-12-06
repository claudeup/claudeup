// ABOUTME: Root command and CLI initialization for claude-pm
// ABOUTME: Sets up cobra command structure and global flags
package commands

import (
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

const version = "0.2.0"

var (
	claudeDir string
)

var rootCmd = &cobra.Command{
	Use:     "claude-pm",
	Short:   "Manage Claude Code plugins, marketplaces, and MCP servers",
	Version: version,
	Long: `claude-pm is a comprehensive CLI tool for managing Claude Code installations.

It provides visibility into and control over:
  - Installed plugins and their state
  - Marketplace repositories
  - MCP server configuration
  - Plugin updates and maintenance`,
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	// Global flags
	homeDir, _ := os.UserHomeDir()
	defaultClaudeDir := filepath.Join(homeDir, ".claude")

	rootCmd.PersistentFlags().StringVar(&claudeDir, "claude-dir", defaultClaudeDir, "Claude installation directory")
}

func initConfig() {
	// Initialize configuration
	// This will be called before any command runs
}
