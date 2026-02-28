// ABOUTME: Acceptance tests for profile diff --original comparing customized to built-in
// ABOUTME: Ensures profile diff --original shows differences between customized and original built-in profiles
package acceptance

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/claudeup/claudeup/v5/internal/profile"
	"github.com/claudeup/claudeup/v5/test/helpers"
)

var _ = Describe("Profile diff --original", func() {
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

	Describe("no arguments", func() {
		It("returns error when no profile name given", func() {
			result := env.Run("profile", "diff", "--original")

			Expect(result.ExitCode).To(Equal(1))
			Expect(result.Stderr).To(ContainSubstring("--original requires exactly 1 profile name"))
		})
	})

	Describe("non-existent profile", func() {
		It("returns error for unknown profile", func() {
			result := env.Run("profile", "diff", "--original", "nonexistent-profile")

			Expect(result.ExitCode).To(Equal(1))
			Expect(result.Stderr).To(ContainSubstring("not a built-in profile"))
		})
	})

	Describe("non-builtin profile", func() {
		BeforeEach(func() {
			// Create a custom profile that isn't a built-in
			env.CreateProfile(&profile.Profile{
				Name:        "my-custom",
				Description: "A custom profile",
				Plugins:     []string{"some@plugin"},
			})
		})

		It("returns error for non-builtin profile", func() {
			result := env.Run("profile", "diff", "--original", "my-custom")

			Expect(result.ExitCode).To(Equal(1))
			Expect(result.Stderr).To(ContainSubstring("not a built-in profile"))
		})
	})

	Describe("unmodified built-in profile", func() {
		It("shows no differences for unmodified built-in", func() {
			result := env.Run("profile", "diff", "--original", "default")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("No differences"))
		})
	})

	Describe("modified built-in profile", func() {
		BeforeEach(func() {
			// Get the built-in default profile and modify it
			defaultProfile, err := profile.GetEmbeddedProfile("default")
			Expect(err).NotTo(HaveOccurred())

			// Create a customized version with an extra plugin
			customized := &profile.Profile{
				Name:         "default",
				Description:  defaultProfile.Description,
				Marketplaces: defaultProfile.Marketplaces,
				Plugins:      []string{"extra-plugin@marketplace"}, // Added plugin
			}
			env.CreateProfile(customized)
		})

		It("shows added plugin", func() {
			result := env.Run("profile", "diff", "--original", "default")

			Expect(result.ExitCode).To(Equal(0))
			// Should show added plugin with + symbol
			Expect(result.Stdout).To(ContainSubstring("+"))
			Expect(result.Stdout).To(ContainSubstring("extra-plugin@marketplace"))
		})
	})

	Describe("modified description", func() {
		BeforeEach(func() {
			// Get the embedded profile and only change the description
			defaultProfile, err := profile.GetEmbeddedProfile("default")
			Expect(err).NotTo(HaveOccurred())

			customized := &profile.Profile{
				Name:         "default",
				Description:  "My custom description",
				Marketplaces: defaultProfile.Marketplaces,
				Plugins:      defaultProfile.Plugins,
				MCPServers:   defaultProfile.MCPServers,
			}
			env.CreateProfile(customized)
		})

		It("shows modified description", func() {
			result := env.Run("profile", "diff", "--original", "default")

			Expect(result.ExitCode).To(Equal(0))
			// Should show modified description with ~ symbol
			Expect(result.Stdout).To(ContainSubstring("~"))
			Expect(result.Stdout).To(ContainSubstring("description"))
		})
	})

	Describe("removed marketplace", func() {
		BeforeEach(func() {
			// Create a customized version with no marketplaces
			customized := &profile.Profile{
				Name:         "default",
				Description:  "Base Claude Code setup with essential marketplaces",
				Marketplaces: []profile.Marketplace{}, // Empty - removed the marketplace
			}
			env.CreateProfile(customized)
		})

		It("shows removed marketplace", func() {
			result := env.Run("profile", "diff", "--original", "default")

			Expect(result.ExitCode).To(Equal(0))
			// Should show removed marketplace with - symbol
			Expect(result.Stdout).To(ContainSubstring("-"))
			Expect(result.Stdout).To(ContainSubstring("marketplace"))
		})
	})
})
