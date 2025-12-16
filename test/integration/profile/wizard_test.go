// ABOUTME: Integration tests for profile wizard functionality
// ABOUTME: Tests name prompting, validation, and wizard helpers
package profile_test

import (
	"github.com/claudeup/claudeup/internal/profile"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
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

			Expect(marketplaces).To(ContainElement(MatchFields(IgnoreExtras, Fields{
				"Source": Equal("github"),
				"Repo":   Equal("wshobson/agents"),
			})))
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
})
