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
