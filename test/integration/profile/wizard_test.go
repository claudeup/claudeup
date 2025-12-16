// ABOUTME: Integration tests for profile wizard functionality
// ABOUTME: Tests name prompting, validation, and wizard helpers
package profile_test

import (
	"github.com/claudeup/claudeup/internal/profile"
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
})
