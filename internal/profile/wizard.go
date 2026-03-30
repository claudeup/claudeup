// ABOUTME: Interactive wizard for creating profiles
// ABOUTME: Handles name validation, marketplace selection, plugin selection
package profile

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// WizardIO provides I/O dependencies for interactive wizard functions.
// All wizard steps in a sequence must share one WizardIO so the buffered
// reader state is preserved between prompts.
type WizardIO struct {
	In       io.Reader
	bufIn    *bufio.Reader // derived from In; use BufIn() accessor
	Out      io.Writer
	Err      io.Writer
	LookPath func(string) (string, error) // resolves gum binary path
}

// NewWizardIO constructs a WizardIO, deriving the buffered reader from in.
func NewWizardIO(in io.Reader, out, errW io.Writer, lookPath func(string) (string, error)) WizardIO {
	return WizardIO{
		In:       in,
		bufIn:    bufio.NewReader(in),
		Out:      out,
		Err:      errW,
		LookPath: lookPath,
	}
}

// BufIn returns the shared buffered reader over In.
func (wio WizardIO) BufIn() *bufio.Reader { return wio.bufIn }

// DefaultWizardIO returns a WizardIO wired to the real OS streams.
func DefaultWizardIO() WizardIO {
	return NewWizardIO(os.Stdin, os.Stdout, os.Stderr, exec.LookPath)
}

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

// sanitizeForGum validates that a string is safe to pass to gum
// Rejects strings with newlines or null bytes that could confuse output parsing
func sanitizeForGum(s string) error {
	if strings.ContainsAny(s, "\n\r\x00") {
		return fmt.Errorf("invalid characters in name: %q", s)
	}
	return nil
}

// knownMarketplaceEntry represents a marketplace entry in known_marketplaces.json
type knownMarketplaceEntry struct {
	Source struct {
		Source string `json:"source"`
		Repo   string `json:"repo,omitempty"`
		URL    string `json:"url,omitempty"`
	} `json:"source"`
}

// loadKnownMarketplaces reads marketplaces from ~/.claude/plugins/known_marketplaces.json
func loadKnownMarketplaces() ([]Marketplace, error) {
	claudeDir := DefaultClaudeDir()
	marketplacesFile := filepath.Join(claudeDir, "plugins", "known_marketplaces.json")

	data, err := os.ReadFile(marketplacesFile)
	if err != nil {
		return nil, err
	}

	var entries map[string]knownMarketplaceEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, fmt.Errorf("failed to parse known_marketplaces.json: %w", err)
	}

	marketplaces := make([]Marketplace, 0, len(entries))
	for _, entry := range entries {
		marketplaces = append(marketplaces, Marketplace{
			Source: entry.Source.Source,
			Repo:   entry.Source.Repo,
			URL:    entry.Source.URL,
		})
	}

	return marketplaces, nil
}

// GetAvailableMarketplaces returns all available marketplaces for selection
// Loads from ~/.claude/plugins/known_marketplaces.json, falling back to embedded profiles
// Filters out invalid entries where both repo and url are empty
func GetAvailableMarketplaces() []Marketplace {
	// Try to load from known_marketplaces.json
	marketplaces, err := loadKnownMarketplaces()
	if err == nil && len(marketplaces) > 0 {
		return filterValidMarketplaces(marketplaces)
	}

	// Fallback to marketplaces from embedded profiles
	embeddedProfiles, err := ListEmbeddedProfiles()
	if err != nil {
		return []Marketplace{}
	}

	// Collect unique marketplaces from embedded profiles
	seen := make(map[string]bool)
	result := make([]Marketplace, 0)
	for _, p := range embeddedProfiles {
		for _, m := range p.Marketplaces {
			key := m.Source + ":" + m.Repo + m.URL
			if !seen[key] {
				seen[key] = true
				result = append(result, m)
			}
		}
	}

	return filterValidMarketplaces(result)
}

