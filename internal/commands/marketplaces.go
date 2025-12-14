// ABOUTME: Marketplace command implementation for managing marketplaces
// ABOUTME: Shows detailed information about Claude Code marketplace repositories
package commands

import (
	"fmt"
	"sort"

	"github.com/claudeup/claudeup/internal/claude"
	"github.com/claudeup/claudeup/internal/ui"
	"github.com/spf13/cobra"
)

var marketplaceCmd = &cobra.Command{
	Use:   "marketplace",
	Short: "Manage Claude Code marketplaces",
	Long:  `Marketplaces are repositories containing Claude Code plugins.`,
}

var marketplaceListCmd = &cobra.Command{
	Use:   "list",
	Short: "List installed marketplaces",
	Long: `Display information about installed Claude Code marketplace repositories.

Shows each marketplace's source, repository, install location, and last update time.`,
	Args: cobra.NoArgs,
	RunE: runMarketplaceList,
}

func init() {
	rootCmd.AddCommand(marketplaceCmd)
	marketplaceCmd.AddCommand(marketplaceListCmd)
}

func runMarketplaceList(cmd *cobra.Command, args []string) error {
	// Load marketplaces
	marketplaces, err := claude.LoadMarketplaces(claudeDir)
	if err != nil {
		return fmt.Errorf("failed to load marketplaces: %w", err)
	}

	// Sort marketplace names for consistent output
	names := make([]string, 0, len(marketplaces))
	for name := range marketplaces {
		names = append(names, name)
	}
	sort.Strings(names)

	// Print header
	fmt.Println(ui.RenderSection("Installed Marketplaces", len(names)))
	fmt.Println()

	// Print each marketplace
	for _, name := range names {
		marketplace := marketplaces[name]

		fmt.Printf("%s %s\n", ui.Success(ui.SymbolSuccess), ui.Bold(name))
		fmt.Println(ui.Indent(ui.RenderDetail("Source", marketplace.Source.Source), 1))
		fmt.Println(ui.Indent(ui.RenderDetail("Repo", marketplace.Source.Repo), 1))
		fmt.Println(ui.Indent(ui.RenderDetail("Location", ui.Muted(marketplace.InstallLocation)), 1))
		fmt.Println(ui.Indent(ui.RenderDetail("Updated", ui.Muted(marketplace.LastUpdated)), 1))
		fmt.Println()
	}

	return nil
}
