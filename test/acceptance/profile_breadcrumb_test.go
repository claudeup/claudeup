package acceptance

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/claudeup/claudeup/v5/internal/profile"
	"github.com/claudeup/claudeup/v5/test/helpers"
)

var _ = Describe("Profile breadcrumb", func() {
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

	Describe("apply writes breadcrumb", func() {
		BeforeEach(func() {
			// Create a minimal profile (no plugins, so apply succeeds without claude CLI)
			env.CreateProfile(&profile.Profile{
				Name:        "test-profile",
				Description: "test",
			})
		})

		It("writes breadcrumb at user scope by default", func() {
			result := env.RunWithInput("y\n", "profile", "apply", "test-profile", "-y")

			Expect(result.ExitCode).To(Equal(0))
			bc := env.ReadBreadcrumb()
			Expect(bc).To(HaveKey("user"))
			Expect(bc["user"].Profile).To(Equal("test-profile"))
		})

		It("does not write breadcrumb in dry-run mode", func() {
			result := env.Run("profile", "apply", "test-profile", "--dry-run")

			Expect(result.ExitCode).To(Equal(0))
			Expect(env.BreadcrumbExists()).To(BeFalse())
		})

		It("warns but succeeds when breadcrumb file is corrupt", func() {
			// Write garbage to the breadcrumb file
			bcPath := filepath.Join(env.ClaudeupDir, "last-applied.json")
			Expect(os.WriteFile(bcPath, []byte("{invalid json"), 0600)).To(Succeed())

			result := env.RunWithInput("y\n", "profile", "apply", "test-profile", "-y")

			Expect(result.ExitCode).To(Equal(0))
			combined := result.Stdout + result.Stderr
			Expect(combined).To(ContainSubstring("breadcrumb"))
		})
	})

	Describe("save with no args", func() {
		It("saves to breadcrumbed profile name", func() {
			// Create original profile
			env.CreateProfile(&profile.Profile{
				Name:        "my-setup",
				Description: "original",
			})
			env.WriteBreadcrumb("user", "my-setup")

			// Add a plugin to live state so there's something to save
			env.CreateInstalledPlugins(map[string]interface{}{
				"new-plugin@marketplace": map[string]interface{}{
					"scope": "user",
				},
			})

			result := env.RunWithInput("y\n", "profile", "save")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("my-setup"))
		})

		It("errors when no breadcrumb exists", func() {
			result := env.Run("profile", "save")

			Expect(result.ExitCode).To(Equal(1))
			Expect(result.Stderr).To(ContainSubstring("no profile has been applied"))
		})

		It("explicit name still works", func() {
			result := env.RunWithInput("y\n", "profile", "save", "explicit-name")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("explicit-name"))
		})
	})

	Describe("delete cleans breadcrumb", func() {
		It("removes breadcrumb entry for deleted profile", func() {
			env.CreateProfile(&profile.Profile{
				Name: "to-delete",
			})
			env.WriteBreadcrumb("user", "to-delete")
			env.WriteBreadcrumb("project", "keep-this")

			result := env.RunWithInput("y\n", "profile", "delete", "to-delete")

			Expect(result.ExitCode).To(Equal(0))

			bc := env.ReadBreadcrumb()
			Expect(bc).NotTo(HaveKey("user"))
			Expect(bc).To(HaveKey("project"))
			Expect(bc["project"].Profile).To(Equal("keep-this"))
		})
	})

	Describe("rename updates breadcrumb", func() {
		It("updates breadcrumb entry to new name", func() {
			env.CreateProfile(&profile.Profile{
				Name: "old-name",
			})
			env.WriteBreadcrumb("user", "old-name")

			result := env.Run("profile", "rename", "old-name", "new-name")

			Expect(result.ExitCode).To(Equal(0))

			bc := env.ReadBreadcrumb()
			Expect(bc).To(HaveKey("user"))
			Expect(bc["user"].Profile).To(Equal("new-name"))
		})
	})

	Describe("diff with no args", func() {
		BeforeEach(func() {
			// Create a profile with a plugin at user scope
			env.CreateProfile(&profile.Profile{
				Name: "my-setup",
				PerScope: &profile.PerScopeSettings{
					User: &profile.ScopeSettings{
						Plugins: []string{"extra-plugin@marketplace"},
					},
				},
			})
		})

		It("uses highest-precedence breadcrumb", func() {
			env.WriteBreadcrumb("user", "my-setup")

			result := env.Run("profile", "diff")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("my-setup"))
		})

		It("prefers project breadcrumb over user", func() {
			env.WriteBreadcrumb("user", "some-other")
			env.WriteBreadcrumb("project", "my-setup")

			// Create the other profile too
			env.CreateProfile(&profile.Profile{Name: "some-other"})

			result := env.Run("profile", "diff")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("my-setup"))
		})

		It("errors when no breadcrumb exists", func() {
			result := env.Run("profile", "diff")

			Expect(result.ExitCode).To(Equal(1))
			Expect(result.Stderr).To(ContainSubstring("no profile has been applied"))
		})

		It("errors when breadcrumbed profile is deleted", func() {
			env.WriteBreadcrumb("user", "deleted-profile")

			result := env.Run("profile", "diff")

			Expect(result.ExitCode).To(Equal(1))
			Expect(result.Stderr).To(ContainSubstring("not found"))
		})

		It("explicit name still works", func() {
			env.WriteBreadcrumb("user", "my-setup")

			result := env.Run("profile", "diff", "my-setup")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("my-setup"))
		})

		It("uses --user flag to select user scope breadcrumb", func() {
			env.WriteBreadcrumb("user", "my-setup")
			env.WriteBreadcrumb("project", "other-profile")
			env.CreateProfile(&profile.Profile{Name: "other-profile"})

			result := env.Run("profile", "diff", "--user")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("my-setup"))
		})

		It("errors when scope flag has no breadcrumb", func() {
			env.WriteBreadcrumb("user", "my-setup")

			result := env.Run("profile", "diff", "--project")

			Expect(result.ExitCode).To(Equal(1))
			Expect(result.Stderr).To(ContainSubstring("no profile has been applied at project scope"))
		})

		It("errors when scope flag used with explicit name", func() {
			env.WriteBreadcrumb("user", "my-setup")

			result := env.Run("profile", "diff", "my-setup", "--user")

			Expect(result.ExitCode).To(Equal(1))
			Expect(result.Stderr).To(ContainSubstring("cannot use --scope"))
		})

		It("errors on invalid scope value", func() {
			result := env.Run("profile", "diff", "--scope", "banana")

			Expect(result.ExitCode).To(Equal(1))
			Expect(result.Stderr).To(ContainSubstring("invalid scope"))
		})
	})

	Describe("status shows last-applied info", func() {
		It("shows last-applied line when breadcrumb exists and profile matches", func() {
			env.CreateProfile(&profile.Profile{
				Name:        "my-setup",
				Description: "test profile",
			})
			env.WriteBreadcrumb("user", "my-setup")

			result := env.Run("profile", "status")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("Last applied"))
			Expect(result.Stdout).To(ContainSubstring("my-setup"))
			Expect(result.Stdout).To(ContainSubstring("user scope"))
		})

		It("shows (modified) when live state differs from saved profile", func() {
			// Create profile with a plugin
			env.CreateProfile(&profile.Profile{
				Name: "my-setup",
				PerScope: &profile.PerScopeSettings{
					User: &profile.ScopeSettings{
						Plugins: []string{"saved-plugin@marketplace"},
					},
				},
			})
			env.WriteBreadcrumb("user", "my-setup")

			// Live state has a different plugin
			env.CreateInstalledPlugins(map[string]interface{}{
				"different-plugin@marketplace": map[string]interface{}{
					"scope": "user",
				},
			})

			result := env.Run("profile", "status")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("Last applied"))
			Expect(result.Stdout).To(ContainSubstring("my-setup"))
			Expect(result.Stdout).To(ContainSubstring("modified"))
		})

		It("does not show (modified) when live matches saved profile", func() {
			env.CreateProfile(&profile.Profile{
				Name:        "my-setup",
				Description: "test",
			})
			env.WriteBreadcrumb("user", "my-setup")

			result := env.Run("profile", "status")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("Last applied"))
			Expect(result.Stdout).To(ContainSubstring("my-setup"))
			Expect(result.Stdout).NotTo(ContainSubstring("modified"))
		})

		It("skips last-applied section when no breadcrumb exists", func() {
			result := env.Run("profile", "status")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).NotTo(ContainSubstring("Last applied"))
		})

		It("skips last-applied section when breadcrumbed profile is missing", func() {
			env.WriteBreadcrumb("user", "deleted-profile")

			result := env.Run("profile", "status")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).NotTo(ContainSubstring("Last applied"))
		})
	})

	Describe("list shows applied marker", func() {
		It("shows (applied) on the matching profile", func() {
			env.CreateProfile(&profile.Profile{
				Name:        "my-setup",
				Description: "test profile",
			})
			env.WriteBreadcrumb("user", "my-setup")

			result := env.Run("profile", "list")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("my-setup"))
			Expect(result.Stdout).To(ContainSubstring("(applied)"))
		})

		It("shows (applied, modified) when profile has drifted", func() {
			env.CreateProfile(&profile.Profile{
				Name: "my-setup",
				PerScope: &profile.PerScopeSettings{
					User: &profile.ScopeSettings{
						Plugins: []string{"saved-plugin@marketplace"},
					},
				},
			})
			env.WriteBreadcrumb("user", "my-setup")

			// Live state has a different plugin
			env.CreateInstalledPlugins(map[string]interface{}{
				"different-plugin@marketplace": map[string]interface{}{
					"scope": "user",
				},
			})

			result := env.Run("profile", "list")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("my-setup"))
			Expect(result.Stdout).To(ContainSubstring("(applied, modified)"))
		})

		It("shows no marker when no breadcrumb exists", func() {
			env.CreateProfile(&profile.Profile{
				Name:        "my-setup",
				Description: "test profile",
			})

			result := env.Run("profile", "list")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("my-setup"))
			Expect(result.Stdout).NotTo(ContainSubstring("(applied"))
		})

		It("shows marker on built-in profile when breadcrumbed", func() {
			env.WriteBreadcrumb("user", "frontend")

			result := env.Run("profile", "list")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("frontend"))
			// Built-in frontend has plugins; empty test env means it's always modified
			Expect(result.Stdout).To(ContainSubstring("(applied, modified)"))
		})
	})
})
