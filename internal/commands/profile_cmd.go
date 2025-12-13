// ABOUTME: Profile subcommands for managing Claude Code profiles
// ABOUTME: Implements list, use, save, and show operations
package commands

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/claudeup/claudeup/internal/claude"
	"github.com/claudeup/claudeup/internal/config"
	"github.com/claudeup/claudeup/internal/profile"
	"github.com/spf13/cobra"
)

var profileCreateFromFlag string

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

var profileSaveCmd = &cobra.Command{
	Use:   "save [name]",
	Short: "Save current Claude Code state to a profile",
	Long: `Saves your current Claude Code configuration (plugins, MCP servers, marketplaces) to a profile.

If no name is given, saves to the currently active profile.
If the profile exists, prompts for confirmation unless -y is used.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runProfileSave,
}

var profileCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a new profile by copying an existing one",
	Long: `Creates a new profile based on an existing profile.

Use --from to specify the source profile, or select interactively.
With -y flag, uses the currently active profile as the source.`,
	Args: cobra.ExactArgs(1),
	RunE: runProfileCreate,
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

var profileCurrentCmd = &cobra.Command{
	Use:   "current",
	Short: "Show the currently active profile",
	RunE:  runProfileCurrent,
}

// Flags for profile use command
var (
	profileUseSetup         bool
	profileUseNoInteractive bool
)

func init() {
	rootCmd.AddCommand(profileCmd)
	profileCmd.AddCommand(profileListCmd)
	profileCmd.AddCommand(profileUseCmd)
	profileCmd.AddCommand(profileSaveCmd)
	profileCmd.AddCommand(profileCreateCmd)
	profileCmd.AddCommand(profileShowCmd)
	profileCmd.AddCommand(profileSuggestCmd)
	profileCmd.AddCommand(profileCurrentCmd)

	profileCreateCmd.Flags().StringVar(&profileCreateFromFlag, "from", "", "Source profile to copy from")

	// Add flags to profile use command
	profileUseCmd.Flags().BoolVar(&profileUseSetup, "setup", false, "Force post-apply setup wizard to run")
	profileUseCmd.Flags().BoolVar(&profileUseNoInteractive, "no-interactive", false, "Skip post-apply setup wizard (for CI/scripting)")
}

func runProfileList(cmd *cobra.Command, args []string) error {
	profilesDir := getProfilesDir()

	// Load user profiles from disk
	userProfiles, err := profile.List(profilesDir)
	if err != nil {
		return fmt.Errorf("failed to list profiles: %w", err)
	}

	// Load embedded (built-in) profiles
	embeddedProfiles, embeddedErr := profile.ListEmbeddedProfiles()
	if embeddedErr != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to load built-in profiles: %v\n", embeddedErr)
		embeddedProfiles = []*profile.Profile{} // Prevent nil slice panic
	}

	// Track which profiles exist on disk
	userProfileNames := make(map[string]bool)
	for _, p := range userProfiles {
		userProfileNames[p.Name] = true
	}

	// Get active profile from config
	cfg, _ := config.Load()
	activeProfile := ""
	if cfg != nil {
		activeProfile = cfg.Preferences.ActiveProfile
	}

	// Check if we have any profiles to show
	hasBuiltIn := false
	for _, p := range embeddedProfiles {
		if !userProfileNames[p.Name] {
			hasBuiltIn = true
			break
		}
	}

	if len(userProfiles) == 0 && !hasBuiltIn {
		fmt.Println("No profiles found.")
		fmt.Println("Create one with: claudeup profile save <name>")
		return nil
	}

	fmt.Println("Available profiles:")
	fmt.Println()

	// Show built-in profiles first (ones not yet extracted to disk)
	for _, p := range embeddedProfiles {
		if userProfileNames[p.Name] {
			continue // Skip if user has customized this profile
		}

		marker := "  "
		if p.Name == activeProfile {
			marker = "* "
		}

		desc := p.Description
		if desc == "" {
			desc = "(no description)"
		}

		fmt.Printf("%s%-20s %s [built-in]\n", marker, p.Name, desc)
	}

	// Show user profiles
	for _, p := range userProfiles {
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

	// Load the profile (try disk first, then embedded)
	p, err := loadProfileWithFallback(profilesDir, name)
	if err != nil {
		return fmt.Errorf("profile %q not found: %w", name, err)
	}

	// Security check FIRST: warn about hooks from non-embedded profiles
	// Users should know about hooks before seeing the diff
	if p.PostApply != nil && !profile.IsEmbeddedProfile(name) {
		fmt.Println()
		fmt.Println("⚠ Security Warning: This profile contains a post-apply hook.")
		fmt.Println("  Hooks execute arbitrary commands on your system.")
		fmt.Println("  Only proceed if you trust the source of this profile.")
		if p.PostApply.Script != "" {
			fmt.Printf("  Script: %s\n", p.PostApply.Script)
		}
		if p.PostApply.Command != "" {
			fmt.Printf("  Command: %s\n", p.PostApply.Command)
		}
		fmt.Println()
		if !confirmProceed() {
			fmt.Println("Cancelled.")
			return nil
		}
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

	// Prepare hook options - extract scripts first so we can defer cleanup immediately
	scriptDir := profile.GetEmbeddedProfileScriptDir(name)
	if scriptDir != "" {
		defer os.RemoveAll(scriptDir)
	}

	hookOpts := profile.HookOptions{
		ForceSetup:    profileUseSetup,
		NoInteractive: profileUseNoInteractive,
		ScriptDir:     scriptDir,
	}

	// Check if hook should run BEFORE applying (captures pre-apply state for first-run detection)
	shouldRunHook := profile.ShouldRunHook(p, claudeDir, claudeJSONPath, hookOpts)

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

	// Silently clean up stale plugin entries
	cleanupStalePlugins(claudeDir)

	fmt.Println()
	fmt.Println("✓ Profile applied!")

	// Run post-apply hook if applicable (decision was made before apply)
	if shouldRunHook {
		fmt.Println()
		if err := profile.RunHook(p, hookOpts); err != nil {
			fmt.Printf("  ✗ Post-apply hook failed: %v\n", err)
			return fmt.Errorf("hook execution failed: %w", err)
		}
	}

	return nil
}

// cleanupStalePlugins removes plugin entries with invalid paths
// This is called automatically after profile apply to clean up zombie entries
func cleanupStalePlugins(claudeDir string) {
	plugins, err := claude.LoadPlugins(claudeDir)
	if err != nil {
		// Only warn if not a simple "file not found" - that's expected on fresh installs
		if !os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "  Warning: could not load plugins for cleanup: %v\n", err)
		}
		return
	}

	removed := 0
	for name, plugin := range plugins.GetAllPlugins() {
		if !plugin.PathExists() {
			if plugins.DisablePlugin(name) {
				removed++
			}
		}
	}

	if removed > 0 {
		if err := claude.SavePlugins(claudeDir, plugins); err != nil {
			fmt.Fprintf(os.Stderr, "  Warning: could not save cleaned plugins: %v\n", err)
		} else {
			fmt.Printf("  Cleaned up %d stale plugin entries\n", removed)
		}
	}
}

func runProfileSave(cmd *cobra.Command, args []string) error {
	profilesDir := getProfilesDir()

	// Determine profile name
	var name string
	if len(args) > 0 {
		name = args[0]
	} else {
		// Use active profile name
		cfg, _ := config.Load()
		if cfg == nil || cfg.Preferences.ActiveProfile == "" {
			return fmt.Errorf("no profile name given and no active profile set. Use 'claudeup profile save <name>' or 'claudeup profile use <name>' first")
		}
		name = cfg.Preferences.ActiveProfile
		fmt.Printf("Saving to active profile: %s\n", name)
	}

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

	fmt.Printf("✓ Saved profile %q\n", name)
	fmt.Println()
	fmt.Printf("  MCP Servers:   %d\n", len(p.MCPServers))
	fmt.Printf("  Marketplaces:  %d\n", len(p.Marketplaces))
	fmt.Printf("  Plugins:       %d\n", len(p.Plugins))

	return nil
}

func runProfileShow(cmd *cobra.Command, args []string) error {
	name := args[0]
	profilesDir := getProfilesDir()

	// Load the profile (try disk first, then embedded)
	p, err := loadProfileWithFallback(profilesDir, name)
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
			fmt.Printf("  - %s\n", m.DisplayName())
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
			fmt.Printf("    + Marketplace: %s\n", m.DisplayName())
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
		fmt.Println("Create one with: claudeup profile save <name>")
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

// loadProfileWithFallback tries to load a profile from disk first,
// falling back to embedded profiles if not found
func loadProfileWithFallback(profilesDir, name string) (*profile.Profile, error) {
	// Try disk first
	p, err := profile.Load(profilesDir, name)
	if err == nil {
		return p, nil
	}

	// Fall back to embedded profiles
	return profile.GetEmbeddedProfile(name)
}

// getAllProfiles returns all available profiles (user + embedded), with user profiles taking precedence
func getAllProfiles(profilesDir string) ([]*profile.Profile, error) {
	// Load user profiles
	userProfiles, err := profile.List(profilesDir)
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to list user profiles: %w", err)
	}

	// Track user profile names
	userNames := make(map[string]bool)
	for _, p := range userProfiles {
		userNames[p.Name] = true
	}

	// Load embedded profiles (skip ones that exist on disk)
	embeddedProfiles, err := profile.ListEmbeddedProfiles()
	if err != nil {
		// Non-fatal - just use user profiles
		return userProfiles, nil
	}

	// Combine: user profiles + embedded profiles not on disk
	result := make([]*profile.Profile, 0, len(userProfiles)+len(embeddedProfiles))
	result = append(result, userProfiles...)
	for _, p := range embeddedProfiles {
		if !userNames[p.Name] {
			result = append(result, p)
		}
	}

	return result, nil
}

// promptProfileSelection displays an interactive menu to select a profile
func promptProfileSelection(profilesDir, newName string) (*profile.Profile, error) {
	profiles, err := getAllProfiles(profilesDir)
	if err != nil {
		return nil, err
	}

	if len(profiles) == 0 {
		return nil, fmt.Errorf("no profiles available to copy from")
	}

	fmt.Printf("\nWhich profile should %q be based on?\n\n", newName)
	for i, p := range profiles {
		desc := p.Description
		if desc == "" {
			desc = "(no description)"
		}
		fmt.Printf("  %d) %-20s %s\n", i+1, p.Name, desc)
	}
	fmt.Println()

	fmt.Print("Enter number or name: ")
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("failed to read input: %w", err)
	}
	input = strings.TrimSpace(input)

	// Validate non-empty input
	if input == "" {
		return nil, fmt.Errorf("no selection made")
	}

	// Try as number first
	if num, err := strconv.Atoi(input); err == nil {
		if num >= 1 && num <= len(profiles) {
			return profiles[num-1], nil
		}
		return nil, fmt.Errorf("invalid selection: %d (must be 1-%d)", num, len(profiles))
	}

	// Try as name
	for _, p := range profiles {
		if p.Name == input {
			return p, nil
		}
	}

	return nil, fmt.Errorf("profile %q not found", input)
}

func runProfileCreate(cmd *cobra.Command, args []string) error {
	name := args[0]
	profilesDir := getProfilesDir()

	// Check if target profile already exists
	existingPath := filepath.Join(profilesDir, name+".json")
	if _, err := os.Stat(existingPath); err == nil {
		return fmt.Errorf("profile %q already exists. Use 'claudeup profile save %s' to update it", name, name)
	}

	// Determine source profile
	var sourceProfile *profile.Profile
	var err error

	if profileCreateFromFlag != "" {
		// Explicit --from flag
		sourceProfile, err = loadProfileWithFallback(profilesDir, profileCreateFromFlag)
		if err != nil {
			return fmt.Errorf("profile %q not found: %w", profileCreateFromFlag, err)
		}
	} else if config.YesFlag {
		// -y flag: use active profile
		cfg, _ := config.Load()
		if cfg == nil || cfg.Preferences.ActiveProfile == "" {
			return fmt.Errorf("no active profile. Use --from <profile> to specify base")
		}
		sourceProfile, err = loadProfileWithFallback(profilesDir, cfg.Preferences.ActiveProfile)
		if err != nil {
			return fmt.Errorf("active profile %q not found: %w", cfg.Preferences.ActiveProfile, err)
		}
		fmt.Printf("Using active profile: %s\n", cfg.Preferences.ActiveProfile)
	} else {
		// Interactive selection
		sourceProfile, err = promptProfileSelection(profilesDir, name)
		if err != nil {
			return err
		}
	}

	// Clone the profile with the new name
	newProfile := sourceProfile.Clone(name)

	// Save
	if err := profile.Save(profilesDir, newProfile); err != nil {
		return fmt.Errorf("failed to save profile: %w", err)
	}

	fmt.Printf("✓ Created profile %q (based on %q)\n", name, sourceProfile.Name)
	fmt.Println()
	fmt.Printf("  MCP Servers:   %d\n", len(newProfile.MCPServers))
	fmt.Printf("  Marketplaces:  %d\n", len(newProfile.Marketplaces))
	fmt.Printf("  Plugins:       %d\n", len(newProfile.Plugins))

	return nil
}

func runProfileCurrent(cmd *cobra.Command, args []string) error {
	// Use same pattern as runStatus - gracefully handle missing config
	cfg, _ := config.Load()
	activeProfile := ""
	if cfg != nil {
		activeProfile = cfg.Preferences.ActiveProfile
	}

	if activeProfile == "" {
		fmt.Println("No profile is currently active.")
		fmt.Println("Use 'claudeup profile use <name>' to apply a profile.")
		return nil
	}

	// Load the profile to show details
	profilesDir := getProfilesDir()
	p, err := loadProfileWithFallback(profilesDir, activeProfile)
	if err != nil {
		// Profile was set but can't be loaded - show name and error
		fmt.Printf("Current profile: %s (details unavailable: %v)\n", activeProfile, err)
		return nil
	}

	fmt.Printf("Current profile: %s\n", p.Name)
	if p.Description != "" {
		fmt.Printf("  %s\n", p.Description)
	}
	fmt.Println()
	fmt.Printf("  Marketplaces: %d\n", len(p.Marketplaces))
	fmt.Printf("  Plugins:      %d\n", len(p.Plugins))
	fmt.Printf("  MCP Servers:  %d\n", len(p.MCPServers))

	return nil
}