// filterValidMarketplaces removes marketplaces with empty display names.
// This handles malformed entries where both repo and url are empty.
func filterValidMarketplaces(marketplaces []Marketplace) []Marketplace {
	result := make([]Marketplace, 0, len(marketplaces))
	for _, m := range marketplaces {
		if m.DisplayName() != "" {
			result = append(result, m)
		}
	}
	return result
}

// installedPluginsFile represents the structure of installed_plugins.json
type installedPluginsFile struct {
	Version int                         `json:"version"`
	Plugins map[string][]map[string]any `json:"plugins"`
}

// getInstalledPlugins returns a set of currently installed plugin names
// Plugin names are in format: plugin-name@marketplace-name
func getInstalledPlugins() map[string]bool {
	claudeDir := DefaultClaudeDir()
	installedFile := filepath.Join(claudeDir, "plugins", "installed_plugins.json")

	data, err := os.ReadFile(installedFile)
	if err != nil {
		// File doesn't exist or can't be read - return empty set
		return make(map[string]bool)
	}

	var installed installedPluginsFile
	if err := json.Unmarshal(data, &installed); err != nil {
		// Parse error - return empty set
		return make(map[string]bool)
	}

	// Build set of installed plugin names
	installedSet := make(map[string]bool, len(installed.Plugins))
	for pluginName := range installed.Plugins {
		installedSet[pluginName] = true
	}

	return installedSet
}

// PromptForName prompts the user to enter a profile name
// Returns the validated name or an error
func PromptForName(wio WizardIO) (string, error) {
	reader := wio.BufIn()

	for {
		fmt.Fprint(wio.Out, "Profile name: ")
		input, err := reader.ReadString('\n')
		if err != nil {
			return "", fmt.Errorf("failed to read input: %w", err)
		}

		name := strings.TrimSpace(input)
		if name == "" {
			fmt.Fprintln(wio.Out, "Error: profile name cannot be empty")
			continue // Re-prompt on empty input
		}

		if err := ValidateName(name); err != nil {
			fmt.Fprintf(wio.Out, "Error: %v\n", err)
			continue // Re-prompt on validation error
		}

		return name, nil
	}
}

