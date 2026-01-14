// ABOUTME: Unit tests for custom error types
// ABOUTME: Tests error message formatting for version and path errors
package claude

import (
	"strings"
	"testing"
)

func TestFormatVersionError(t *testing.T) {
	err := &FormatVersionError{
		Component: "plugin registry",
		Found:     3,
		Supported: "1-2",
	}

	msg := err.Error()

	// Check key information is present
	if !strings.Contains(msg, "plugin registry") {
		t.Error("Error message should contain component name")
	}
	if !strings.Contains(msg, "version: 3") {
		t.Error("Error message should contain found version")
	}
	if !strings.Contains(msg, "versions: 1-2") {
		t.Error("Error message should contain supported versions")
	}
	if !strings.Contains(msg, "go install") {
		t.Error("Error message should contain update instructions")
	}
	if !strings.Contains(msg, "github.com/claudeup/claudeup/v2/issues") {
		t.Error("Error message should contain issue tracker URL")
	}
}

func TestPathNotFoundError(t *testing.T) {
	err := &PathNotFoundError{
		Component:    "plugin registry",
		ExpectedPath: "/Users/test/.claude/plugins/installed_plugins.json",
		ClaudeDir:    "/Users/test/.claude",
	}

	msg := err.Error()

	// Check key information is present
	if !strings.Contains(msg, "plugin registry") {
		t.Error("Error message should contain component name")
	}
	if !strings.Contains(msg, "installed_plugins.json") {
		t.Error("Error message should contain expected path")
	}
	if !strings.Contains(msg, "ls -R /Users/test/.claude") {
		t.Error("Error message should contain diagnostic command")
	}
	if !strings.Contains(msg, "github.com/claudeup/claudeup/v2/issues") {
		t.Error("Error message should contain issue tracker URL")
	}
}
