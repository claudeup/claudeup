// ABOUTME: Integration tests for profile wizard functionality
// ABOUTME: Tests name prompting, validation, and wizard helpers
package profile_test

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/claudeup/claudeup/v5/internal/profile"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// noGumLookPath simulates gum not being installed, forcing fallback paths.
func noGumLookPath(name string) (string, error) {
	return "", fmt.Errorf("executable file not found in $PATH")
}

// testWizardIO creates a WizardIO with piped input and no gum.
func testWizardIO(input string) (profile.WizardIO, *bytes.Buffer) {
	out := &bytes.Buffer{}
	return profile.WizardIO{
		In:       strings.NewReader(input),
		Out:      out,
		Err:      &bytes.Buffer{},
		LookPath: noGumLookPath,
	}, out
}

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

			// All marketplaces should have valid source type and identifier
			for _, m := range marketplaces {
				// Accept github, git, or directory as valid source types
				Expect(m.Source).To(BeElementOf("github", "git", "directory"))
				// Must have either Repo or URL set
				Expect(m.Repo != "" || m.URL != "").To(BeTrue(), "marketplace must have Repo or URL")
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
		It("reads and validates a valid name from input", func() {
			wio, _ := testWizardIO("my-profile\n")

			name, err := profile.PromptForName(wio)
			Expect(err).NotTo(HaveOccurred())
			Expect(name).To(Equal("my-profile"))
		})

		It("returns error on EOF", func() {
			wio, _ := testWizardIO("")

			_, err := profile.PromptForName(wio)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to read input"))
		})

		It("re-prompts on invalid then accepts valid name", func() {
			// First line is invalid (has spaces), second is valid
			wio, out := testWizardIO("bad name!\nmy-profile\n")

			name, err := profile.PromptForName(wio)
			Expect(err).NotTo(HaveOccurred())
			Expect(name).To(Equal("my-profile"))
			// Should have printed an error for the invalid name
			Expect(out.String()).To(ContainSubstring("Error:"))
		})
	})

	Describe("SelectMarketplaces", func() {
		It("returns error if no marketplaces available", func() {
			wio, _ := testWizardIO("")
			selected, err := profile.SelectMarketplaces(wio, []profile.Marketplace{})
			Expect(err).To(MatchError("no marketplaces available"))
			Expect(selected).To(BeNil())
		})

		It("selects a marketplace by number via fallback", func() {
			marketplaces := []profile.Marketplace{
				{Source: "github", Repo: "owner/first"},
				{Source: "github", Repo: "owner/second"},
			}
			wio, out := testWizardIO("2\n")

			selected, err := profile.SelectMarketplaces(wio, marketplaces)
			Expect(err).NotTo(HaveOccurred())
			Expect(selected).To(HaveLen(1))
			Expect(selected[0].Repo).To(Equal("owner/second"))
			// Should have shown the numbered menu
			Expect(out.String()).To(ContainSubstring("1) owner/first"))
			Expect(out.String()).To(ContainSubstring("2) owner/second"))
		})

		It("selects multiple marketplaces by comma-separated numbers", func() {
			marketplaces := []profile.Marketplace{
				{Source: "github", Repo: "owner/first"},
				{Source: "github", Repo: "owner/second"},
				{Source: "github", Repo: "owner/third"},
			}
			wio, _ := testWizardIO("1,3\n")

			selected, err := profile.SelectMarketplaces(wio, marketplaces)
			Expect(err).NotTo(HaveOccurred())
			Expect(selected).To(HaveLen(2))
			Expect(selected[0].Repo).To(Equal("owner/first"))
			Expect(selected[1].Repo).To(Equal("owner/third"))
		})

		It("returns error on empty input", func() {
			marketplaces := []profile.Marketplace{
				{Source: "github", Repo: "owner/first"},
			}
			wio, _ := testWizardIO("\n")

			_, err := profile.SelectMarketplaces(wio, marketplaces)
			Expect(err).To(MatchError("no marketplaces selected"))
		})

		It("returns error on EOF", func() {
			marketplaces := []profile.Marketplace{
				{Source: "github", Repo: "owner/first"},
			}
			wio, _ := testWizardIO("")

			_, err := profile.SelectMarketplaces(wio, marketplaces)
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("SelectPluginsForMarketplace", func() {
		It("returns error on EOF for category-based marketplace", func() {
			// wshobson/agents has categories — the fallback category selection
			// will hit EOF and return an empty category list (graceful skip).
			marketplace := profile.Marketplace{
				Source: "github",
				Repo:   "wshobson/agents",
			}
			wio, _ := testWizardIO("")

			// Empty input in fallbackCategorySelection returns empty categories (q/skip behavior)
			// which means no plugins are collected — empty result, no error
			_, err := profile.SelectPluginsForMarketplace(wio, marketplace)
			// EOF on the category fallback returns "failed to read input" error
			Expect(err).To(HaveOccurred())
		})

		It("selects categories then returns plugins on EOF refinement", func() {
			marketplace := profile.Marketplace{
				Source: "github",
				Repo:   "wshobson/agents",
			}
			// Select category 1 (Core Development), then empty input for plugin refinement
			// accepts pre-selected (none installed → empty)
			wio, _ := testWizardIO("1\n\n")

			plugins, err := profile.SelectPluginsForMarketplace(wio, marketplace)
			Expect(err).NotTo(HaveOccurred())
			// Should have plugins from Core Development category
			Expect(plugins).NotTo(BeEmpty())
		})

		It("uses flat selection for marketplaces without categories", func() {
			marketplace := profile.Marketplace{
				Source: "github",
				Repo:   "unknown/marketplace",
			}

			// Flat selection path — listPluginsFromMarketplace fails gracefully
			// for unknown marketplace, returns empty list
			wio, _ := testWizardIO("")
			plugins, err := profile.SelectPluginsForMarketplace(wio, marketplace)
			Expect(err).To(BeNil())
			Expect(plugins).To(BeEmpty())
		})
	})

	Describe("PromptForDescription", func() {
		It("accepts auto-generated description when user declines edit", func() {
			wio, _ := testWizardIO("n\n")

			desc, err := profile.PromptForDescription(wio, "Auto description")
			Expect(err).NotTo(HaveOccurred())
			Expect(desc).To(Equal("Auto description"))
		})

		It("returns auto-generated on EOF", func() {
			wio, _ := testWizardIO("")

			desc, err := profile.PromptForDescription(wio, "Auto description")
			Expect(err).NotTo(HaveOccurred())
			Expect(desc).To(Equal("Auto description"))
		})

		It("allows user to enter custom description", func() {
			wio, _ := testWizardIO("y\nMy custom description\n")

			desc, err := profile.PromptForDescription(wio, "Auto description")
			Expect(err).NotTo(HaveOccurred())
			Expect(desc).To(Equal("My custom description"))
		})

		It("uses auto-generated if user says yes but enters empty description", func() {
			wio, _ := testWizardIO("y\n\n")

			desc, err := profile.PromptForDescription(wio, "Auto description")
			Expect(err).NotTo(HaveOccurred())
			Expect(desc).To(Equal("Auto description"))
		})
	})
})
