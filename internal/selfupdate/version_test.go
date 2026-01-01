// ABOUTME: Unit tests for semantic version comparison
// ABOUTME: Tests IsNewer function for update checks
package selfupdate

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestSelfupdate(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Selfupdate Suite")
}

var _ = Describe("IsNewer", func() {
	It("returns true when remote is newer major version", func() {
		Expect(IsNewer("v1.0.0", "v2.0.0")).To(BeTrue())
	})

	It("returns true when remote is newer minor version", func() {
		Expect(IsNewer("v1.0.0", "v1.1.0")).To(BeTrue())
	})

	It("returns true when remote is newer patch version", func() {
		Expect(IsNewer("v1.0.0", "v1.0.1")).To(BeTrue())
	})

	It("returns false when versions are equal", func() {
		Expect(IsNewer("v1.0.0", "v1.0.0")).To(BeFalse())
	})

	It("returns false when remote is older", func() {
		Expect(IsNewer("v1.1.0", "v1.0.0")).To(BeFalse())
	})

	It("handles versions without v prefix", func() {
		Expect(IsNewer("1.0.0", "1.1.0")).To(BeTrue())
	})

	It("handles dev version as always outdated", func() {
		Expect(IsNewer("dev", "v1.0.0")).To(BeTrue())
	})

	It("handles (devel) version as always outdated", func() {
		Expect(IsNewer("(devel)", "v1.0.0")).To(BeTrue())
	})
})
