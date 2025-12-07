// ABOUTME: Profile subcommands for managing Claude Code profiles
// ABOUTME: Implements list, use, create, and show operations
package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/claudeup/claudeup/internal/config"
	"github.com/claudeup/claudeup/internal/profile"
	"github.com/spf13/cobra"
)

var profileCmd = &cobra.Command{
	Use:   "profile",
	Short: "Manage Claude Code configuration profiles",
	Long: `Profiles are saved configurations of plugins, MCP servers, and marketplaces.

Use profiles to:
  - Save your current setup for later
  - Switch between different configurations
  - Share configurations between machines`,
}

var profileListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available profiles",
	RunE:  runProfileList,
}

var profileUseCmd = &cobra.Command{
	Use:   "use <name>",
	Short: "Apply a profile to Claude Code",
	Args:  cobra.ExactArgs(1),
	RunE:  runProfileUse,
}

var profileCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a profile from current state",
	Args:  cobra.ExactArgs(1),
	RunE:  runProfileCreate,
}

var profileShowCmd = &cobra.Command{
	Use:   "show <name>",
	Short: "Display a profile's contents",
	Args:  cobra.ExactArgs(1),
	RunE:  runProfileShow,
}

var profileSuggestCmd = &cobra.Command{
	Use:   "suggest",
	Short: "Suggest a profile based on current directory",
	RunE:  runProfileSuggest,
}

func init() {
	rootCmd.AddCommand(profileCmd)
	profileCmd.AddCommand(profileListCmd)
	profileCmd.AddCommand(profileUseCmd)
	profileCmd.AddCommand(profileCreateCmd)
	profileCmd.AddCommand(profileShowCmd)
	profileCmd.AddCommand(profileSuggestCmd)
}

func runProfileList(cmd *cobra.Command, args []string) error {
	profilesDir := getProfilesDir()

	profiles, err := profile.List(profilesDir)
	if err != nil {
		return fmt.Errorf("failed to list profiles: %w", err)
	}

	if len(profiles) == 0 {
		fmt.Println("No profiles found.")
		fmt.Println("Create one with: claudeup profile create <name>")
		return nil
	}

	// Get active profile from config
	cfg, _ := config.Load()
	activeProfile := ""
	if cfg != nil {
		activeProfile = cfg.Preferences.ActiveProfile
	}

	fmt.Println("Available profiles:")
	fmt.Println()

	for _, p := range profiles {
		marker := "  "
		if p.Name == activeProfile {
			marker = "* "
		}

		desc := p.Description
		if desc == "" {
			desc = "(no description)"
		}

		fmt.Printf("%s%-20s %s\n", marker, p.Name, desc)
	}

	fmt.Println()
	fmt.Println("Use 'claudeup profile show <name>' for details")
	fmt.Println("Use 'claudeup profile use <name>' to apply a profile")

	return nil
}

func runProfileUse(cmd *cobra.Command, args []string) error {
	name := args[0]
	profilesDir := getProfilesDir()

	// Load the profile
	p, err := profile.Load(profilesDir, name)
	if err != nil {
		return fmt.Errorf("profile %q not found: %w", name, err)
	}

	claudeDir := profile.DefaultClaudeDir()
	claudeJSONPath := profile.DefaultClaudeJSONPath()

	// Compute and show diff
	diff, err := profile.ComputeDiff(p, claudeDir, claudeJSONPath)
	if err != nil {
		return fmt.Errorf("failed to compute changes: %w", err)
	}

	if !hasDiffChanges(diff) {
		fmt.Println("No changes needed - profile already matches current state.")
		return nil
	}

	fmt.Printf("Profile: %s\n", name)
	fmt.Println()
	showDiff(diff)
	fmt.Println()

	if !confirmProceed() {
		fmt.Println("Cancelled.")
		return nil
	}

	// Apply
	fmt.Println()
	fmt.Println("Applying profile...")

	chain := buildSecretChain()
	result, err := profile.Apply(p, claudeDir, claudeJSONPath, chain)
	if err != nil {
		return fmt.Errorf("failed to apply profile: %w", err)
	}

	showApplyResults(result)

	// Update active profile in config
	cfg, err := config.Load()
	if err != nil {
		cfg = config.DefaultConfig()
	}
	cfg.Preferences.ActiveProfile = name
	if err := config.Save(cfg); err != nil {
		fmt.Printf("  ⚠ Could not save active profile: %v\n", err)
	}

	fmt.Println()
	fmt.Println("✓ Profile applied!")

	return nil
}

