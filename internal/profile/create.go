// ABOUTME: Non-interactive profile creation from flags or file input
// ABOUTME: Provides CreateSpec validation and profile construction
package profile

import (
	"fmt"
	"strings"
)

// ParseMarketplaceArg parses a marketplace argument in "owner/repo" format.
// Whitespace around owner and repo is trimmed for robustness.
// Additional path segments (owner/repo/extra) are preserved in Repo field
// and will be validated when the marketplace is actually accessed.
func ParseMarketplaceArg(arg string) (Marketplace, error) {
	arg = strings.TrimSpace(arg)
	if arg == "" {
		return Marketplace{}, fmt.Errorf("marketplace cannot be empty")
	}

	parts := strings.SplitN(arg, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return Marketplace{}, fmt.Errorf("invalid marketplace format %q: expected owner/repo", arg)
	}

	owner := strings.TrimSpace(parts[0])
	repo := strings.TrimSpace(parts[1])

	if owner == "" || repo == "" {
		return Marketplace{}, fmt.Errorf("invalid marketplace format %q: expected owner/repo", arg)
	}

	return Marketplace{
		Source: "github",
		Repo:   owner + "/" + repo,
	}, nil
}

// ValidatePluginFormat validates a plugin string is in "name@marketplace-ref" format.
// Plugin names can contain colons (e.g., "backend:api-design@marketplace").
// Uses LastIndex to find @ since plugin names may contain @, but the last @ is the separator.
func ValidatePluginFormat(plugin string) error {
	plugin = strings.TrimSpace(plugin)
	if plugin == "" {
		return fmt.Errorf("plugin cannot be empty")
	}

	atIdx := strings.LastIndex(plugin, "@")
	if atIdx == -1 {
		return fmt.Errorf("invalid plugin format %q: expected name@marketplace-ref", plugin)
	}

	name := plugin[:atIdx]
	ref := plugin[atIdx+1:]

	if name == "" {
		return fmt.Errorf("invalid plugin format %q: plugin name cannot be empty", plugin)
	}
	if ref == "" {
		return fmt.Errorf("invalid plugin format %q: marketplace ref cannot be empty", plugin)
	}

	return nil
}

// ValidateCreateSpec validates input for non-interactive profile creation.
// Requires a description and at least one marketplace.
// Validates marketplace and plugin formats using ParseMarketplaceArg and ValidatePluginFormat.
func ValidateCreateSpec(description string, marketplaces []string, plugins []string) error {
	if strings.TrimSpace(description) == "" {
		return fmt.Errorf("description is required")
	}

	if len(marketplaces) == 0 {
		return fmt.Errorf("at least one marketplace is required")
	}

	for _, m := range marketplaces {
		if _, err := ParseMarketplaceArg(m); err != nil {
			return err
		}
	}

	for _, p := range plugins {
		if err := ValidatePluginFormat(p); err != nil {
			return err
		}
	}

	return nil
}

// CreateFromFlags creates a profile from CLI flag values.
// Uses ValidateCreateSpec for input validation and ParseMarketplaceArg
// to convert marketplace strings to Marketplace structs.
func CreateFromFlags(name, description string, marketplaceArgs, plugins []string) (*Profile, error) {
	if err := ValidateCreateSpec(description, marketplaceArgs, plugins); err != nil {
		return nil, err
	}

	marketplaces := make([]Marketplace, 0, len(marketplaceArgs))
	for _, arg := range marketplaceArgs {
		m, _ := ParseMarketplaceArg(arg) // Already validated by ValidateCreateSpec
		marketplaces = append(marketplaces, m)
	}

	return &Profile{
		Name:         name,
		Description:  description,
		Marketplaces: marketplaces,
		Plugins:      plugins,
		MCPServers:   []MCPServer{},
	}, nil
}
