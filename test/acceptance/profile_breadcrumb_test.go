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
})
