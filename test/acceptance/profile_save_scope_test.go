// ABOUTME: Acceptance tests for profile save --scope functionality
// ABOUTME: Tests project-local profile saving and context-aware defaults
package acceptance

import (
	"os"
	"path/filepath"

	"github.com/claudeup/claudeup/v2/test/helpers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("profile save --scope", func() {
	var env *helpers.TestEnv

	BeforeEach(func() {
		env = helpers.NewTestEnv(binaryPath)
		env.CreateClaudeSettings()
	})

	Describe("explicit --scope project", func() {
		var projectDir string

		BeforeEach(func() {
			projectDir = env.ProjectDir("test-project")
		})

		Context("when project has different settings than user scope", func() {
			BeforeEach(func() {
				// Set up user-scope settings with some plugins
				env.CreateSettings(map[string]bool{
					"user-plugin-a@marketplace": true,
					"user-plugin-b@marketplace": true,
				})

				// Set up project-scope settings with DIFFERENT plugins
				claudeDir := filepath.Join(projectDir, ".claude")
				Expect(os.MkdirAll(claudeDir, 0755)).To(Succeed())
				projectSettings := `{"enabledPlugins":{"project-plugin-x@marketplace":true,"project-plugin-y@marketplace":true}}`
				Expect(os.WriteFile(filepath.Join(claudeDir, "settings.json"), []byte(projectSettings), 0644)).To(Succeed())
			})

			It("captures project-scope settings, not user-scope settings", func() {
				result := env.RunInDir(projectDir, "profile", "save", "team-profile", "--scope", "project", "-y")

				Expect(result.ExitCode).To(Equal(0))

				// Load the saved profile
				profilePath := filepath.Join(projectDir, ".claudeup", "profiles", "team-profile.json")
				data, err := os.ReadFile(profilePath)
				Expect(err).NotTo(HaveOccurred())

				profileContent := string(data)
				// Should contain project-scope plugins
				Expect(profileContent).To(ContainSubstring("project-plugin-x@marketplace"))
				Expect(profileContent).To(ContainSubstring("project-plugin-y@marketplace"))
				// Should NOT contain user-scope plugins
				Expect(profileContent).NotTo(ContainSubstring("user-plugin-a@marketplace"))
				Expect(profileContent).NotTo(ContainSubstring("user-plugin-b@marketplace"))
			})
		})

		It("saves profile to .claudeup/profiles/ in project directory", func() {
			result := env.RunInDir(projectDir, "profile", "save", "team-profile", "--scope", "project", "-y")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("project scope"))

			// Verify file location in project directory
			profilePath := filepath.Join(projectDir, ".claudeup", "profiles", "team-profile.json")
			Expect(profilePath).To(BeAnExistingFile())
		})

		It("does not save to user profiles directory", func() {
			result := env.RunInDir(projectDir, "profile", "save", "team-profile", "--scope", "project", "-y")

			Expect(result.ExitCode).To(Equal(0))

			// Should NOT be in user profiles dir
			userProfilePath := filepath.Join(env.ProfilesDir, "team-profile.json")
			Expect(userProfilePath).NotTo(BeAnExistingFile())
		})

		It("does not set active profile in user config for project scope", func() {
			result := env.RunInDir(projectDir, "profile", "save", "team-profile", "--scope", "project", "-y")

			Expect(result.ExitCode).To(Equal(0))

			// Active profile should not be set for project-scope saves
			// (active profile is only tracked for user-scope profiles)
			Expect(env.GetActiveProfile()).To(BeEmpty())
		})

		It("shows git add hint for project scope", func() {
			result := env.RunInDir(projectDir, "profile", "save", "team-profile", "--scope", "project", "-y")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring(".claudeup/profiles/"))
		})
	})

	Describe("explicit --scope user", func() {
		var projectDir string

		BeforeEach(func() {
			projectDir = env.ProjectDir("test-project")
		})

		It("saves profile to ~/.claudeup/profiles/", func() {
			result := env.RunInDir(projectDir, "profile", "save", "my-profile", "--scope", "user", "-y")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("user scope"))

			// Verify file location in user profiles directory
			profilePath := filepath.Join(env.ProfilesDir, "my-profile.json")
			Expect(profilePath).To(BeAnExistingFile())
		})

		It("does not save to project directory", func() {
			result := env.RunInDir(projectDir, "profile", "save", "my-profile", "--scope", "user", "-y")

			Expect(result.ExitCode).To(Equal(0))

			// Should NOT be in project profiles dir
			projectProfilePath := filepath.Join(projectDir, ".claudeup", "profiles", "my-profile.json")
			Expect(projectProfilePath).NotTo(BeAnExistingFile())
		})

		It("sets active profile in user config for user scope", func() {
			result := env.RunInDir(projectDir, "profile", "save", "my-profile", "--scope", "user", "-y")

			Expect(result.ExitCode).To(Equal(0))
			Expect(env.GetActiveProfile()).To(Equal("my-profile"))
		})
	})

	Describe("context-aware default", func() {
		var projectDir string

		BeforeEach(func() {
			projectDir = env.ProjectDir("context-test")
		})

		Context("when .claudeup.json exists", func() {
			BeforeEach(func() {
				// Create .claudeup.json in project dir
				env.CreateClaudeupJSON(projectDir, map[string]interface{}{
					"version": "1",
					"profile": "existing",
				})
			})

			It("defaults to project scope", func() {
				result := env.RunInDir(projectDir, "profile", "save", "team-profile", "-y")

				Expect(result.ExitCode).To(Equal(0))
				Expect(result.Stdout).To(ContainSubstring("project scope"))

				// Profile should be in project directory
				profilePath := filepath.Join(projectDir, ".claudeup", "profiles", "team-profile.json")
				Expect(profilePath).To(BeAnExistingFile())
			})

			It("does not save to user profiles directory", func() {
				result := env.RunInDir(projectDir, "profile", "save", "team-profile", "-y")

				Expect(result.ExitCode).To(Equal(0))

				// Should NOT be in user profiles dir
				userProfilePath := filepath.Join(env.ProfilesDir, "team-profile.json")
				Expect(userProfilePath).NotTo(BeAnExistingFile())
			})
		})

		Context("when .claudeup.json does not exist", func() {
			It("defaults to user scope", func() {
				result := env.RunInDir(projectDir, "profile", "save", "my-profile", "-y")

				Expect(result.ExitCode).To(Equal(0))
				Expect(result.Stdout).To(ContainSubstring("user scope"))

				// Profile should be in user profiles directory
				profilePath := filepath.Join(env.ProfilesDir, "my-profile.json")
				Expect(profilePath).To(BeAnExistingFile())
			})

			It("does not save to project directory", func() {
				result := env.RunInDir(projectDir, "profile", "save", "my-profile", "-y")

				Expect(result.ExitCode).To(Equal(0))

				// Should NOT be in project profiles dir
				projectProfilePath := filepath.Join(projectDir, ".claudeup", "profiles", "my-profile.json")
				Expect(projectProfilePath).NotTo(BeAnExistingFile())
			})
		})
	})

	Describe("invalid scope", func() {
		It("returns error for unknown scope", func() {
			result := env.Run("profile", "save", "test", "--scope", "invalid")

			Expect(result.ExitCode).NotTo(Equal(0))
			Expect(result.Stderr).To(ContainSubstring("invalid scope"))
		})

		It("rejects 'local' scope (only user and project supported)", func() {
			result := env.Run("profile", "save", "test", "--scope", "local")

			Expect(result.ExitCode).NotTo(Equal(0))
			Expect(result.Stderr).To(ContainSubstring("invalid scope"))
		})
	})

	Describe("overwrite behavior with scope", func() {
		var projectDir string

		BeforeEach(func() {
			projectDir = env.ProjectDir("overwrite-test")
		})

		It("prompts when overwriting project-scope profile", func() {
			// Create initial project profile
			projectProfilesDir := filepath.Join(projectDir, ".claudeup", "profiles")
			Expect(os.MkdirAll(projectProfilesDir, 0755)).To(Succeed())
			Expect(os.WriteFile(
				filepath.Join(projectProfilesDir, "existing.json"),
				[]byte(`{"name":"existing"}`),
				0644,
			)).To(Succeed())

			// Try to save with same name - should prompt
			result := env.RunInDirWithInput(projectDir, "n\n", "profile", "save", "existing", "--scope", "project")

			Expect(result.Stdout).To(ContainSubstring("already exists"))
			Expect(result.Stdout).To(ContainSubstring("Cancelled"))
		})

		It("overwrites project-scope profile with -y flag", func() {
			// Create initial project profile
			projectProfilesDir := filepath.Join(projectDir, ".claudeup", "profiles")
			Expect(os.MkdirAll(projectProfilesDir, 0755)).To(Succeed())
			Expect(os.WriteFile(
				filepath.Join(projectProfilesDir, "existing.json"),
				[]byte(`{"name":"existing"}`),
				0644,
			)).To(Succeed())

			result := env.RunInDir(projectDir, "profile", "save", "existing", "--scope", "project", "-y")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("Saved profile"))
		})
	})
})
