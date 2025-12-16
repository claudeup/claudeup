// ABOUTME: Interactive wizard for creating profiles
// ABOUTME: Handles name validation, marketplace selection, plugin selection
package profile

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

// validNameRegex matches valid profile names: alphanumeric, hyphens, underscores
var validNameRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

// ValidateName checks if a profile name is valid
func ValidateName(name string) error {
	if name == "" {
		return fmt.Errorf("profile name cannot be empty")
	}

	if name == "current" {
		return fmt.Errorf("'current' is a reserved name")
	}

	if !validNameRegex.MatchString(name) {
		return fmt.Errorf("profile name contains invalid characters (use letters, numbers, hyphens, underscores only)")
	}

	return nil
}

// GetAvailableMarketplaces returns all available marketplaces for selection
// Currently returns only embedded marketplaces (wshobson/agents)
func GetAvailableMarketplaces() []Marketplace {
	// For V1, return hardcoded list of known marketplaces
	// Future: could load from registry or config
	return []Marketplace{
		{Source: "github", Repo: "wshobson/agents"},
	}
}

// PromptForName prompts the user to enter a profile name
// Returns the validated name or an error
func PromptForName() (string, error) {
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print("Profile name: ")
		input, err := reader.ReadString('\n')
		if err != nil {
			return "", fmt.Errorf("failed to read input: %w", err)
		}

		name := strings.TrimSpace(input)
		if name == "" {
			fmt.Println("Error: profile name cannot be empty")
			continue // Re-prompt on empty input
		}

		if err := ValidateName(name); err != nil {
			fmt.Printf("Error: %v\n", err)
			continue // Re-prompt on validation error
		}

		return name, nil
	}
}

// SelectMarketplaces prompts user to select marketplaces
// Returns selected marketplaces or error
func SelectMarketplaces(available []Marketplace) ([]Marketplace, error) {
	if len(available) == 0 {
		return nil, fmt.Errorf("no marketplaces available")
	}

	// Check if gum is available
	if _, err := exec.LookPath("gum"); err != nil {
		return fallbackMarketplaceSelection(available)
	}

	// Build gum command with marketplace choices
	args := []string{"choose", "--no-limit", "--header=Select marketplaces:"}
	for _, m := range available {
		args = append(args, m.DisplayName())
	}

	cmd := exec.Command("gum", args...)
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	output, err := cmd.Output()
	if err != nil {
		// User cancelled or gum error
		return nil, fmt.Errorf("marketplace selection cancelled")
	}

	// Parse selected marketplace names
	selectedNames := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(selectedNames) == 0 || selectedNames[0] == "" {
		return nil, fmt.Errorf("no marketplaces selected")
	}

	// Map selected names back to Marketplace structs
	selected := make([]Marketplace, 0)
	for _, name := range selectedNames {
		found := false
		for _, m := range available {
			if m.DisplayName() == name {
				selected = append(selected, m)
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("selected marketplace not found: %s", name)
		}
	}

	return selected, nil
}

// fallbackMarketplaceSelection provides simple numbered menu when gum unavailable
func fallbackMarketplaceSelection(available []Marketplace) ([]Marketplace, error) {
	fmt.Println("\nSelect marketplaces (enter numbers separated by commas):")
	for i, m := range available {
		fmt.Printf("  %d) %s\n", i+1, m.DisplayName())
	}
	fmt.Print("\nYour selection: ")

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("failed to read input: %w", err)
	}

	// Parse comma-separated numbers
	input = strings.TrimSpace(input)
	if input == "" {
		return nil, fmt.Errorf("no marketplaces selected")
	}

	parts := strings.Split(input, ",")
	selected := make([]Marketplace, 0)
	seen := make(map[int]bool)

	for _, part := range parts {
		part = strings.TrimSpace(part)
		idx, err := strconv.Atoi(part)
		if err != nil || idx < 1 || idx > len(available) {
			return nil, fmt.Errorf("invalid selection: %s", part)
		}
		// Skip duplicates
		if seen[idx] {
			continue
		}
		seen[idx] = true
		selected = append(selected, available[idx-1])
	}

	return selected, nil
}

// SelectPluginsForMarketplace prompts user to select plugins from a marketplace
// Uses category-based selection if marketplace has categories, otherwise flat list
func SelectPluginsForMarketplace(marketplace Marketplace) ([]string, error) {
	if HasCategories(marketplace.Repo) {
		return selectPluginsByCategory(marketplace)
	}
	return selectPluginsFlat(marketplace)
}

