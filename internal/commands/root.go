// ABOUTME: Root command and CLI initialization for claudeup
// ABOUTME: Sets up cobra command structure and global flags
package commands

import (
	"github.com/claudeup/claudeup/v5/internal/config"
	"github.com/claudeup/claudeup/v5/internal/ui"
	"github.com/spf13/cobra"
)

var (
	claudeDir    string
	claudeupHome string
)

var rootCmd = &cobra.Command{
	Use:   "claudeup",
	Short: "Manage Claude Code profiles and configurations",
	Long: `claudeup is a CLI tool for managing Claude Code installations.

It provides visibility into and control over:
  - Profile management and configuration
  - Installed plugins and their state
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

	// Set up custom help template with lipgloss styling
	ui.SetupHelpTemplate(rootCmd)

	// Global flags - respect CLAUDE_CONFIG_DIR if set
	rootCmd.PersistentFlags().StringVar(&claudeDir, "claude-dir", config.MustClaudeDir(), "Claude installation directory")
	rootCmd.PersistentFlags().StringVar(&claudeupHome, "claudeup-home", config.MustClaudeupHome(), "claudeup home directory")
	rootCmd.PersistentFlags().BoolVarP(&config.YesFlag, "yes", "y", false, "Skip all prompts, use defaults")
}

func initConfig() {
	// Initialize configuration
	// This will be called before any command runs
}
