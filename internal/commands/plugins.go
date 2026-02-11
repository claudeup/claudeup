// ABOUTME: Backward-compatible alias for plugin list command
// ABOUTME: Kept for users accustomed to the old command name
package commands

import (
	"github.com/spf13/cobra"
)

var pluginsCmd = &cobra.Command{
	Use:        "plugins",
	Short:      "List installed plugins (alias for 'plugin list')",
	Long:       `This command is an alias for 'claudeup plugin list'.`,
	Args:       cobra.NoArgs,
	Deprecated: "use 'claudeup plugin list' instead",
	RunE:       runPluginList,
}

func init() {
	rootCmd.AddCommand(pluginsCmd)
	pluginsCmd.Flags().BoolVar(&pluginListSummary, "summary", false, "Show only summary statistics")
}
