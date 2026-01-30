// ABOUTME: Acceptance tests for scope-aware config drift cleanup with profile clean command
// ABOUTME: Tests drift detection and cleanup for plugins across project and local scopes
package acceptance

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/claudeup/claudeup/v4/test/helpers"
)

var _ = Describe("Profile clean command for config drift", func() {
	var (
		env        *helpers.TestEnv
		binaryPath string
		projectDir string
	)

	BeforeEach(func() {
		binaryPath = helpers.BuildBinary()
		env = helpers.NewTestEnv(binaryPath)

		projectDir = env.ProjectDir("test-project")
		err := os.MkdirAll(filepath.Join(projectDir, ".claude"), 0755)
		Expect(err).NotTo(HaveOccurred())

		// Create minimal installed_plugins.json
		installedPlugins := map[string]interface{}{
			"version": 2,
			"plugins": map[string]interface{}{},
		}
		helpers.WriteJSON(filepath.Join(env.ClaudeDir, "plugins", "installed_plugins.json"), installedPlugins)
	})

	AfterEach(func() {
		env.Cleanup()
	})

	// NOTE: Drift detection and cleanup tests removed during .claudeup.json simplification.
	// The .claudeup.json file no longer contains a 'plugins' field - it only stores the
	// active profile name. Drift is now detected by comparing Claude settings against the
	// profile definition (not .claudeup.json). These tests were testing obsolete behavior.
	//
	// See recovery doc: ~/.claudeup/prompts/2025-12-27-simplify-claudeup-json.md
	// Unit test coverage for the new architecture exists in internal/profile/*_test.go
})
