package claude_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestClaude(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Claude Integration Suite")
}
