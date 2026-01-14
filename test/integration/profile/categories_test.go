// ABOUTME: Tests for marketplace category detection and plugin mapping
// ABOUTME: Hardcoded category support for known marketplaces
package profile_test

import (
	"github.com/claudeup/claudeup/v2/internal/profile"
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
			It("returns all 8 categories", func() {
				categories := profile.GetCategories("wshobson/agents")
				Expect(categories).To(HaveLen(8))
			})

			It("returns Core Development category", func() {
				categories := profile.GetCategories("wshobson/agents")

				var found bool
				for _, cat := range categories {
					if cat.Name == "Core Development" {
						found = true
						Expect(cat.Description).To(Equal("workflows, debugging, docs, refactoring"))
						Expect(cat.Plugins).To(ContainElement("code-documentation"))
						Expect(cat.Plugins).To(ContainElement("debugging-toolkit"))
						break
					}
				}
				Expect(found).To(BeTrue())
			})

			It("returns Quality & Testing category", func() {
				categories := profile.GetCategories("wshobson/agents")

				var found bool
				for _, cat := range categories {
					if cat.Name == "Quality & Testing" {
						found = true
						Expect(cat.Description).To(Equal("code review, testing, cleanup"))
						Expect(cat.Plugins).To(ContainElement("unit-testing"))
						Expect(cat.Plugins).To(ContainElement("tdd-workflows"))
						break
					}
				}
				Expect(found).To(BeTrue())
			})

			It("returns AI & Machine Learning category", func() {
				categories := profile.GetCategories("wshobson/agents")

				var found bool
				for _, cat := range categories {
					if cat.Name == "AI & Machine Learning" {
						found = true
						Expect(cat.Description).To(Equal("LLM dev, agents, MLOps"))
						Expect(cat.Plugins).To(ContainElement("llm-application-dev"))
						break
					}
				}
				Expect(found).To(BeTrue())
			})

			It("returns Infrastructure & DevOps category", func() {
				categories := profile.GetCategories("wshobson/agents")

				var found bool
				for _, cat := range categories {
					if cat.Name == "Infrastructure & DevOps" {
						found = true
						Expect(cat.Plugins).To(ContainElement("kubernetes-operations"))
						break
					}
				}
				Expect(found).To(BeTrue())
			})

			It("returns Security & Compliance category", func() {
				categories := profile.GetCategories("wshobson/agents")

				var found bool
				for _, cat := range categories {
					if cat.Name == "Security & Compliance" {
						found = true
						Expect(cat.Plugins).To(ContainElement("security-scanning"))
						break
					}
				}
				Expect(found).To(BeTrue())
			})

			It("returns Data & Databases category", func() {
				categories := profile.GetCategories("wshobson/agents")

				var found bool
				for _, cat := range categories {
					if cat.Name == "Data & Databases" {
						found = true
						Expect(cat.Plugins).To(ContainElement("database-design"))
						break
					}
				}
				Expect(found).To(BeTrue())
			})

			It("returns Languages category", func() {
				categories := profile.GetCategories("wshobson/agents")

				var found bool
				for _, cat := range categories {
					if cat.Name == "Languages" {
						found = true
						Expect(cat.Plugins).To(ContainElement("python-development"))
						Expect(cat.Plugins).To(ContainElement("javascript-typescript"))
						break
					}
				}
				Expect(found).To(BeTrue())
			})

			It("returns Business & Specialty category", func() {
				categories := profile.GetCategories("wshobson/agents")

				var found bool
				for _, cat := range categories {
					if cat.Name == "Business & Specialty" {
						found = true
						Expect(cat.Plugins).To(ContainElement("blockchain-web3"))
						break
					}
				}
				Expect(found).To(BeTrue())
			})

			It("parses plugin names correctly", func() {
				categories := profile.GetCategories("wshobson/agents")

				coreDevCat := categories[0]
				Expect(coreDevCat.Plugins).To(HaveLen(14))
				Expect(coreDevCat.Plugins[0]).To(Equal("code-documentation"))
				Expect(coreDevCat.Plugins[13]).To(Equal("developer-essentials"))
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
