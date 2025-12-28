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

var _ = Describe("profile apply hobson", func() {
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
					"profile", "apply", "hobson", "-y",
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
				result := env.Run("profile", "apply", "hobson", "-y")

				// Wizard should NOT have started
				Expect(result.Stdout).NotTo(ContainSubstring("Hobson Profile Setup"))
				Expect(result.ExitCode).To(Equal(0))
			})
		})
	})

	Describe("--no-interactive flag", func() {
		It("skips the wizard entirely for CI/scripting", func() {
			result := env.Run("profile", "apply", "hobson", "-y", "--no-interactive")

			// Should succeed without wizard
			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("Profile applied"))
			Expect(result.Stdout).NotTo(ContainSubstring("Hobson Profile Setup"))
		})

		It("applies profile settings even without wizard", func() {
			result := env.Run("profile", "apply", "hobson", "-y", "--no-interactive")

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
					"profile", "apply", "hobson", "-y", "--setup",
				)

				// Wizard should have started despite existing plugins
				Expect(result.Stdout).To(ContainSubstring("Hobson Profile Setup"))
			})
		})

		Context("with marketplace already installed (no configuration changes)", func() {
			BeforeEach(func() {
				// Pre-install the hobson marketplace AND plugins
				env.CreateKnownMarketplaces(map[string]interface{}{
					"claude-code-workflows": map[string]interface{}{
						"source": map[string]interface{}{
							"source": "github",
							"repo":   "wshobson/agents",
						},
						"installLocation": filepath.Join(env.ClaudeDir, "plugins", "marketplaces", "claude-code-workflows"),
						"lastUpdated":     "2024-01-01T00:00:00Z",
					},
				})

				env.CreateInstalledPlugins(map[string]interface{}{
					"debugging-toolkit@wshobson-agents": []map[string]interface{}{
						{"scope": "user", "version": "1.0"},
					},
				})
			})

			It("runs the hook even when there are no config changes", func() {
				// This tests the fix for: when there are no diff changes (marketplace exists,
				// no plugins to install/remove because skipPluginDiff=true), the --setup flag
				// should still cause the hook to run
				result := env.RunWithEnvAndInput(
					map[string]string{"PATH": filepath.Dir(binaryPath) + ":" + os.Getenv("PATH")},
					"q\n",
					"profile", "apply", "hobson", "--setup",
				)

				// Should NOT say "No changes needed" and exit
				// Should run the wizard
				Expect(result.Stdout).To(ContainSubstring("Hobson Profile Setup"))
				Expect(result.Stdout).NotTo(ContainSubstring("No changes needed - profile already matches current state"))
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
				"profile", "apply", "hobson", "-y",
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

		It("falls back to prompt mode when gum cannot access TTY", func() {
			// When gum cannot access a TTY, the script falls back to prompt-based selection
			// Send 'q' to quit the fallback prompts
			result := env.RunWithEnvAndInput(
				map[string]string{"PATH": filepath.Dir(binaryPath) + ":" + os.Getenv("PATH")},
				"q\n", // Quit the prompt-based selection
				"profile", "apply", "hobson", "-y",
			)

			// Wizard should start
			Expect(result.Stdout).To(ContainSubstring("Hobson Profile Setup"))
			// Should show prompt-based selection (fallback mode)
			Expect(result.Stdout).To(ContainSubstring("Select development categories"))
			// And handle quit gracefully
			Expect(result.Stdout).To(ContainSubstring("Setup cancelled"))
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

			result := env.Run("profile", "apply", "failing-hook", "-y")

			Expect(result.ExitCode).NotTo(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("Post-apply hook failed"))
		})
	})
})
