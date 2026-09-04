// ABOUTME: Acceptance tests for doctor command
// ABOUTME: Tests diagnostic output including missing plugin detection and recommendations
package acceptance

import (
	"os"
	"path/filepath"

	"github.com/claudeup/claudeup/v5/test/helpers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("doctor", func() {
	var env *helpers.TestEnv

	BeforeEach(func() {
		env = helpers.NewTestEnv(binaryPath)
	})

	Describe("scope settings load errors", func() {
		BeforeEach(func() {
			// Write invalid JSON to the user-scope settings file to trigger a load error
			env.WriteFile(env.ClaudeDir, "settings.json", "{invalid json")
		})

		It("surfaces a warning for the failed scope and counts it in the summary", func() {
			result := env.Run("doctor")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("Checking Settings Scopes"))
			Expect(result.Stdout).To(ContainSubstring("user scope: failed to load settings"))
			Expect(result.Stdout).To(ContainSubstring("Settings: 1 scope load error"))
			Expect(result.Stdout).To(ContainSubstring("Restore or delete the corrupted file:"))
			Expect(result.Stdout).To(ContainSubstring("settings.json"))
			Expect(result.Stdout).To(ContainSubstring("Plugin analysis may be incomplete"))
			Expect(result.Stdout).To(ContainSubstring("Run the suggested commands to fix these issues"))
		})
	})

	Describe("corrupt project-scope settings", func() {
		It("surfaces a warning for the failed project scope", func() {
			projectDir := env.ProjectDir("corrupt-project")
			claudeDir := filepath.Join(projectDir, ".claude")
			Expect(os.MkdirAll(claudeDir, 0755)).To(Succeed())
			env.WriteFile(claudeDir, "settings.json", "{invalid json")

			result := env.RunInDir(projectDir, "doctor")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("Checking Settings Scopes"))
			Expect(result.Stdout).To(ContainSubstring("project scope: failed to load settings"))
			Expect(result.Stdout).To(ContainSubstring("Settings: 1 scope load error"))
			Expect(result.Stdout).To(ContainSubstring("Restore or delete the corrupted file:"))
			Expect(result.Stdout).To(ContainSubstring("settings.json"))
		})
	})

	Describe("multi-scope settings load errors", func() {
		It("reports the correct count and plural form when multiple scopes fail", func() {
			// Corrupt user-scope settings
			env.WriteFile(env.ClaudeDir, "settings.json", "{invalid json")

			// Corrupt project-scope settings
			projectDir := env.ProjectDir("multi-corrupt")
			claudeDir := filepath.Join(projectDir, ".claude")
			Expect(os.MkdirAll(claudeDir, 0755)).To(Succeed())
			env.WriteFile(claudeDir, "settings.json", "{invalid json")

			result := env.RunInDir(projectDir, "doctor")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("Checking Settings Scopes"))
			Expect(result.Stdout).To(ContainSubstring("user scope: failed to load settings"))
			Expect(result.Stdout).To(ContainSubstring("project scope: failed to load settings"))
			Expect(result.Stdout).To(ContainSubstring("Settings: 2 scope load errors"))
			Expect(result.Stdout).To(ContainSubstring("Plugin analysis may be incomplete: 2 scopes"))
		})
	})

	Describe("absent settings file", func() {
		It("does not warn when settings file is simply absent", func() {
			// No settings.json written — this is the normal case for fresh installs
			result := env.Run("doctor")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).NotTo(ContainSubstring("Checking Settings Scopes"))
			Expect(result.Stdout).NotTo(ContainSubstring("scope load error"))
			Expect(result.Stdout).NotTo(ContainSubstring("Settings:"))
			Expect(result.Stdout).NotTo(ContainSubstring("Plugin analysis may be incomplete"))
			Expect(result.Stdout).To(ContainSubstring("No issues detected!"))
		})
	})

	Describe("marketplace directory that cannot be accessed", func() {
		It("reports the OS error instead of a green checkmark", func() {
			// A regular file used as a path component makes os.Stat fail with
			// ENOTDIR, which is not fs.ErrNotExist -- exactly the class of error
			// an ErrNotExist-only check would report as healthy.
			env.WriteFile(env.ClaudeDir, "not-a-dir", "")
			badLocation := filepath.Join(env.ClaudeDir, "not-a-dir", "blocked-marketplace")
			env.CreateKnownMarketplaces(map[string]interface{}{
				"blocked-marketplace": map[string]interface{}{
					"source":          map[string]interface{}{"source": "github", "repo": "test/blocked"},
					"installLocation": badLocation,
					"lastUpdated":     "2026-01-01T00:00:00Z",
				},
			})

			result := env.Run("doctor")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("blocked-marketplace: Cannot access directory at " + badLocation))
			Expect(result.Stdout).NotTo(ContainSubstring("All marketplaces OK"))
			Expect(result.Stdout).To(ContainSubstring("Marketplaces: 1 installed, 1 issues"))
			Expect(result.Stdout).NotTo(ContainSubstring("No issues detected!"))
			Expect(result.Stdout).To(ContainSubstring("Run the suggested commands to fix these issues"))
		})
	})

	Describe("symlinks that cannot be resolved", func() {
		It("reports them as unchecked instead of claiming all symlinks are valid", func() {
			agentsDir := filepath.Join(env.ClaudeDir, "agents")
			Expect(os.MkdirAll(agentsDir, 0755)).To(Succeed())
			// A self-referencing symlink makes os.Stat and EvalSymlinks fail with
			// ELOOP, which is not fs.ErrNotExist, so it is neither valid nor broken.
			loop := filepath.Join(agentsDir, "loop.md")
			Expect(os.Symlink("loop.md", loop)).To(Succeed())

			result := env.Run("doctor")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("1 path could not be checked:"))
			Expect(result.Stdout).To(ContainSubstring(loop))
			Expect(result.Stdout).NotTo(ContainSubstring("All local symlinks are valid"))
			Expect(result.Stdout).NotTo(ContainSubstring("broken symlink"))
			Expect(result.Stdout).To(ContainSubstring("Symlinks: 0 broken, 1 unchecked"))
			Expect(result.Stdout).NotTo(ContainSubstring("No issues detected!"))
			Expect(result.Stdout).To(ContainSubstring("Run the suggested commands to fix these issues"))
		})
	})

	Describe("missing plugin recommendations", func() {
		BeforeEach(func() {
			// Create empty plugin registry (no plugins installed)
			env.CreateInstalledPlugins(map[string]interface{}{})
			// Enable a plugin in settings that is NOT installed
			env.CreateSettings(map[string]bool{
				"missing-plugin@test-marketplace": true,
			})
		})

		It("reports missing plugin with scope and install recommendation", func() {
			result := env.Run("doctor")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("1 plugin enabled but not installed"))
			Expect(result.Stdout).To(ContainSubstring("missing-plugin@test-marketplace"))
			Expect(result.Stdout).To(ContainSubstring("(user)"))
			Expect(result.Stdout).To(ContainSubstring("claude plugin install --scope <scope> <plugin-name>"))
			Expect(result.Stdout).To(ContainSubstring("claudeup profile clean --<scope> <plugin-name>"))
		})
	})
})
