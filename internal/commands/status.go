// ABOUTME: Status command implementation showing overview of Claude installation
// ABOUTME: Displays marketplaces, plugins, MCP servers, and detected issues
package commands

import (
	"fmt"
	"strings"

	"github.com/claudeup/claudeup/internal/claude"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show overview of Claude Code installation",
	Long:  `Display status of marketplaces, plugins, MCP servers, and any detected issues.`,
	RunE:  runStatus,
}

func init() {
	rootCmd.AddCommand(statusCmd)
}

func runStatus(cmd *cobra.Command, args []string) error {
	// Load marketplaces
	marketplaces, err := claude.LoadMarketplaces(claudeDir)
	if err != nil {
		return fmt.Errorf("failed to load marketplaces: %w", err)
	}

	// Load plugins
	plugins, err := claude.LoadPlugins(claudeDir)
	if err != nil {
		return fmt.Errorf("failed to load plugins: %w", err)
	}

	// Print header
	printHeader("claudeup Status")

	// Print marketplaces
	fmt.Println("\nMarketplaces (" + fmt.Sprint(len(marketplaces)) + ")")
	for name := range marketplaces {
		fmt.Printf("  ✓ %s\n", name)
	}

	// Count enabled/disabled plugins and detect issues
	enabledCount := 0
	disabledPlugins := []string{}
	stalePlugins := []string{}

	for name, plugin := range plugins.Plugins {
		if plugin.PathExists() {
			enabledCount++
		} else {
			stalePlugins = append(stalePlugins, name)
		}
	}

	// Print plugins summary
	fmt.Printf("\nPlugins (%d total)\n", len(plugins.Plugins))
	fmt.Printf("  ✓ %d enabled\n", enabledCount)
	if len(disabledPlugins) > 0 {
		fmt.Printf("  ✗ %d disabled\n", len(disabledPlugins))
		for _, name := range disabledPlugins {
			fmt.Printf("    - %s\n", name)
		}
	}

	// Print MCP servers placeholder
	fmt.Println("\nMCP Servers")
	fmt.Println("  → Run 'claudeup mcp list' for details")

	// Print issues if any
	if len(stalePlugins) > 0 {
		fmt.Println("\nIssues Detected")
		fmt.Printf("  ⚠ %d plugins have stale paths\n", len(stalePlugins))
		for _, name := range stalePlugins {
			fmt.Printf("    - %s\n", name)
		}
		fmt.Println("  → Run 'claudeup doctor' for details")
	}

	return nil
}

func printHeader(title string) {
	width := 40
	border := "═"
	padding := (width - len(title) - 2) / 2

	fmt.Println("╔" + strings.Repeat(border, width) + "╗")
	fmt.Printf("║%s%s%s║\n",
		strings.Repeat(" ", padding),
		title,
		strings.Repeat(" ", width-padding-len(title)))
	fmt.Println("╚" + strings.Repeat(border, width) + "╝")
}
