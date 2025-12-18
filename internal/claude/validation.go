// ABOUTME: Schema validation functions for Claude CLI data structures
// ABOUTME: Validates plugin registry and settings formats to detect incompatibilities
package claude

import "fmt"

// validatePluginRegistry checks if a plugin registry has valid structure
// Returns FormatVersionError for unsupported versions
// Returns detailed errors for structural issues
func validatePluginRegistry(r *PluginRegistry) error {
	// Check version is in supported range
	if r.Version < 1 || r.Version > 2 {
		return &FormatVersionError{
			Component: "plugin registry",
			Found:     r.Version,
			Supported: "1-2",
		}
	}

	// V2-specific validation
	if r.Version == 2 {
		for name, instances := range r.Plugins {
			if len(instances) == 0 {
				return fmt.Errorf("plugin %s has empty metadata array (invalid v2 format)", name)
			}
			for i, inst := range instances {
				if inst.Scope == "" {
					return fmt.Errorf("plugin %s[%d] missing required 'scope' field", name, i)
				}
			}
		}
	}

	return nil
}

// validateSettings checks if settings has valid structure
// Settings format doesn't have explicit versioning yet, so we just check basic structure
func validateSettings(s *Settings) error {
	if s.EnabledPlugins == nil {
		return fmt.Errorf("invalid settings format: EnabledPlugins map is nil (format may have changed)")
	}
	return nil
}
