// ABOUTME: Tests for marketplace category detection and plugin mapping
// ABOUTME: Hardcoded category support for known marketplaces
package profile_test

import (
	"github.com/claudeup/claudeup/internal/profile"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Categories", func() {
	Describe("HasCategories", func() {
		It("returns true for wshobson/agents marketplace", func() {
			hasCategories := profile.HasCategories("wshobson/agents")
			Expect(hasCategories).To(BeTrue())
		})

		It("returns false for unknown marketplaces", func() {
			hasCategories := profile.HasCategories("unknown/marketplace")
			Expect(hasCategories).To(BeFalse())
		})
	})

	Describe("GetCategories", func() {
		Context("for wshobson/agents", func() {
			It("returns backend development category", func() {
				categories := profile.GetCategories("wshobson/agents")

				var found bool
				for _, cat := range categories {
					if cat.Name == "Backend Development" {
						found = true
						break
					}
				}
				Expect(found).To(BeTrue())
			})

			It("returns frontend development category", func() {
				categories := profile.GetCategories("wshobson/agents")

				var found bool
				for _, cat := range categories {
					if cat.Name == "Frontend Development" {
						found = true
						break
					}
				}
				Expect(found).To(BeTrue())
			})
		})

		Context("for unknown marketplace", func() {
			It("returns empty list", func() {
				categories := profile.GetCategories("unknown/marketplace")
				Expect(categories).To(BeEmpty())
			})
		})
	})
})
