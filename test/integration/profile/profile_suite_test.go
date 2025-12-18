// ABOUTME: Profile integration test suite
// ABOUTME: Uses Ginkgo BDD framework for testing profile package
package profile_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestProfile(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Profile Integration Suite")
}
