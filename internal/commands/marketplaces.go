// ABOUTME: Marketplaces command implementation for listing installed marketplaces
// ABOUTME: Shows detailed information about Claude Code marketplace repositories
package commands

import (
	"fmt"
	"sort"

	"github.com/claudeup/claudeup/internal/claude"
	"github.com/spf13/cobra"
)

var marketplacesCmd = &cobra.Command{
	Use:   "marketplaces",
	Short: "List installed marketplaces",
	Long:  `Display information about installed Claude Code marketplace repositories.`,
	RunE:  runMarketplaces,
}

func init() {
	rootCmd.AddCommand(marketplacesCmd)
}

func runMarketplaces(cmd *cobra.Command, args []string) error {
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
	fmt.Printf("=== Installed Marketplaces (%d) ===\n\n", len(names))

	// Print each marketplace
	for _, name := range names {
		marketplace := marketplaces[name]

		fmt.Printf("✓ %s\n", name)
		fmt.Printf("   Source:     %s\n", marketplace.Source.Source)
		fmt.Printf("   Repo:       %s\n", marketplace.Source.Repo)
		fmt.Printf("   Location:   %s\n", marketplace.InstallLocation)
		fmt.Printf("   Updated:    %s\n", marketplace.LastUpdated)
		fmt.Println()
	}

	return nil
}
