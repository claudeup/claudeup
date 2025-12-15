// ABOUTME: Entry point for the claudeup CLI tool
// ABOUTME: Initializes and executes the root command
package main

import (
	"fmt"
	"os"
	"runtime/debug"

	"github.com/claudeup/claudeup/internal/commands"
	"github.com/claudeup/claudeup/internal/ui"
)

var version = "dev" // Injected at build time via -ldflags

func main() {
	// If version is still "dev", try to get it from build info (set by go install)
	if version == "dev" {
		if info, ok := debug.ReadBuildInfo(); ok {
			version = info.Main.Version
		}
	}
	commands.SetVersion(version)

	if err := commands.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, ui.FormatError(err))
		os.Exit(1)
	}
}
