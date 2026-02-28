// ABOUTME: Acceptance tests for profile diff comparing saved profiles to live Claude Code state
// ABOUTME: Tests diff-vs-live behavior across scopes with plugins, MCP servers, and extensions
package acceptance

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/claudeup/claudeup/v5/internal/profile"
	"github.com/claudeup/claudeup/v5/test/helpers"
)

var _ = Describe("Profile diff vs live", func() {
	var (
		env        *helpers.TestEnv
		binaryPath string
	)

	BeforeEach(func() {
		binaryPath = helpers.BuildBinary()
		env = helpers.NewTestEnv(binaryPath)
		env.CreateClaudeSettings()
	})

	AfterEach(func() {
		env.Cleanup()
	})

	Describe("profile not found", func() {
		It("returns error for unknown profile", func() {
			result := env.Run("profile", "diff", "nonexistent-profile")

			Expect(result.ExitCode).To(Equal(1))
			Expect(result.Stderr).To(ContainSubstring("not found"))
		})
	})

	Describe("profile matches live state", func() {
		BeforeEach(func() {
			// Save a profile whose description matches the auto-generated one
			// SnapshotAllScopes generates "Empty profile" for an empty snapshot
			env.CreateProfile(&profile.Profile{
				Name:        "empty-match",
				Description: "Empty profile",
			})
		})

		It("shows no differences", func() {
			result := env.Run("profile", "diff", "empty-match")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("No differences"))
		})
	})

	Describe("live has extra plugins not in profile", func() {
		BeforeEach(func() {
			// Create a profile with no plugins
			env.CreateProfile(&profile.Profile{
				Name: "sparse",
			})

			// Set up live state with a plugin enabled
			env.CreateInstalledPlugins(map[string]interface{}{
				"extra-plugin@marketplace": map[string]interface{}{
					"scope": "user",
				},
			})
		})

		It("shows added plugins", func() {
			result := env.Run("profile", "diff", "sparse")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("+"))
			Expect(result.Stdout).To(ContainSubstring("extra-plugin@marketplace"))
		})
	})

	Describe("profile has plugins not in live", func() {
		BeforeEach(func() {
			// Create a profile with plugins
			env.CreateProfile(&profile.Profile{
				Name: "full-setup",
				PerScope: &profile.PerScopeSettings{
					User: &profile.ScopeSettings{
						Plugins: []string{"missing-plugin@marketplace"},
					},
				},
			})

			// Live state has no plugins
			env.CreateInstalledPlugins(map[string]interface{}{})
		})

		It("shows removed plugins", func() {
			result := env.Run("profile", "diff", "full-setup")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("-"))
			Expect(result.Stdout).To(ContainSubstring("missing-plugin@marketplace"))
		})
	})

	Describe("scope labels shown", func() {
		var projectDir string

		BeforeEach(func() {
			// Create a project directory with project-scope plugins
			projectDir = filepath.Join(env.TempDir, "myproject")
			Expect(os.MkdirAll(projectDir, 0755)).To(Succeed())

			// Create a profile with no plugins at any scope
			env.CreateProfile(&profile.Profile{
				Name:     "scoped-test",
				PerScope: &profile.PerScopeSettings{},
			})

			// Set up user-scope plugins in live state
			env.CreateInstalledPlugins(map[string]interface{}{
				"user-plugin@marketplace": map[string]interface{}{
					"scope": "user",
				},
			})

			// Set up project-scope plugins in live state
			env.CreateProjectScopeSettings(projectDir, map[string]bool{
				"project-plugin@marketplace": true,
			})
		})

		It("shows scope labels when diffs exist across scopes", func() {
			result := env.RunInDir(projectDir, "profile", "diff", "scoped-test")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("User"))
			Expect(result.Stdout).To(ContainSubstring("user-plugin@marketplace"))
		})
	})

	Describe("built-in profile works without --original", func() {
		BeforeEach(func() {
			// Live state has an extra plugin beyond what 'default' specifies
			env.CreateInstalledPlugins(map[string]interface{}{
				"extra-live-plugin@marketplace": map[string]interface{}{
					"scope": "user",
				},
			})
		})

		It("diffs built-in against live state", func() {
			result := env.Run("profile", "diff", "default")

			Expect(result.ExitCode).To(Equal(0))
			// Should show the extra live plugin as added (in live, not in profile)
			Expect(result.Stdout).To(ContainSubstring("extra-live-plugin@marketplace"))
		})
	})

	Describe("hint message", func() {
		BeforeEach(func() {
			env.CreateProfile(&profile.Profile{
				Name: "drift-test",
			})

			env.CreateInstalledPlugins(map[string]interface{}{
				"drifted-plugin@marketplace": map[string]interface{}{
					"scope": "user",
				},
			})
		})

		It("shows save hint when differences found", func() {
			result := env.Run("profile", "diff", "drift-test")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("profile save"))
			Expect(result.Stdout).To(ContainSubstring("drift-test"))
		})
	})
})
