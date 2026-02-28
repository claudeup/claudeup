// ABOUTME: Acceptance tests for nested profile discovery
// ABOUTME: Tests subdirectory profiles in list, disambiguation prompts, and path references
package acceptance

import (
	"github.com/claudeup/claudeup/v5/internal/profile"
	"github.com/claudeup/claudeup/v5/test/helpers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("nested profile discovery", func() {
	var env *helpers.TestEnv

	BeforeEach(func() {
		env = helpers.NewTestEnv(binaryPath)
		env.CreateClaudeSettings()
		env.CreateInstalledPlugins(map[string]interface{}{})
		env.CreateKnownMarketplaces(map[string]interface{}{})
	})

	Describe("profile list", func() {
		It("shows nested profiles grouped by prefix", func() {
			env.CreateNestedProfile("backend", &profile.Profile{
				Name:        "api",
				Description: "Backend API service",
			})

			result := env.Run("profile", "list")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("Your profiles"))
			Expect(result.Stdout).To(ContainSubstring("backend/"))
			Expect(result.Stdout).To(ContainSubstring("api"))
			Expect(result.Stdout).To(ContainSubstring("Backend API service"))
		})

		It("shows both root and nested profiles", func() {
			env.CreateProfile(&profile.Profile{
				Name:        "mobile",
				Description: "Mobile apps",
			})
			env.CreateNestedProfile("backend", &profile.Profile{
				Name:        "worker",
				Description: "Worker service",
			})

			result := env.Run("profile", "list")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("mobile"))
			Expect(result.Stdout).To(ContainSubstring("backend/"))
			Expect(result.Stdout).To(ContainSubstring("worker"))
		})

		It("shows deeply nested profiles under top-level group", func() {
			env.CreateNestedProfile("team/backend", &profile.Profile{
				Name:        "worker",
				Description: "Team worker profile",
			})

			result := env.Run("profile", "list")

			Expect(result.ExitCode).To(Equal(0))
			// Grouped by first path component; short name includes remaining path
			Expect(result.Stdout).To(ContainSubstring("team/"))
			Expect(result.Stdout).To(ContainSubstring("backend/worker"))
		})
	})

	Describe("name collision disambiguation", func() {
		BeforeEach(func() {
			env.CreateProfile(&profile.Profile{
				Name:        "api",
				Description: "Root API profile",
				Plugins:     []string{"root-plugin@marketplace"},
			})
			env.CreateNestedProfile("backend", &profile.Profile{
				Name:        "api",
				Description: "Backend API profile",
				Plugins:     []string{"backend-plugin@marketplace"},
			})
		})

		It("prompts for disambiguation in interactive mode", func() {
			// Select option 1 (root api)
			result := env.RunWithInput("1\n", "profile", "apply", "api")

			Expect(result.Stdout).To(ContainSubstring("Multiple profiles match"))
			Expect(result.Stdout).To(ContainSubstring("api"))
			Expect(result.Stdout).To(ContainSubstring("backend/api"))
		})

		It("errors with --yes flag on ambiguous name", func() {
			result := env.Run("profile", "apply", "api", "-y")

			Expect(result.ExitCode).NotTo(Equal(0))
			Expect(result.Stderr).To(ContainSubstring("ambiguous"))
			Expect(result.Stderr).To(ContainSubstring("backend/api"))
		})

		It("lists both profiles with their paths", func() {
			result := env.Run("profile", "list")

			Expect(result.ExitCode).To(Equal(0))
			// Root "api" appears ungrouped, nested "api" appears under backend/ group
			Expect(result.Stdout).To(ContainSubstring("api"))
			Expect(result.Stdout).To(ContainSubstring("backend/"))
		})
	})

	Describe("path reference", func() {
		BeforeEach(func() {
			env.CreateNestedProfile("backend", &profile.Profile{
				Name:        "api",
				Description: "Backend API profile",
				Plugins:     []string{"backend-plugin@marketplace"},
			})
		})

		It("applies a nested profile by path reference", func() {
			result := env.Run("profile", "apply", "backend/api", "-y")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("Profile applied"))
		})

		It("bypasses disambiguation when using path reference", func() {
			// Also create a root "api" profile to create ambiguity
			env.CreateProfile(&profile.Profile{
				Name:        "api",
				Description: "Root API profile",
				Plugins:     []string{"root-plugin@marketplace"},
			})

			// Path reference should resolve directly without prompting
			result := env.Run("profile", "apply", "backend/api", "-y")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("Profile applied"))
			Expect(result.Stdout).NotTo(ContainSubstring("Multiple profiles match"))
		})

		It("returns error for nonexistent path reference", func() {
			result := env.Run("profile", "apply", "backend/missing", "-y")

			Expect(result.ExitCode).NotTo(Equal(0))
			Expect(result.Stderr).To(ContainSubstring("not found"))
		})
	})
})
