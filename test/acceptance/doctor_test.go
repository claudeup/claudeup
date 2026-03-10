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
