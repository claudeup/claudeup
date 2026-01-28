// ABOUTME: Acceptance tests for sandbox profile auto-detection
// ABOUTME: Tests that sandbox reads .claudeup.json for automatic profile selection
package acceptance

import (
	"os"
	"path/filepath"

	"github.com/claudeup/claudeup/v3/internal/profile"
	"github.com/claudeup/claudeup/v3/test/helpers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Sandbox Auto-Detection", func() {
	var env *helpers.TestEnv
	var projectDir string

	BeforeEach(func() {
		env = helpers.NewTestEnv(binaryPath)
		projectDir = filepath.Join(env.TempDir, "myproject")
		Expect(os.MkdirAll(projectDir, 0755)).To(Succeed())
	})

	Describe("profile detection from .claudeup.json", func() {
		BeforeEach(func() {
			env.CreateProfile(&profile.Profile{
				Name:        "test-profile",
				Description: "Test profile for sandbox",
			})
		})

		It("prints auto-detection message when .claudeup.json exists", func() {
			env.CreateClaudeupJSON(projectDir, map[string]interface{}{
				"version":   "1",
				"profile":   "test-profile",
				"appliedAt": "2026-01-02T00:00:00Z",
			})

			result := env.RunInDir(projectDir, "sandbox", "--shell")
			Expect(result.Combined()).To(ContainSubstring("Using profile 'test-profile' from .claudeup.json"))
		})

		It("uses explicit --profile over .claudeup.json", func() {
			env.CreateClaudeupJSON(projectDir, map[string]interface{}{
				"version":   "1",
				"profile":   "test-profile",
				"appliedAt": "2026-01-02T00:00:00Z",
			})

			env.CreateProfile(&profile.Profile{
				Name:        "other-profile",
				Description: "Other profile",
			})

			result := env.RunInDir(projectDir, "sandbox", "--profile", "other-profile", "--shell")
			Expect(result.Combined()).NotTo(ContainSubstring("from .claudeup.json"))
		})

		It("skips detection with --ephemeral", func() {
			env.CreateClaudeupJSON(projectDir, map[string]interface{}{
				"version":   "1",
				"profile":   "test-profile",
				"appliedAt": "2026-01-02T00:00:00Z",
			})

			result := env.RunInDir(projectDir, "sandbox", "--ephemeral", "--shell")
			Expect(result.Combined()).NotTo(ContainSubstring("from .claudeup.json"))
			Expect(result.Combined()).To(ContainSubstring("ephemeral"))
		})

		It("runs ephemeral when no .claudeup.json exists", func() {
			result := env.RunInDir(projectDir, "sandbox", "--shell")
			Expect(result.Combined()).NotTo(ContainSubstring("from .claudeup.json"))
			Expect(result.Combined()).To(ContainSubstring("ephemeral"))
		})

		It("errors on malformed .claudeup.json", func() {
			configPath := filepath.Join(projectDir, ".claudeup.json")
			Expect(os.WriteFile(configPath, []byte("not valid json"), 0644)).To(Succeed())

			result := env.RunInDir(projectDir, "sandbox", "--shell")
			Expect(result.ExitCode).NotTo(Equal(0))
			Expect(result.Combined()).To(ContainSubstring("invalid"))
		})

		It("errors when profile referenced in .claudeup.json doesn't exist", func() {
			env.CreateClaudeupJSON(projectDir, map[string]interface{}{
				"version":   "1",
				"profile":   "nonexistent",
				"appliedAt": "2026-01-02T00:00:00Z",
			})

			result := env.RunInDir(projectDir, "sandbox", "--shell")
			Expect(result.ExitCode).NotTo(Equal(0))
			Expect(result.Combined()).To(ContainSubstring("failed to load profile"))
		})
	})
})
