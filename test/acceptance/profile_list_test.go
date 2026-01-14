// ABOUTME: Acceptance tests for profile list, delete, and restore commands
// ABOUTME: Tests built-in vs user profile grouping, customization indicators, deletion, and restoration
package acceptance

import (
	"github.com/claudeup/claudeup/v2/internal/profile"
	"github.com/claudeup/claudeup/v2/test/helpers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("profile list", func() {
	var env *helpers.TestEnv

	BeforeEach(func() {
		env = helpers.NewTestEnv(binaryPath)
	})

	Describe("profile grouping", func() {
		Context("with no user profiles", func() {
			It("shows only built-in profiles section", func() {
				result := env.Run("profile", "list")

				Expect(result.ExitCode).To(Equal(0))
				Expect(result.Stdout).To(ContainSubstring("Built-in profiles"))
				Expect(result.Stdout).NotTo(ContainSubstring("Your profiles"))
				// Should show built-in profiles without (customized)
				Expect(result.Stdout).To(ContainSubstring("default"))
				Expect(result.Stdout).To(ContainSubstring("frontend"))
				Expect(result.Stdout).To(ContainSubstring("hobson"))
			})
		})

		Context("with only custom user profiles", func() {
			BeforeEach(func() {
				env.CreateProfile(&profile.Profile{
					Name:        "my-custom-profile",
					Description: "A custom profile",
				})
			})

			It("shows both sections", func() {
				result := env.Run("profile", "list")

				Expect(result.ExitCode).To(Equal(0))
				Expect(result.Stdout).To(ContainSubstring("Built-in profiles"))
				Expect(result.Stdout).To(ContainSubstring("Your profiles"))
				Expect(result.Stdout).To(ContainSubstring("my-custom-profile"))
			})

			It("shows custom profile in Your profiles section", func() {
				result := env.Run("profile", "list")

				// Custom profile should be in Your profiles, not Built-in
				lines := splitLines(result.Stdout)
				yourProfilesIdx := findLineContaining(lines, "Your profiles")
				builtInIdx := findLineContaining(lines, "Built-in profiles")

				customIdx := findLineContaining(lines, "my-custom-profile")
				Expect(customIdx).To(BeNumerically(">", yourProfilesIdx))
				Expect(customIdx).To(BeNumerically(">", builtInIdx))
			})
		})

		Context("with customized built-in profile", func() {
			BeforeEach(func() {
				// Create a user profile with same name as built-in
				env.CreateProfile(&profile.Profile{
					Name:        "frontend",
					Description: "My customized frontend",
				})
			})

			It("shows built-in profile in Built-in section", func() {
				result := env.Run("profile", "list")

				Expect(result.ExitCode).To(Equal(0))
				Expect(result.Stdout).To(ContainSubstring("frontend"))
				Expect(result.Stdout).To(ContainSubstring("Built-in profiles"))
			})

			It("does not show customized built-in in Your profiles section", func() {
				result := env.Run("profile", "list")

				// Should only have Built-in section since frontend is the only profile
				// and it's a customized built-in
				Expect(result.Stdout).To(ContainSubstring("Built-in profiles"))
				Expect(result.Stdout).NotTo(ContainSubstring("Your profiles"))
			})
		})

		Context("with mix of custom and customized built-in profiles", func() {
			BeforeEach(func() {
				// Customized built-in
				env.CreateProfile(&profile.Profile{
					Name:        "default",
					Description: "My customized default",
				})
				// Truly custom profile
				env.CreateProfile(&profile.Profile{
					Name:        "my-workflow",
					Description: "My custom workflow",
				})
			})

			It("shows built-in profiles in Built-in section", func() {
				result := env.Run("profile", "list")

				Expect(result.ExitCode).To(Equal(0))
				// default should be in Built-in section
				Expect(result.Stdout).To(ContainSubstring("Built-in profiles"))
				Expect(result.Stdout).To(ContainSubstring("default"))
			})

			It("shows custom profile in Your profiles section", func() {
				result := env.Run("profile", "list")

				Expect(result.ExitCode).To(Equal(0))
				Expect(result.Stdout).To(ContainSubstring("Your profiles"))
				Expect(result.Stdout).To(ContainSubstring("my-workflow"))
			})

			It("orders sections correctly: Built-in first, then Your profiles", func() {
				result := env.Run("profile", "list")

				lines := splitLines(result.Stdout)
				builtInIdx := findLineContaining(lines, "Built-in profiles")
				yourIdx := findLineContaining(lines, "Your profiles")

				Expect(builtInIdx).To(BeNumerically("<", yourIdx))
			})
		})
	})

	Describe("active profile indicator", func() {
		BeforeEach(func() {
			env.CreateProfile(&profile.Profile{
				Name:        "my-profile",
				Description: "Test profile",
			})
			env.SetActiveProfile("my-profile")
		})

		It("marks active profile with asterisk", func() {
			result := env.Run("profile", "list")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(MatchRegexp(`\*\s+my-profile`))
		})

		It("updates active profile marker even when no changes needed", func() {
			// Create two empty profiles (same content)
			env.CreateProfile(&profile.Profile{
				Name:        "profile-a",
				Description: "Empty profile A",
			})
			env.CreateProfile(&profile.Profile{
				Name:        "profile-b",
				Description: "Empty profile B",
			})

			// Set profile-a as active
			env.SetActiveProfile("profile-a")

			// Verify profile-a is marked as active
			result := env.Run("profile", "list")
			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(MatchRegexp(`\*\s+profile-a`))
			Expect(result.Stdout).NotTo(MatchRegexp(`\*\s+profile-b`))

			// Use profile-b (should say "No changes needed" since both are empty)
			result = env.RunWithInput("y\n", "profile", "apply", "profile-b")
			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("No changes needed"))

			// Verify active profile marker moved to profile-b
			result = env.Run("profile", "list")
			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(MatchRegexp(`\*\s+profile-b`))
			Expect(result.Stdout).NotTo(MatchRegexp(`\*\s+profile-a`))
		})

		Context("with project-scoped profile", func() {
			var projectDir string

			BeforeEach(func() {
				// Create a project directory with .claudeup.json
				projectDir = env.ProjectDir("my-project")

				// Set user-level active profile to "default"
				env.SetActiveProfile("default")

				// Create project-scoped profile config with different profile
				env.CreateClaudeupJSON(projectDir, map[string]interface{}{
					"version": "1",
					"profile": "frontend",
				})
			})

			It("marks project-scoped profile as active when run from project directory", func() {
				result := env.RunInDir(projectDir, "profile", "list")

				Expect(result.ExitCode).To(Equal(0))
				// Should mark "frontend" (project scope) as active, not "default" (user scope)
				Expect(result.Stdout).To(MatchRegexp(`\*\s+frontend`))
				Expect(result.Stdout).NotTo(MatchRegexp(`\*\s+default`))
			})

			It("marks user-level profile as active when run from non-project directory", func() {
				// Run from temp root (no .claudeup.json)
				result := env.Run("profile", "list")

				Expect(result.ExitCode).To(Equal(0))
				// Should mark "default" (user scope) as active since no project config
				Expect(result.Stdout).To(MatchRegexp(`\*\s+default`))
			})
		})

		Context("with local-scoped profile (projects registry)", func() {
			var projectDir string

			BeforeEach(func() {
				// Create a project directory WITHOUT .claudeup.json
				projectDir = env.ProjectDir("local-project")

				// Set user-level active profile to "default"
				env.SetActiveProfile("default")

				// Register project in projects.json with different profile (local scope)
				env.RegisterProject(projectDir, "frontend")
			})

			It("marks local-scoped profile as active when run from registered project directory", func() {
				result := env.RunInDir(projectDir, "profile", "list")

				Expect(result.ExitCode).To(Equal(0))
				// Should mark "frontend" (local scope) as active, not "default" (user scope)
				Expect(result.Stdout).To(MatchRegexp(`\*\s+frontend`))
				Expect(result.Stdout).NotTo(MatchRegexp(`\*\s+default`))
			})

			It("prefers project scope over local scope", func() {
				// Add a .claudeup.json that specifies a different profile
				env.CreateClaudeupJSON(projectDir, map[string]interface{}{
					"version": "1",
					"profile": "hobson",
				})

				result := env.RunInDir(projectDir, "profile", "list")

				Expect(result.ExitCode).To(Equal(0))
				// Should mark "hobson" (project scope) as active, not "frontend" (local scope)
				Expect(result.Stdout).To(MatchRegexp(`\*\s+hobson`))
				Expect(result.Stdout).NotTo(MatchRegexp(`\*\s+frontend`))
			})
		})
	})

	Describe("reserved name warning", func() {
		Context("when a profile named 'current' exists", func() {
			BeforeEach(func() {
				env.CreateProfile(&profile.Profile{
					Name:        "current",
					Description: "Old profile with reserved name",
				})
			})

			It("shows a warning about the reserved name", func() {
				result := env.Run("profile", "list")

				Expect(result.ExitCode).To(Equal(0))
				Expect(result.Stdout).To(ContainSubstring("current"))
				Expect(result.Stdout).To(ContainSubstring("reserved"))
				Expect(result.Stdout).To(ContainSubstring("profile rename"))
			})
		})

		Context("when no profile named 'current' exists", func() {
			BeforeEach(func() {
				env.CreateProfile(&profile.Profile{
					Name:        "my-profile",
					Description: "Normal profile",
				})
			})

			It("does not show a warning", func() {
				result := env.Run("profile", "list")

				Expect(result.ExitCode).To(Equal(0))
				Expect(result.Stdout).NotTo(ContainSubstring("reserved"))
			})
		})
	})

	Describe("scope flag", func() {
		BeforeEach(func() {
			// Create some test profiles with distinct names that won't match as substrings
			env.CreateProfile(&profile.Profile{
				Name:        "my-active-profile",
				Description: "This profile will be active",
			})
			env.CreateProfile(&profile.Profile{
				Name:        "other-profile",
				Description: "This profile is not active",
			})
			// Set one profile as active at user scope
			env.SetActiveProfile("my-active-profile")
		})

		Context("with --scope user", func() {
			It("shows only the profile active at user scope", func() {
				result := env.Run("profile", "list", "--scope", "user")

				Expect(result.ExitCode).To(Equal(0))
				Expect(result.Stdout).To(ContainSubstring("my-active-profile"))
				Expect(result.Stdout).NotTo(ContainSubstring("other-profile"))
				Expect(result.Stdout).To(ContainSubstring("Showing profile active at:"))
			})
		})

		Context("with --scope project", func() {
			var projectDir string
			BeforeEach(func() {
				projectDir = env.ProjectDir("project-scope-test")
				// Create project settings for "hobson"
				// 1. profile list --scope project checks for .claude/settings.json as a pre-flight
				settingsPath := filepath.Join(projectDir, ".claude", "settings.json")
				helpers.WriteJSON(settingsPath, map[string]interface{}{
					"profile": "hobson",
				})
				// 2. getAllActiveProfiles checks for .claudeup.json via profile.ProjectConfigExists
				env.CreateClaudeupJSON(projectDir, map[string]interface{}{
					"version": "1",
					"profile": "hobson",
				})
			})

			It("shows only the profile active at project scope", func() {
				result := env.RunInDir(projectDir, "profile", "list", "--scope", "project")

				Expect(result.ExitCode).To(Equal(0))
				Expect(result.Stdout).To(ContainSubstring("hobson"))
				Expect(result.Stdout).NotTo(ContainSubstring("my-active-profile"))
				Expect(result.Stdout).To(ContainSubstring("Showing profile active at: project"))
			})
		})

		Context("with invalid scope value", func() {
			It("returns an error", func() {
				result := env.Run("profile", "list", "--scope", "invalid-value")

				Expect(result.ExitCode).NotTo(Equal(0))
				Expect(result.Stderr).To(ContainSubstring("invalid scope"))
			})
		})

		Context("when filtering by local scope without local settings file", func() {
			It("shows a warning about missing file", func() {
				// Test env runs in temp directory without .claude/settings.local.json
				result := env.Run("profile", "list", "--scope", "local")

				Expect(result.ExitCode).To(Equal(0))
				Expect(result.Stdout).To(ContainSubstring("No .claude/settings.local.json found"))
			})
		})

		Context("with customized built-in profile at user scope", func() {
			BeforeEach(func() {
				// Shadow "frontend" built-in
				env.CreateProfile(&profile.Profile{
					Name:        "frontend",
					Description: "Customized frontend description",
				})
				// Make it active at user scope
				env.SetActiveProfile("frontend")
			})

			It("shows customized built-in only once with custom description and indicator", func() {
				result := env.Run("profile", "list", "--scope", "user")

				Expect(result.ExitCode).To(Equal(0))
				Expect(result.Stdout).To(ContainSubstring("frontend"))
				Expect(result.Stdout).To(ContainSubstring("Customized frontend description"))
				Expect(result.Stdout).To(ContainSubstring("(customized)"))

				// Verify it's not duplicated in Your profiles (which it shouldn't be anyway)
				// but more importantly ensure it's not shown twice due to filtering logic
				lines := splitLines(result.Stdout)
				count := 0
				for _, line := range lines {
					if strings.Contains(line, "frontend") {
						count++
					}
				}
				Expect(count).To(Equal(1), "frontend profile should appear only once in the output")
			})
		})
	})
})