func runProfileCreate(cmd *cobra.Command, args []string) error {
	name := args[0]
	profilesDir := getProfilesDir()

	// Check if profile already exists
	existingPath := filepath.Join(profilesDir, name+".json")
	if _, err := os.Stat(existingPath); err == nil {
		if !config.YesFlag {
			fmt.Printf("Profile %q already exists. Overwrite? [y/N]: ", name)
			choice := promptChoice("", "n")
			if choice != "y" && choice != "yes" {
				fmt.Println("Cancelled.")
				return nil
			}
		}
	}

	claudeDir := profile.DefaultClaudeDir()
	claudeJSONPath := profile.DefaultClaudeJSONPath()

	// Create snapshot
	p, err := profile.Snapshot(name, claudeDir, claudeJSONPath)
	if err != nil {
		return fmt.Errorf("failed to snapshot current state: %w", err)
	}

	// Save
	if err := profile.Save(profilesDir, p); err != nil {
		return fmt.Errorf("failed to save profile: %w", err)
	}

	fmt.Printf("✓ Created profile %q\n", name)
	fmt.Println()
	fmt.Printf("  MCP Servers:   %d\n", len(p.MCPServers))
	fmt.Printf("  Marketplaces:  %d\n", len(p.Marketplaces))
	fmt.Printf("  Plugins:       %d\n", len(p.Plugins))

	return nil
}

func runProfileShow(cmd *cobra.Command, args []string) error {
	name := args[0]
	profilesDir := getProfilesDir()

	p, err := profile.Load(profilesDir, name)
	if err != nil {
		return fmt.Errorf("profile %q not found: %w", name, err)
	}

	fmt.Printf("Profile: %s\n", p.Name)
	if p.Description != "" {
		fmt.Printf("Description: %s\n", p.Description)
	}
	fmt.Println()

	if len(p.MCPServers) > 0 {
		fmt.Println("MCP Servers:")
		for _, m := range p.MCPServers {
			fmt.Printf("  - %s (%s)\n", m.Name, m.Command)
			if len(m.Secrets) > 0 {
				for envVar := range m.Secrets {
					fmt.Printf("      requires: %s\n", envVar)
				}
			}
		}
		fmt.Println()
	}

	if len(p.Marketplaces) > 0 {
		fmt.Println("Marketplaces:")
		for _, m := range p.Marketplaces {
			fmt.Printf("  - %s\n", m.Repo)
		}
		fmt.Println()
	}

	if len(p.Plugins) > 0 {
		fmt.Println("Plugins:")
		for _, plug := range p.Plugins {
			fmt.Printf("  - %s\n", plug)
		}
		fmt.Println()
	}

	return nil
}

func hasDiffChanges(diff *profile.Diff) bool {
	return len(diff.PluginsToRemove) > 0 ||
		len(diff.PluginsToInstall) > 0 ||
		len(diff.MCPToRemove) > 0 ||
		len(diff.MCPToInstall) > 0 ||
		len(diff.MarketplacesToAdd) > 0
}

func showDiff(diff *profile.Diff) {
	if len(diff.PluginsToRemove) > 0 || len(diff.MCPToRemove) > 0 {
		fmt.Println("  Remove:")
		for _, p := range diff.PluginsToRemove {
			fmt.Printf("    - %s\n", p)
		}
		for _, m := range diff.MCPToRemove {
			fmt.Printf("    - MCP: %s\n", m)
		}
	}

	if len(diff.PluginsToInstall) > 0 || len(diff.MCPToInstall) > 0 || len(diff.MarketplacesToAdd) > 0 {
		fmt.Println("  Install:")
		for _, m := range diff.MarketplacesToAdd {
			fmt.Printf("    + Marketplace: %s\n", m.Repo)
		}
		for _, p := range diff.PluginsToInstall {
			fmt.Printf("    + %s\n", p)
		}
		for _, m := range diff.MCPToInstall {
			secretInfo := ""
			if len(m.Secrets) > 0 {
				for k := range m.Secrets {
					secretInfo = fmt.Sprintf(" (requires %s)", k)
					break
				}
			}
			fmt.Printf("    + MCP: %s%s\n", m.Name, secretInfo)
		}
	}
}

func runProfileSuggest(cmd *cobra.Command, args []string) error {
	profilesDir := getProfilesDir()

	// Get current directory
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Load all profiles
	profiles, err := profile.List(profilesDir)
	if err != nil {
		return fmt.Errorf("failed to list profiles: %w", err)
	}

	if len(profiles) == 0 {
		fmt.Println("No profiles available.")
		fmt.Println("Create one with: claudeup profile create <name>")
		return nil
	}

	// Find matching profiles
	suggested := profile.SuggestProfile(cwd, profiles)

	if suggested == nil {
		fmt.Println("No profile matches the current directory.")
		fmt.Println()
		fmt.Println("Available profiles:")
		for _, p := range profiles {
			fmt.Printf("  - %s\n", p.Name)
		}
		return nil
	}

	fmt.Printf("Suggested profile: %s\n", suggested.Name)
	if suggested.Description != "" {
		fmt.Printf("  %s\n", suggested.Description)
	}
	fmt.Println()

	fmt.Print("Apply this profile? [Y/n]: ")
	choice := promptChoice("", "y")
	if choice == "y" || choice == "yes" || choice == "" {
		// Run the use command
		return runProfileUse(cmd, []string{suggested.Name})
	}

	fmt.Println("Cancelled.")
	return nil
}
