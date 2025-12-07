// ABOUTME: Root command and CLI initialization for claudeup
// ABOUTME: Sets up cobra command structure and global flags
package commands

import (
	"os"
	"path/filepath"

	"github.com/claudeup/claudeup/internal/config"
	"github.com/spf13/cobra"
)

var (
	claudeDir string
)

var rootCmd = &cobra.Command{
	Use:   "claudeup",
	Short: "Manage Claude Code plugins, marketplaces, and MCP servers",
	Long: `claudeup is a comprehensive CLI tool for managing Claude Code installations.

It provides visibility into and control over:
  - Installed plugins and their state
  - Marketplace repositories
  - MCP server configuration
  - Plugin updates and maintenance`,
}

func Execute() error {
	return rootCmd.Execute()
}

// SetVersion sets the version for the root command
func SetVersion(version string) {
	rootCmd.Version = version
}

func init() {
	cobra.OnInitialize(initConfig)

	// Global flags - respect CLAUDE_CONFIG_DIR if set
	defaultClaudeDir := os.Getenv("CLAUDE_CONFIG_DIR")
	if defaultClaudeDir == "" {
		homeDir, _ := os.UserHomeDir()
		defaultClaudeDir = filepath.Join(homeDir, ".claude")
	}

	rootCmd.PersistentFlags().StringVar(&claudeDir, "claude-dir", defaultClaudeDir, "Claude installation directory")
	rootCmd.PersistentFlags().BoolVarP(&config.YesFlag, "yes", "y", false, "Skip all prompts, use defaults")
}

func initConfig() {
	// Initialize configuration
	// This will be called before any command runs
}