var _ = Describe("profile delete", func() {
	var env *helpers.TestEnv

	BeforeEach(func() {
		env = helpers.NewTestEnv(binaryPath)
	})

	Context("deleting a custom profile", func() {
		BeforeEach(func() {
			env.CreateProfile(&profile.Profile{
				Name:        "my-custom",
				Description: "A custom profile",
			})
		})

		It("removes the profile file", func() {
			Expect(env.ProfileExists("my-custom")).To(BeTrue())

			result := env.RunWithInput("y\n", "profile", "delete", "my-custom")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("Deleted profile"))
			Expect(env.ProfileExists("my-custom")).To(BeFalse())
		})

		It("shows permanent removal warning", func() {
			result := env.RunWithInput("n\n", "profile", "delete", "my-custom")

			Expect(result.Stdout).To(ContainSubstring("permanently remove"))
		})
	})

	Context("trying to delete a built-in profile", func() {
		It("returns error for unmodified built-in", func() {
			result := env.Run("profile", "delete", "hobson")

			Expect(result.ExitCode).NotTo(Equal(0))
			Expect(result.Stderr).To(ContainSubstring("cannot be deleted"))
		})

		It("suggests using restore for customized built-in", func() {
			env.CreateProfile(&profile.Profile{
				Name:        "frontend",
				Description: "My customized frontend",
			})

			result := env.Run("profile", "delete", "frontend")

			Expect(result.ExitCode).NotTo(Equal(0))
			Expect(result.Stderr).To(ContainSubstring("profile restore"))
		})
	})

	Context("deleting a non-existent profile", func() {
		It("returns error for unknown profile", func() {
			result := env.Run("profile", "delete", "nonexistent")

			Expect(result.ExitCode).NotTo(Equal(0))
			Expect(result.Stderr).To(ContainSubstring("not found"))
		})
	})

	Context("with -y flag", func() {
		BeforeEach(func() {
			env.CreateProfile(&profile.Profile{
				Name:        "auto-delete",
				Description: "Will be auto-deleted",
			})
		})

		It("skips confirmation prompt", func() {
			result := env.Run("profile", "delete", "auto-delete", "-y")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("Deleted profile"))
			Expect(env.ProfileExists("auto-delete")).To(BeFalse())
		})
	})

	Context("deleting the currently active profile", func() {
		BeforeEach(func() {
			env.CreateProfile(&profile.Profile{
				Name:        "active-profile",
				Description: "Currently active",
			})
			env.SetActiveProfile("active-profile")
		})

		It("shows special warning about deleting active profile", func() {
			result := env.RunWithInput("y\n", "profile", "delete", "active-profile")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("currently active"))
		})

		It("tells user to run 'profile apply' after deletion", func() {
			result := env.RunWithInput("y\n", "profile", "delete", "active-profile")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("profile apply"))
		})

		It("clears the active profile from config", func() {
			result := env.RunWithInput("y\n", "profile", "delete", "active-profile")

			Expect(result.ExitCode).To(Equal(0))
			Expect(env.GetActiveProfile()).To(BeEmpty())
		})
	})
})