// SelectMarketplaces prompts user to select marketplaces
// Returns selected marketplaces or error
func SelectMarketplaces(wio WizardIO, available []Marketplace) ([]Marketplace, error) {
	if len(available) == 0 {
		return nil, fmt.Errorf("no marketplaces available")
	}

	// Check if gum is available
	if _, err := wio.LookPath("gum"); err != nil {
		return fallbackMarketplaceSelection(wio, available)
	}

	// Build gum command with marketplace choices
	args := []string{"choose", "--no-limit", "--header=Select marketplaces:"}
	for _, m := range available {
		name := m.DisplayName()
		if err := sanitizeForGum(name); err != nil {
			// Skip marketplaces with invalid names rather than failing
			// This protects against malicious marketplace names
			continue
		}
		args = append(args, name)
	}

	cmd := exec.Command("gum", args...)
	cmd.Stdin = wio.In
	cmd.Stderr = wio.Err
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
func fallbackMarketplaceSelection(wio WizardIO, available []Marketplace) ([]Marketplace, error) {
	fmt.Fprintln(wio.Out, "\nSelect marketplaces (enter numbers separated by commas):")
	for i, m := range available {
		fmt.Fprintf(wio.Out, "  %d) %s\n", i+1, m.DisplayName())
	}
	fmt.Fprint(wio.Out, "\nYour selection: ")

	reader := wio.BufIn()
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
func SelectPluginsForMarketplace(wio WizardIO, marketplace Marketplace) ([]string, error) {
	if HasCategories(marketplace.Repo) {
		return selectPluginsByCategory(wio, marketplace)
	}
	return selectPluginsFlat(wio, marketplace)
}

// selectPluginsByCategory shows category selection, then collects plugins from selected categories
func selectPluginsByCategory(wio WizardIO, marketplace Marketplace) ([]string, error) {
	categories := GetCategories(marketplace.Repo)
	if len(categories) == 0 {
		return selectPluginsFlat(wio, marketplace)
	}

	// Select categories using gum
	selectedCategories, err := selectCategories(wio, categories)
	if err != nil {
		return nil, err
	}

	// Collect unique plugins from selected categories
	pluginSet := make(map[string]bool)
	for _, cat := range selectedCategories {
		for _, plugin := range cat.Plugins {
			pluginSet[plugin] = true
		}
	}

	// Convert to slice for refinement
	availablePlugins := make([]string, 0, len(pluginSet))
	for plugin := range pluginSet {
		availablePlugins = append(availablePlugins, plugin)
	}

	// Get installed plugins for pre-selection
	installed := getInstalledPlugins()

	// Let user refine plugin selection with installed ones pre-selected
	return refinePluginSelection(wio, availablePlugins, installed)
}

// selectCategories prompts user to select categories
func selectCategories(wio WizardIO, categories []Category) ([]Category, error) {
	// Check if gum is available
	if _, err := wio.LookPath("gum"); err != nil {
		return fallbackCategorySelection(wio, categories)
	}

	// Build gum command
	args := []string{"choose", "--no-limit", "--header=Select plugin categories:"}
	for _, cat := range categories {
		if err := sanitizeForGum(cat.Name); err != nil {
			// Skip categories with invalid names
			continue
		}
		args = append(args, cat.Name)
	}

	cmd := exec.Command("gum", args...)
	cmd.Stdin = wio.In
	cmd.Stderr = wio.Err
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
func fallbackCategorySelection(wio WizardIO, categories []Category) ([]Category, error) {
	fmt.Fprintln(wio.Out, "\nSelect categories (enter numbers separated by commas, or 'q' to skip):")
	for i, cat := range categories {
		fmt.Fprintf(wio.Out, "  %d) %s - %s\n", i+1, cat.Name, cat.Description)
	}
	fmt.Fprint(wio.Out, "\nYour selection: ")

	reader := wio.BufIn()
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

// refinePluginSelection allows user to select/deselect plugins from available list
// Installed plugins are pre-selected by default
func refinePluginSelection(wio WizardIO, availablePlugins []string, installed map[string]bool) ([]string, error) {
	if len(availablePlugins) == 0 {
		return []string{}, nil
	}

	// Check if gum is available
	if _, err := wio.LookPath("gum"); err != nil {
		return fallbackPluginRefinement(wio, availablePlugins, installed)
	}

	// Build gum command with plugin choices
	args := []string{"choose", "--no-limit", "--header=Select plugins (installed plugins are pre-selected):"}

	// Build index of installed plugin prefixes for O(1) lookup
	// Plugin format in installed_plugins.json: plugin-name@marketplace-suffix
	installedPrefixes := make(map[string]bool)
	for installedKey := range installed {
		// Extract plugin name before '@'
		if idx := strings.Index(installedKey, "@"); idx != -1 {
			prefix := installedKey[:idx]
			installedPrefixes[prefix] = true
		}
	}

	// Add selected flag for each installed plugin
	preselected := make([]string, 0)
	validPlugins := make([]string, 0, len(availablePlugins))
	for _, plugin := range availablePlugins {
		// Skip plugins with invalid names
		if err := sanitizeForGum(plugin); err != nil {
			continue
		}
		validPlugins = append(validPlugins, plugin)

		// Check if plugin is installed using O(1) map lookup
		if installedPrefixes[plugin] {
			preselected = append(preselected, plugin)
		}
	}

	// Add pre-selected plugins
	for _, plugin := range preselected {
		args = append(args, "--selected="+plugin)
	}

	// Add all valid plugins as choices
	args = append(args, validPlugins...)

	cmd := exec.Command("gum", args...)
	cmd.Stdin = wio.In
	cmd.Stderr = wio.Err
	output, err := cmd.Output()
	if err != nil {
		// User cancelled or gum error - return pre-selected plugins
		return preselected, nil
	}

	// Parse selected plugins
	selected := strings.Split(strings.TrimSpace(string(output)), "\n")
	result := make([]string, 0, len(selected))
	for _, plugin := range selected {
		plugin = strings.TrimSpace(plugin)
		if plugin != "" {
			result = append(result, plugin)
		}
	}

	return result, nil
}

// fallbackPluginRefinement provides simple plugin selection when gum unavailable
func fallbackPluginRefinement(wio WizardIO, availablePlugins []string, installed map[string]bool) ([]string, error) {
	fmt.Fprintln(wio.Out, "\nSelect plugins (enter numbers separated by commas, or press Enter to select all pre-selected):")

	preselected := make([]int, 0)
	for i, plugin := range availablePlugins {
		// Check if plugin is installed
		isInstalled := false
		// Simple heuristic: check if any key contains this plugin name
		for key := range installed {
			if strings.HasPrefix(key, plugin+"@") {
				isInstalled = true
				break
			}
		}

		marker := " "
		if isInstalled {
			marker = "*"
			preselected = append(preselected, i+1)
		}
		fmt.Fprintf(wio.Out, " %s %d) %s\n", marker, i+1, plugin)
	}

	if len(preselected) > 0 {
		preselectedNums := make([]string, len(preselected))
		for i, num := range preselected {
			preselectedNums[i] = fmt.Sprintf("%d", num)
		}
		fmt.Fprintf(wio.Out, "\n* = installed (pre-selected: %s)\n", strings.Join(preselectedNums, ","))
	}
	fmt.Fprint(wio.Out, "\nYour selection: ")

	reader := wio.BufIn()
	input, err := reader.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("failed to read input: %w", err)
	}

	input = strings.TrimSpace(input)

	// Empty input means accept pre-selected
	if input == "" {
		result := make([]string, len(preselected))
		for i, idx := range preselected {
			result[i] = availablePlugins[idx-1]
		}
		return result, nil
	}

	// Parse comma-separated numbers
	parts := strings.Split(input, ",")
	selected := make([]string, 0)
	seen := make(map[int]bool)

	for _, part := range parts {
		part = strings.TrimSpace(part)
		idx, err := strconv.Atoi(part)
		if err != nil || idx < 1 || idx > len(availablePlugins) {
			return nil, fmt.Errorf("invalid selection: %s", part)
		}
		// Skip duplicates
		if seen[idx] {
			continue
		}
		seen[idx] = true
		selected = append(selected, availablePlugins[idx-1])
	}

	return selected, nil
}

// marketplaceMetadata represents the structure of .claude-plugin/marketplace.json
type marketplaceMetadata struct {
	Plugins []struct {
		Name        string `json:"name"`
		Description string `json:"description,omitempty"`
		Version     string `json:"version,omitempty"`
	} `json:"plugins"`
}

// listPluginsFromMarketplace reads marketplace.json and returns available plugin names
func listPluginsFromMarketplace(marketplace Marketplace) ([]string, error) {
	claudeDir := DefaultClaudeDir()

	// Load known_marketplaces.json to find the install location
	marketplacesFile := filepath.Join(claudeDir, "plugins", "known_marketplaces.json")
	data, err := os.ReadFile(marketplacesFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read known_marketplaces.json: %w", err)
	}

	var knownMarketplaces map[string]struct {
		Source struct {
			Source string `json:"source"`
			Repo   string `json:"repo,omitempty"`
			URL    string `json:"url,omitempty"`
		} `json:"source"`
		InstallLocation string `json:"installLocation"`
	}

	if err := json.Unmarshal(data, &knownMarketplaces); err != nil {
		return nil, fmt.Errorf("failed to parse known_marketplaces.json: %w", err)
	}

	// Find matching marketplace by comparing source details
	var marketplacePath string
	for _, entry := range knownMarketplaces {
		if entry.Source.Source == marketplace.Source {
			match := false
			if marketplace.Repo != "" && entry.Source.Repo == marketplace.Repo {
				match = true
			} else if marketplace.URL != "" && entry.Source.URL == marketplace.URL {
				match = true
			}

			if match {
				marketplacePath = entry.InstallLocation
				break
			}
		}
	}

	if marketplacePath == "" {
		return nil, fmt.Errorf("marketplace not found in known_marketplaces.json")
	}

	// Read marketplace.json from the install location
	metadataPath := filepath.Join(marketplacePath, ".claude-plugin", "marketplace.json")
	metadataData, err := os.ReadFile(metadataPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read marketplace.json: %w", err)
	}

	var metadata marketplaceMetadata
	if err := json.Unmarshal(metadataData, &metadata); err != nil {
		return nil, fmt.Errorf("failed to parse marketplace.json: %w", err)
	}

	// Extract plugin names
	plugins := make([]string, 0, len(metadata.Plugins))
	for _, plugin := range metadata.Plugins {
		plugins = append(plugins, plugin.Name)
	}

	return plugins, nil
}

// selectPluginsFlat shows flat plugin list for marketplaces without categories
func selectPluginsFlat(wio WizardIO, marketplace Marketplace) ([]string, error) {
	// List available plugins from marketplace metadata
	availablePlugins, err := listPluginsFromMarketplace(marketplace)
	if err != nil {
		fmt.Fprintf(wio.Err, "Warning: Failed to list plugins from marketplace %q: %v\n", marketplace.DisplayName(), err)
		fmt.Fprintln(wio.Err, "Profile will be created without any plugins from this marketplace.")
		fmt.Fprintln(wio.Err)
		return []string{}, nil
	}

	if len(availablePlugins) == 0 {
		fmt.Fprintf(wio.Out, "No plugins found in marketplace %q\n", marketplace.DisplayName())
		return []string{}, nil
	}

	// Get installed plugins for pre-selection
	installed := getInstalledPlugins()

	// Let user select plugins with installed ones pre-selected
	return refinePluginSelection(wio, availablePlugins, installed)
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
func PromptForDescription(wio WizardIO, autoGenerated string) (string, error) {
	// Check if gum is available
	if _, err := wio.LookPath("gum"); err != nil {
		return fallbackDescriptionPrompt(wio, autoGenerated)
	}

	// Ask if user wants to edit (Yes = edit, No = use auto-generated)
	confirmMsg := fmt.Sprintf("Edit description?\n  Auto-generated: %s", autoGenerated)
	cmd := exec.Command("gum", "confirm", confirmMsg)
	cmd.Stdin = wio.In
	cmd.Stderr = wio.Err
	if err := cmd.Run(); err != nil {
		// User said no or cancelled - use auto-generated
		return autoGenerated, nil
	}

	// User said yes - open editor
	return editDescription(wio, autoGenerated)
}

// editDescription opens gum write for editing the description
func editDescription(wio WizardIO, placeholder string) (string, error) {
	cmd := exec.Command("gum", "write", "--placeholder", placeholder, "--header=Edit description (Ctrl+D to save):")
	cmd.Stdin = wio.In
	cmd.Stderr = wio.Err
	output, err := cmd.Output()
	if err != nil {
		// User cancelled, use placeholder
		return placeholder, nil
	}

	description := strings.TrimSpace(string(output))
	if description == "" {
		return placeholder, nil
	}

	return description, nil
}

// fallbackDescriptionPrompt provides simple prompt when gum unavailable
func fallbackDescriptionPrompt(wio WizardIO, autoGenerated string) (string, error) {
	fmt.Fprintf(wio.Out, "Edit description?\n  Auto-generated: %s\n", autoGenerated)
	fmt.Fprint(wio.Out, "[y/N]: ")

	reader := wio.BufIn()
	input, err := reader.ReadString('\n')
	if err != nil {
		if err == io.EOF {
			return autoGenerated, nil
		}
		return "", fmt.Errorf("failed to read input: %w", err)
	}

	choice := strings.TrimSpace(strings.ToLower(input))
	if choice == "y" || choice == "yes" {
		// User wants to edit
		fmt.Fprint(wio.Out, "\nEnter custom description: ")
		newInput, err := reader.ReadString('\n')
		if err != nil {
			return "", fmt.Errorf("failed to read input: %w", err)
		}

		description := strings.TrimSpace(newInput)
		if description == "" {
			return autoGenerated, nil
		}

		return description, nil
	}

	// User declined or pressed enter - use auto-generated
	return autoGenerated, nil
}
