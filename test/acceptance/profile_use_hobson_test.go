// ABOUTME: Acceptance tests for hobson profile wizard behavior
// ABOUTME: Tests wizard triggering, --no-interactive, --setup flags, and gum/fallback modes
package acceptance

import (
	"os"
	"path/filepath"

	"github.com/claudeup/claudeup/internal/profile"
	"github.com/claudeup/claudeup/test/helpers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("profile use hobson", func() {
	var env *helpers.TestEnv

	BeforeEach(func() {
		env = helpers.NewTestEnv(binaryPath)
		env.CreateClaudeSettings()
		env.CreateInstalledPlugins(map[string]interface{}{})
		env.CreateKnownMarketplaces(map[string]interface{}{})
	})

	Describe("wizard triggering", func() {
		Context("on fresh install (no plugins from wshobson-agents)", func() {
			It("triggers the setup wizard", func() {
				// Provide 'q' to quit the wizard immediately
				result := env.RunWithEnvAndInput(
					map[string]string{"PATH": filepath.Dir(binaryPath) + ":" + os.Getenv("PATH")},
					"q\n",
					"profile", "use", "hobson", "-y",
				)

				// Wizard should have started - look for header
				Expect(result.Stdout).To(ContainSubstring("Hobson Profile Setup"))
			})
		})

		Context("with existing plugins from wshobson-agents marketplace", func() {
			BeforeEach(func() {
				// Simulate existing plugin from the hobson marketplace
				env.CreateInstalledPlugins(map[string]interface{}{
					"debugging-toolkit@wshobson-agents": []map[string]interface{}{
						{"scope": "user", "version": "1.0"},
					},
				})
			})

			It("does not trigger wizard (not first run)", func() {
				result := env.Run("profile", "use", "hobson", "-y")

				// Wizard should NOT have started
				Expect(result.Stdout).NotTo(ContainSubstring("Hobson Profile Setup"))
				Expect(result.ExitCode).To(Equal(0))
			})
		})
	})

	Describe("--no-interactive flag", func() {
		It("skips the wizard entirely for CI/scripting", func() {
			result := env.Run("profile", "use", "hobson", "-y", "--no-interactive")

			// Should succeed without wizard
			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("Profile applied"))
			Expect(result.Stdout).NotTo(ContainSubstring("Hobson Profile Setup"))
		})

		It("applies profile settings even without wizard", func() {
			result := env.Run("profile", "use", "hobson", "-y", "--no-interactive")

			Expect(result.ExitCode).To(Equal(0))
			// Should still set up the marketplace
			Expect(result.Stdout).To(ContainSubstring("marketplace"))
		})
	})

	Describe("--setup flag", func() {
		Context("with existing plugins (not first run)", func() {
			BeforeEach(func() {
				// Simulate existing plugin from the hobson marketplace
				env.CreateInstalledPlugins(map[string]interface{}{
					"debugging-toolkit@wshobson-agents": []map[string]interface{}{
						{"scope": "user", "version": "1.0"},
					},
				})
			})

			It("forces the wizard to re-run", func() {
				// Provide 'q' to quit the wizard immediately
				result := env.RunWithEnvAndInput(
					map[string]string{"PATH": filepath.Dir(binaryPath) + ":" + os.Getenv("PATH")},
					"q\n",
					"profile", "use", "hobson", "-y", "--setup",
				)

				// Wizard should have started despite existing plugins
				Expect(result.Stdout).To(ContainSubstring("Hobson Profile Setup"))
			})
		})
	})

	Describe("wizard execution", func() {
		BeforeEach(func() {
			// These tests require gum for non-TTY handling
			// In CI without gum, the fallback mode's read command fails without TTY
			// Check common gum locations including Go's bin directory
			gumPaths := []string{
				"/opt/homebrew/bin/gum",           // macOS homebrew (Apple Silicon)
				"/usr/local/bin/gum",              // macOS homebrew (Intel) / Linux
				"/usr/bin/gum",                    // System install
				os.Getenv("HOME") + "/go/bin/gum", // go install location
			}
			gumFound := false
			for _, path := range gumPaths {
				if _, err := os.Stat(path); err == nil {
					gumFound = true
					break
				}
			}
			if !gumFound {
				Skip("gum not installed - wizard execution tests require gum for non-TTY environments")
			}
		})

		It("starts the wizard and can be cancelled", func() {
			// The wizard uses gum if available, fallback prompts otherwise
			// Both modes show the same header and can be cancelled
			result := env.RunWithEnvAndInput(
				map[string]string{"PATH": filepath.Dir(binaryPath) + ":" + os.Getenv("PATH")},
				"q\n", // Send 'q' to cancel (works in fallback mode)
				"profile", "use", "hobson", "-y",
			)

			// Wizard should start and show header
			Expect(result.Stdout).To(ContainSubstring("Hobson Profile Setup"))
			// Should either be cancelled or show selection prompt
			Expect(result.Stdout).To(SatisfyAny(
				ContainSubstring("Setup cancelled"),
				ContainSubstring("No categories selected"),
				ContainSubstring("Select development categories"),
			))
		})

		It("handles empty selection gracefully", func() {
			// When run without TTY, gum returns empty selection
			result := env.RunWithEnvAndInput(
				map[string]string{"PATH": filepath.Dir(binaryPath) + ":" + os.Getenv("PATH")},
				"", // Empty input - gum will return empty selection without TTY
				"profile", "use", "hobson", "-y",
			)

			// Wizard should start
			Expect(result.Stdout).To(ContainSubstring("Hobson Profile Setup"))
			// And handle empty selection
			Expect(result.Stdout).To(SatisfyAny(
				ContainSubstring("No categories selected"),
				ContainSubstring("Setup cancelled"),
			))
		})
	})

	Describe("hook failure handling", func() {
		It("returns non-zero exit code when hook fails", func() {
			// Create a profile with a hook that will fail
			// Include a marketplace so there's a diff to apply (otherwise "no changes needed" exits early)
			env.CreateProfile(&profile.Profile{
				Name:        "failing-hook",
				Description: "Profile with failing hook for testing",
				Marketplaces: []profile.Marketplace{
					{Source: "github", Repo: "test/fake-marketplace"},
				},
				PostApply: &profile.PostApplyHook{
					Command:   "exit 1",
					Condition: "always",
				},
			})

			result := env.Run("profile", "use", "failing-hook", "-y")

			Expect(result.ExitCode).NotTo(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("Post-apply hook failed"))
		})
	})
})