var _ = Describe("profile restore", func() {
	var env *helpers.TestEnv

	BeforeEach(func() {
		env = helpers.NewTestEnv(binaryPath)
	})

	Context("restoring a customized built-in profile", func() {
		BeforeEach(func() {
			env.CreateProfile(&profile.Profile{
				Name:        "frontend",
				Description: "My customized frontend",
			})
		})

		It("removes the customization file", func() {
			Expect(env.ProfileExists("frontend")).To(BeTrue())

			result := env.RunWithInput("y\n", "profile", "restore", "frontend")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("Restored built-in profile"))
			Expect(env.ProfileExists("frontend")).To(BeFalse())
		})

		It("shows restore message", func() {
			result := env.RunWithInput("n\n", "profile", "restore", "frontend")

			Expect(result.Stdout).To(ContainSubstring("restore the original built-in"))
		})
	})

	Context("trying to restore a non-built-in profile", func() {
		BeforeEach(func() {
			env.CreateProfile(&profile.Profile{
				Name:        "my-custom",
				Description: "A custom profile",
			})
		})

		It("returns error and suggests delete", func() {
			result := env.Run("profile", "restore", "my-custom")

			Expect(result.ExitCode).NotTo(Equal(0))
			Expect(result.Stderr).To(ContainSubstring("not a built-in profile"))
			Expect(result.Stderr).To(ContainSubstring("profile delete"))
		})
	})

	Context("restoring an unmodified built-in", func() {
		It("returns error", func() {
			result := env.Run("profile", "restore", "hobson")

			Expect(result.ExitCode).NotTo(Equal(0))
			Expect(result.Stderr).To(ContainSubstring("no customizations"))
		})
	})

	Context("with -y flag", func() {
		BeforeEach(func() {
			env.CreateProfile(&profile.Profile{
				Name:        "default",
				Description: "My customized default",
			})
		})

		It("skips confirmation prompt", func() {
			result := env.Run("profile", "restore", "default", "-y")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("Restored built-in profile"))
			Expect(env.ProfileExists("default")).To(BeFalse())
		})
	})
})

// Helper functions for parsing output
func splitLines(s string) []string {
	var lines []string
	for _, line := range []byte(s) {
		if line == '\n' {
			lines = append(lines, "")
		}
	}
	// Simple split - just use the string directly for matching
	result := make([]string, 0)
	start := 0
	for i, c := range s {
		if c == '\n' {
			result = append(result, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		result = append(result, s[start:])
	}
	return result
}

func findLineContaining(lines []string, substr string) int {
	for i, line := range lines {
		if contains(line, substr) {
			return i
		}
	}
	return -1
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && findSubstring(s, substr) >= 0
}

func findSubstring(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
