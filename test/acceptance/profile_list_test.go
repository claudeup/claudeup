// ABOUTME: Acceptance tests for profile list, delete, and restore commands
// ABOUTME: Tests built-in vs user profile grouping, customization indicators, deletion, and restoration
package acceptance

import (
	"strings"

	"github.com/claudeup/claudeup/v5/internal/profile"
	"github.com/claudeup/claudeup/v5/test/helpers"
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

			It("shows customized indicator for built-in profile in default view", func() {
				result := env.Run("profile", "list")

				Expect(result.ExitCode).To(Equal(0))
				Expect(result.Stdout).To(ContainSubstring("(customized)"))
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

	Describe("path prefix grouping", func() {
		Context("with nested profiles under path prefixes", func() {
			BeforeEach(func() {
				// Ungrouped profile (no prefix)
				env.CreateProfile(&profile.Profile{
					Name:        "base",
					Description: "base marketplaces and plugins",
				})
				// Profiles under languages/ prefix
				env.CreateNestedProfile("languages", &profile.Profile{
					Name:        "go",
					Description: "Go language development",
				})
				env.CreateNestedProfile("languages", &profile.Profile{
					Name:        "python",
					Description: "Python development",
				})
				// Profiles under tools/ prefix
				env.CreateNestedProfile("tools", &profile.Profile{
					Name:        "conductor",
					Description: "my conductor settings",
				})
			})

			It("shows group headers for prefixed profiles", func() {
				result := env.Run("profile", "list")

				Expect(result.ExitCode).To(Equal(0))
				Expect(result.Stdout).To(ContainSubstring("languages/"))
				Expect(result.Stdout).To(ContainSubstring("tools/"))
			})

			It("shows short names under group headers", func() {
				result := env.Run("profile", "list")

				lines := splitLines(result.Stdout)
				langIdx := findLineContaining(lines, "languages/")
				Expect(langIdx).To(BeNumerically(">=", 0), "languages/ header not found")

				// "go" and "python" should appear after the languages/ header
				goIdx := findLineContainingAfter(lines, langIdx+1, "go")
				Expect(goIdx).To(BeNumerically(">", langIdx))

				pythonIdx := findLineContainingAfter(lines, langIdx+1, "python")
				Expect(pythonIdx).To(BeNumerically(">", langIdx))
			})

			It("shows ungrouped profiles before grouped sections", func() {
				result := env.Run("profile", "list")

				lines := splitLines(result.Stdout)
				yourIdx := findLineContaining(lines, "Your profiles")
				Expect(yourIdx).To(BeNumerically(">=", 0))

				baseIdx := findLineContainingAfter(lines, yourIdx, "base")
				langIdx := findLineContaining(lines, "languages/")
				toolsIdx := findLineContaining(lines, "tools/")

				Expect(baseIdx).To(BeNumerically("<", langIdx),
					"ungrouped profile 'base' should appear before 'languages/' group")
				Expect(baseIdx).To(BeNumerically("<", toolsIdx),
					"ungrouped profile 'base' should appear before 'tools/' group")
			})

			It("sorts groups alphabetically", func() {
				result := env.Run("profile", "list")

				lines := splitLines(result.Stdout)
				langIdx := findLineContaining(lines, "languages/")
				toolsIdx := findLineContaining(lines, "tools/")

				Expect(langIdx).To(BeNumerically("<", toolsIdx),
					"'languages/' group should appear before 'tools/' group")
			})
		})
	})

	Describe("hidden profile filtering", func() {
		Context("with underscore-prefixed profiles", func() {
			BeforeEach(func() {
				env.CreateProfile(&profile.Profile{
					Name:        "my-visible",
					Description: "A visible profile",
				})
				env.CreateProfile(&profile.Profile{
					Name:        "_lab-snapshot-1",
					Description: "Lab snapshot",
				})
				env.CreateProfile(&profile.Profile{
					Name:        "_lab-snapshot-2",
					Description: "Another lab snapshot",
				})
			})

			It("hides underscore-prefixed profiles by default", func() {
				result := env.Run("profile", "list")

				Expect(result.ExitCode).To(Equal(0))
				Expect(result.Stdout).To(ContainSubstring("my-visible"))
				Expect(result.Stdout).NotTo(ContainSubstring("_lab-snapshot-1"))
				Expect(result.Stdout).NotTo(ContainSubstring("_lab-snapshot-2"))
			})

			It("shows hidden profile count", func() {
				result := env.Run("profile", "list")

				Expect(result.ExitCode).To(Equal(0))
				Expect(result.Stdout).To(ContainSubstring("2 hidden"))
				Expect(result.Stdout).To(ContainSubstring("--all"))
			})

			It("shows all profiles with --all flag", func() {
				result := env.Run("profile", "list", "--all")

				Expect(result.ExitCode).To(Equal(0))
				Expect(result.Stdout).To(ContainSubstring("my-visible"))
				Expect(result.Stdout).To(ContainSubstring("_lab-snapshot-1"))
				Expect(result.Stdout).To(ContainSubstring("_lab-snapshot-2"))
			})

			It("does not show hidden count with --all flag", func() {
				result := env.Run("profile", "list", "--all")

				Expect(result.ExitCode).To(Equal(0))
				Expect(result.Stdout).NotTo(ContainSubstring("hidden"))
			})
		})

		Context("with no hidden profiles", func() {
			BeforeEach(func() {
				env.CreateProfile(&profile.Profile{
					Name:        "visible-only",
					Description: "A regular profile",
				})
			})

			It("does not show hidden count message", func() {
				result := env.Run("profile", "list")

				Expect(result.ExitCode).To(Equal(0))
				Expect(result.Stdout).NotTo(ContainSubstring("hidden"))
			})
		})

		Context("with underscore-prefixed nested profiles", func() {
			BeforeEach(func() {
				env.CreateNestedProfile("tools", &profile.Profile{
					Name:        "_internal-tool",
					Description: "Hidden nested tool",
				})
				env.CreateNestedProfile("tools", &profile.Profile{
					Name:        "visible-tool",
					Description: "Visible tool",
				})
			})

			It("hides nested underscore-prefixed profiles by default", func() {
				result := env.Run("profile", "list")

				Expect(result.ExitCode).To(Equal(0))
				Expect(result.Stdout).To(ContainSubstring("visible-tool"))
				Expect(result.Stdout).NotTo(ContainSubstring("_internal-tool"))
			})

			It("shows nested hidden profiles with --all", func() {
				result := env.Run("profile", "list", "--all")

				Expect(result.ExitCode).To(Equal(0))
				Expect(result.Stdout).To(ContainSubstring("_internal-tool"))
				Expect(result.Stdout).To(ContainSubstring("visible-tool"))
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

	Describe("footer hints", func() {
		It("shows profile status command in footer", func() {
			result := env.Run("profile", "list")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("claudeup profile status"))
			Expect(result.Stdout).To(ContainSubstring("claudeup profile show <name>"))
			Expect(result.Stdout).To(ContainSubstring("claudeup profile apply <name>"))
		})
	})
})

// Helper functions for parsing output
func splitLines(s string) []string {
	return strings.Split(s, "\n")
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

// findLineContainingAfter finds the first line containing substr starting from startIdx
func findLineContainingAfter(lines []string, startIdx int, substr string) int {
	for i := startIdx; i < len(lines); i++ {
		if contains(lines[i], substr) {
			return i
		}
	}
	return -1
}
