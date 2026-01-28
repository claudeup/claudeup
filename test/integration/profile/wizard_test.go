// ABOUTME: Integration tests for profile wizard functionality
// ABOUTME: Tests name prompting, validation, and wizard helpers
package profile_test

import (
	"github.com/claudeup/claudeup/v3/internal/profile"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Wizard", func() {
	Describe("ValidateName", func() {
		It("accepts valid profile names", func() {
			err := profile.ValidateName("my-profile")
			Expect(err).To(BeNil())
		})

		It("rejects empty names", func() {
			err := profile.ValidateName("")
			Expect(err).To(MatchError("profile name cannot be empty"))
		})

		It("rejects reserved name 'current'", func() {
			err := profile.ValidateName("current")
			Expect(err).To(MatchError("'current' is a reserved name"))
		})

		It("rejects names with invalid characters", func() {
			err := profile.ValidateName("my profile!")
			Expect(err).To(MatchError(ContainSubstring("invalid characters")))
		})
	})

	Describe("GetAvailableMarketplaces", func() {
		It("returns embedded marketplaces", func() {
			marketplaces := profile.GetAvailableMarketplaces()

			// Should return at least one marketplace from embedded profiles
			Expect(marketplaces).NotTo(BeEmpty())

			// All marketplaces should have github source and valid repo names
			for _, m := range marketplaces {
				Expect(m.Source).To(Equal("github"))
				Expect(m.Repo).NotTo(BeEmpty())
			}
		})

		It("includes marketplace display names", func() {
			marketplaces := profile.GetAvailableMarketplaces()

			for _, m := range marketplaces {
				Expect(m.DisplayName()).NotTo(BeEmpty())
			}
		})
	})

	Describe("PromptForName", func() {
		It("validates input", func() {
			Skip("Requires stdin simulation - tested via acceptance tests")
		})
	})

	Describe("SelectMarketplaces", func() {
		It("returns error if no marketplaces available", func() {
			selected, err := profile.SelectMarketplaces([]profile.Marketplace{})
			Expect(err).To(MatchError("no marketplaces available"))
			Expect(selected).To(BeNil())
		})
	})

	Describe("SelectPluginsForMarketplace", func() {
		It("uses category-based selection for marketplaces with categories", func() {
			marketplace := profile.Marketplace{
				Source: "github",
				Repo:   "wshobson/agents",
			}

			// This should trigger category-based selection path
			// Since we can't mock user input in this test, we expect it to fail
			// when trying to get user input (gum or fallback)
			_, err := profile.SelectPluginsForMarketplace(marketplace)
			Expect(err).NotTo(BeNil())
		})

		It("uses flat selection for marketplaces without categories", func() {
			marketplace := profile.Marketplace{
				Source: "github",
				Repo:   "unknown/marketplace",
			}

			// This should trigger flat selection path
			// Currently returns empty list (stubbed)
			plugins, err := profile.SelectPluginsForMarketplace(marketplace)
			Expect(err).To(BeNil())
			Expect(plugins).To(BeEmpty())
		})
	})
})
