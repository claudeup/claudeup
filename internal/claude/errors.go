// ABOUTME: Custom error types for Claude CLI compatibility issues
// ABOUTME: Provides structured errors with actionable user guidance
package claude

import "fmt"

// FormatVersionError indicates Claude CLI uses an unsupported format version
type FormatVersionError struct {
	Component string // e.g., "plugin registry", "settings"
	Found     int    // Version number found in file
	Supported string // Supported version range, e.g., "1-2"
}

func (e *FormatVersionError) Error() string {
	return fmt.Sprintf(`Claude CLI format incompatibility detected:

  Component: %s
  Found version: %d
  Supported versions: %s

This likely means your Claude CLI has been updated to a newer format.
Please update claudeup:

  go install github.com/claudeup/claudeup@latest

If the issue persists, please report at:
  https://github.com/claudeup/claudeup/issues`,
		e.Component, e.Found, e.Supported)
}

// PathNotFoundError indicates a Claude CLI file is missing from expected location
type PathNotFoundError struct {
	Component    string // e.g., "plugin registry", "settings"
	ExpectedPath string // Full path where file was expected
	ClaudeDir    string // Claude installation directory
}

func (e *PathNotFoundError) Error() string {
	return fmt.Sprintf(`Claude CLI file not found:

  Component: %s
  Expected path: %s
  Claude directory: %s

Possible causes:
  1. Claude CLI changed file locations (please report this issue)

To diagnose:
  ls -la %s

Please report at https://github.com/claudeup/claudeup/issues
Include the output of: ls -R ~/.claude`,
		e.Component, e.ExpectedPath, e.ClaudeDir, e.ClaudeDir)
}
