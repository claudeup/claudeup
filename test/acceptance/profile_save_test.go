// ABOUTME: Acceptance tests for profile save command
// ABOUTME: Tests CLI behavior for saving profiles from current state
package acceptance

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/claudeup/claudeup/v5/internal/profile"
	"github.com/claudeup/claudeup/v5/test/helpers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("profile save", func() {
	var env *helpers.TestEnv

	BeforeEach(func() {
		env = helpers.NewTestEnv(binaryPath)
		env.CreateClaudeSettings()
	})

	Context("with a new profile name", func() {
		It("creates the profile from current state", func() {
			result := env.Run("profile", "save", "my-profile")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("Saved profile"))
			Expect(env.ProfileExists("my-profile")).To(BeTrue())
		})

		It("marks the saved profile as active", func() {
			result := env.Run("profile", "save", "my-profile")

			Expect(result.ExitCode).To(Equal(0))
			Expect(env.GetActiveProfile()).To(Equal("my-profile"))
		})
	})

	Context("with an existing profile name", func() {
		BeforeEach(func() {
			env.CreateProfile(&profile.Profile{
				Name:        "existing",
				Description: "Existing profile",
			})
		})

		It("prompts for confirmation and cancels on 'n'", func() {
			result := env.RunWithInput("n\n", "profile", "save", "existing")

			Expect(result.Stdout).To(ContainSubstring("Overwrite?"))
			Expect(result.Stdout).To(ContainSubstring("Cancelled"))
		})

		It("overwrites when user confirms with 'y'", func() {
			result := env.RunWithInput("y\n", "profile", "save", "existing")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("Saved profile"))
		})

		It("overwrites without prompting when -y flag is used", func() {
			result := env.Run("profile", "save", "existing", "-y")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("Saved profile"))
		})

		It("marks the overwritten profile as active", func() {
			result := env.RunWithInput("y\n", "profile", "save", "existing")

			Expect(result.ExitCode).To(Equal(0))
			Expect(env.GetActiveProfile()).To(Equal("existing"))
		})
	})

	Context("without a profile name", func() {
		Context("when an active profile is set", func() {
			BeforeEach(func() {
				env.CreateProfile(&profile.Profile{Name: "active-one"})
				env.SetActiveProfile("active-one")
			})

			It("saves to the active profile", func() {
				result := env.Run("profile", "save")

				Expect(result.ExitCode).To(Equal(0))
				Expect(result.Stdout).To(ContainSubstring("Saving to active profile"))
				Expect(result.Stdout).To(ContainSubstring("active-one"))
			})

			It("keeps the profile as active", func() {
				result := env.Run("profile", "save")

				Expect(result.ExitCode).To(Equal(0))
				Expect(env.GetActiveProfile()).To(Equal("active-one"))
			})

			It("does not prompt for overwrite when saving to active profile", func() {
				result := env.Run("profile", "save")

				Expect(result.ExitCode).To(Equal(0))
				Expect(result.Stdout).NotTo(ContainSubstring("Overwrite?"))
				Expect(result.Stdout).To(ContainSubstring("Saved profile"))
			})
		})

		Context("when no active profile is set", func() {
			It("returns an error", func() {
				result := env.Run("profile", "save")

				Expect(result.ExitCode).NotTo(Equal(0))
				Expect(result.Stderr).To(ContainSubstring("no profile name"))
			})
		})
	})

	Describe("profile save --scope", func() {
		Context("with --project flag", func() {
			var projectDir string

			BeforeEach(func() {
				projectDir = env.ProjectDir("save-scope-project")

				// Create project-scope settings with plugins
				env.CreateProjectScopeSettings(projectDir, map[string]bool{
					"plugin-a@marketplace": true,
					"plugin-b@marketplace": true,
				})

				// Create user-scope settings so we can verify they are excluded
				env.CreateSettings(map[string]bool{
					"user-plugin@marketplace": true,
				})
			})

			It("saves only project scope settings", func() {
				result := env.RunInDir(projectDir, "profile", "save", "project-only", "--project")

				Expect(result.ExitCode).To(Equal(0))

				// Load the saved profile JSON and verify it only has project scope
				profilePath := filepath.Join(env.ProfilesDir, "project-only.json")
				data, err := os.ReadFile(profilePath)
				Expect(err).NotTo(HaveOccurred())

				var p map[string]any
				Expect(json.Unmarshal(data, &p)).To(Succeed())

				perScope, ok := p["perScope"].(map[string]any)
				Expect(ok).To(BeTrue(), "expected perScope field in profile")

				// Project scope should exist with plugins
				projectScope, ok := perScope["project"].(map[string]any)
				Expect(ok).To(BeTrue(), "expected project scope in perScope")
				projectPlugins := projectScope["plugins"].([]any)
				Expect(projectPlugins).To(HaveLen(2))

				// User scope should NOT exist
				_, hasUser := perScope["user"]
				Expect(hasUser).To(BeFalse(), "user scope should not exist in project-only profile")

				// Local scope should NOT exist
				_, hasLocal := perScope["local"]
				Expect(hasLocal).To(BeFalse(), "local scope should not exist in project-only profile")
			})
		})

		Context("with --user flag", func() {
			var projectDir string

			BeforeEach(func() {
				projectDir = env.ProjectDir("save-scope-user")

				// Create user-scope settings
				env.CreateSettings(map[string]bool{
					"user-plugin@marketplace": true,
				})

				// Create project-scope settings so we can verify they are excluded
				env.CreateProjectScopeSettings(projectDir, map[string]bool{
					"project-plugin@marketplace": true,
				})
			})

			It("saves only user scope settings", func() {
				result := env.RunInDir(projectDir, "profile", "save", "user-only", "--user")

				Expect(result.ExitCode).To(Equal(0))

				// Load the saved profile JSON
				profilePath := filepath.Join(env.ProfilesDir, "user-only.json")
				data, err := os.ReadFile(profilePath)
				Expect(err).NotTo(HaveOccurred())

				var p map[string]any
				Expect(json.Unmarshal(data, &p)).To(Succeed())

				perScope, ok := p["perScope"].(map[string]any)
				Expect(ok).To(BeTrue(), "expected perScope field in profile")

				// User scope should exist
				_, hasUser := perScope["user"]
				Expect(hasUser).To(BeTrue(), "user scope should exist in user-only profile")

				// Project scope should NOT exist
				_, hasProject := perScope["project"]
				Expect(hasProject).To(BeFalse(), "project scope should not exist in user-only profile")

				// Local scope should NOT exist
				_, hasLocal := perScope["local"]
				Expect(hasLocal).To(BeFalse(), "local scope should not exist in user-only profile")
			})
		})

		Context("without scope flag", func() {
			var projectDir string

			BeforeEach(func() {
				projectDir = env.ProjectDir("save-all-scopes")

				env.CreateProjectScopeSettings(projectDir, map[string]bool{
					"project-plugin@marketplace": true,
				})
				env.CreateSettings(map[string]bool{
					"user-plugin@marketplace": true,
				})
			})

			It("saves all scopes (default behavior)", func() {
				result := env.RunInDir(projectDir, "profile", "save", "all-scopes")

				Expect(result.ExitCode).To(Equal(0))

				// Load the saved profile JSON
				profilePath := filepath.Join(env.ProfilesDir, "all-scopes.json")
				data, err := os.ReadFile(profilePath)
				Expect(err).NotTo(HaveOccurred())

				var p map[string]any
				Expect(json.Unmarshal(data, &p)).To(Succeed())

				perScope, ok := p["perScope"].(map[string]any)
				Expect(ok).To(BeTrue(), "expected perScope field in profile")

				// Both user and project scopes should exist
				_, hasUser := perScope["user"]
				Expect(hasUser).To(BeTrue(), "user scope should exist when no scope flag is provided")

				_, hasProject := perScope["project"]
				Expect(hasProject).To(BeTrue(), "project scope should exist when no scope flag is provided")
			})
		})

		Context("with multiple scope flags", func() {
			It("returns an error", func() {
				result := env.Run("profile", "save", "bad-save", "--user", "--project")

				Expect(result.ExitCode).NotTo(Equal(0))
				Expect(result.Stderr).To(ContainSubstring("cannot specify multiple scope flags"))
			})
		})
	})

	Describe("--apply flag", func() {
		It("saves and tracks the profile at project scope", func() {
			projectDir := env.ProjectDir("save-apply-project")
			env.CreateProjectScopeSettings(projectDir, map[string]bool{
				"plugin-a@marketplace": true,
				"plugin-b@marketplace": true,
			})

			result := env.RunInDir(projectDir, "profile", "save", "my-proj", "--project", "--apply")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("Saved and applied"))
			Expect(result.Stdout).To(ContainSubstring("my-proj"))

			// Verify the profile is tracked in the projects registry at project scope
			registry := env.LoadProjectsRegistry()
			Expect(registry).NotTo(BeNil())
			projects := registry["projects"].(map[string]any)
			// Normalize path to handle macOS /var -> /private/var symlinks
			normalizedDir := projectDir
			if resolved, err := filepath.EvalSymlinks(projectDir); err == nil {
				normalizedDir = resolved
			}
			entry := projects[normalizedDir].(map[string]any)
			Expect(entry["projectProfile"]).To(Equal("my-proj"))
		})

		It("shows profile as active in list after save --apply", func() {
			projectDir := env.ProjectDir("save-apply-list")
			env.CreateProjectScopeSettings(projectDir, map[string]bool{
				"plugin-a@marketplace": true,
			})

			result := env.RunInDir(projectDir, "profile", "save", "my-proj", "--project", "--apply")
			Expect(result.ExitCode).To(Equal(0))

			listResult := env.RunInDir(projectDir, "profile", "list")
			Expect(listResult.ExitCode).To(Equal(0))
			Expect(listResult.Stdout).To(ContainSubstring("my-proj"))
		})

		It("composes with --yes flag for non-interactive operation", func() {
			projectDir := env.ProjectDir("save-apply-yes")
			env.CreateProjectScopeSettings(projectDir, map[string]bool{
				"plugin-a@marketplace": true,
			})

			// Create profile first so overwrite prompt would trigger
			env.RunInDir(projectDir, "profile", "save", "my-proj", "--project")

			result := env.RunInDir(projectDir, "profile", "save", "my-proj", "--project", "--apply", "-y")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("Saved and applied"))
		})

		It("tracks at local scope when --local --apply is used", func() {
			localDir := env.ProjectDir("save-apply-local")
			env.CreateLocalScopeSettings(localDir, map[string]bool{
				"local-plugin@marketplace": true,
			})

			result := env.RunInDir(localDir, "profile", "save", "my-local", "--local", "--apply")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("Saved and applied"))

			// Verify the profile is tracked in the projects registry at local scope
			registry := env.LoadProjectsRegistry()
			Expect(registry).NotTo(BeNil())
			projects := registry["projects"].(map[string]any)
			normalizedDir := localDir
			if resolved, err := filepath.EvalSymlinks(localDir); err == nil {
				normalizedDir = resolved
			}
			entry := projects[normalizedDir].(map[string]any)
			Expect(entry["profile"]).To(Equal("my-local"))
		})

		It("works for user scope without explicit scope flag", func() {
			result := env.Run("profile", "save", "my-user", "--apply")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("Saved and applied"))
			Expect(env.GetActiveProfile()).To(Equal("my-user"))
		})

		It("does not say applied when --apply is not used", func() {
			result := env.Run("profile", "save", "plain-save")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("Saved profile"))
			Expect(result.Stdout).NotTo(ContainSubstring("Saved and applied"))
		})

		It("does not set user-scope active profile when saving at project scope", func() {
			projectDir := env.ProjectDir("save-apply-no-user-project")
			env.CreateProjectScopeSettings(projectDir, map[string]bool{
				"plugin-a@marketplace": true,
			})

			result := env.RunInDir(projectDir, "profile", "save", "proj-only", "--project", "--apply")

			Expect(result.ExitCode).To(Equal(0))
			Expect(env.GetActiveProfile()).To(BeEmpty())
		})

		It("does not set user-scope active profile when saving at local scope", func() {
			localDir := env.ProjectDir("save-apply-no-user-local")
			env.CreateLocalScopeSettings(localDir, map[string]bool{
				"local-plugin@marketplace": true,
			})

			result := env.RunInDir(localDir, "profile", "save", "local-only", "--local", "--apply")

			Expect(result.ExitCode).To(Equal(0))
			Expect(env.GetActiveProfile()).To(BeEmpty())
		})
	})
})
