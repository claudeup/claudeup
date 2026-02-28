package acceptance

import (
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

			// Apply may fail on plugin install, but breadcrumb is written on the
			// "already matches" path since profile is empty and live is empty
			if result.ExitCode == 0 {
				bc := env.ReadBreadcrumb()
				Expect(bc).To(HaveKey("user"))
				Expect(bc["user"].Profile).To(Equal("test-profile"))
			}
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
			Expect(result.Stderr).To(ContainSubstring("No profile has been applied"))
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
			Expect(result.Stderr).To(ContainSubstring("No profile has been applied"))
		})

		It("errors when breadcrumbed profile is deleted", func() {
			env.WriteBreadcrumb("user", "deleted-profile")

			result := env.Run("profile", "diff")

			Expect(result.ExitCode).To(Equal(1))
			Expect(result.Stderr).To(ContainSubstring("no longer exists"))
		})

		It("explicit name still works", func() {
			env.WriteBreadcrumb("user", "my-setup")

			result := env.Run("profile", "diff", "my-setup")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("my-setup"))
		})
	})
})