// selectPluginsByCategory shows category selection, then collects plugins from selected categories
func selectPluginsByCategory(marketplace Marketplace) ([]string, error) {
	categories := GetCategories(marketplace.Repo)
	if len(categories) == 0 {
		return selectPluginsFlat(marketplace)
	}

	// Select categories using gum
	selectedCategories, err := selectCategories(categories)
	if err != nil {
		return nil, err
	}

	// Collect plugins from selected categories
	pluginSet := make(map[string]bool)
	for _, cat := range selectedCategories {
		for _, plugin := range cat.Plugins {
			pluginSet[plugin] = true
		}
	}

	plugins := make([]string, 0, len(pluginSet))
	for plugin := range pluginSet {
		plugins = append(plugins, plugin)
	}

	return plugins, nil
}

// selectCategories prompts user to select categories
func selectCategories(categories []Category) ([]Category, error) {
	// Check if gum is available
	if _, err := exec.LookPath("gum"); err != nil {
		return fallbackCategorySelection(categories)
	}

	// Build gum command
	args := []string{"choose", "--no-limit", "--header=Select plugin categories:"}
	for _, cat := range categories {
		args = append(args, cat.Name)
	}

	cmd := exec.Command("gum", args...)
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("category selection cancelled")
	}

	// Parse selected category names
	selectedNames := strings.Split(strings.TrimSpace(string(output)), "\n")

	// Map back to Category structs
	selected := make([]Category, 0)
	for _, name := range selectedNames {
		for _, cat := range categories {
			if cat.Name == name {
				selected = append(selected, cat)
				break
			}
		}
	}

	if len(selected) == 0 {
		return nil, fmt.Errorf("no categories selected")
	}

	return selected, nil
}

// fallbackCategorySelection provides numbered menu when gum unavailable
func fallbackCategorySelection(categories []Category) ([]Category, error) {
	fmt.Println("\nSelect categories (enter numbers separated by commas, or 'q' to skip):")
	for i, cat := range categories {
		fmt.Printf("  %d) %s - %s\n", i+1, cat.Name, cat.Description)
	}
	fmt.Print("\nYour selection: ")

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("failed to read input: %w", err)
	}

	input = strings.TrimSpace(input)
	if input == "" || input == "q" {
		// Allow skipping category selection (empty plugin list is valid)
		return []Category{}, nil
	}

	parts := strings.Split(input, ",")
	selected := make([]Category, 0)
	seen := make(map[int]bool)

	for _, part := range parts {
		part = strings.TrimSpace(part)
		idx, err := strconv.Atoi(part)
		if err != nil || idx < 1 || idx > len(categories) {
			return nil, fmt.Errorf("invalid selection: %s", part)
		}
		// Skip duplicates
		if seen[idx] {
			continue
		}
		seen[idx] = true
		selected = append(selected, categories[idx-1])
	}

	return selected, nil
}

// selectPluginsFlat shows flat plugin list for marketplaces without categories
func selectPluginsFlat(marketplace Marketplace) ([]string, error) {
	// TODO: Implement flat plugin selection
	// For now, warn user and return empty list (no plugins selected)
	// This will be implemented when we have a way to list all plugins from a marketplace
	fmt.Printf("Warning: Plugin selection not yet supported for marketplace %q\n", marketplace.Source)
	fmt.Println("Profile will be created without any plugins.")
	fmt.Println()
	return []string{}, nil
}

// GenerateWizardDescription creates description based on wizard selections
func GenerateWizardDescription(marketplaceCount, pluginCount int) string {
	pluginWord := "plugins"
	if pluginCount == 1 {
		pluginWord = "plugin"
	}
	marketplaceWord := "marketplaces"
	if marketplaceCount == 1 {
		marketplaceWord = "marketplace"
	}
	return fmt.Sprintf("Custom profile with %d %s from %d %s",
		pluginCount, pluginWord, marketplaceCount, marketplaceWord)
}

// PromptForDescription shows auto-generated description and allows editing
func PromptForDescription(autoGenerated string) (string, error) {
	// Check if gum is available for input
	if _, err := exec.LookPath("gum"); err != nil {
		return fallbackDescriptionPrompt(autoGenerated)
	}

	// Use gum write to allow editing
	cmd := exec.Command("gum", "write", "--placeholder", autoGenerated, "--header=Profile description (Ctrl+D to save):")
	output, err := cmd.Output()
	if err != nil {
		// User cancelled, use auto-generated
		return autoGenerated, nil
	}

	description := strings.TrimSpace(string(output))
	if description == "" {
		return autoGenerated, nil
	}

	return description, nil
}

// fallbackDescriptionPrompt provides simple prompt when gum unavailable
func fallbackDescriptionPrompt(autoGenerated string) (string, error) {
	fmt.Printf("\nDescription [%s]: ", autoGenerated)

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return autoGenerated, nil
	}

	description := strings.TrimSpace(input)
	if description == "" {
		return autoGenerated, nil
	}

	return description, nil
}
