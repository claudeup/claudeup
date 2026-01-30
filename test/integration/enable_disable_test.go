// ABOUTME: Integration tests for enable/disable functionality
// ABOUTME: Tests MCP server enable/disable workflows
package integration

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/claudeup/claudeup/v4/internal/config"
)

var _ = Describe("MCPServerDisableEnable", func() {
	It("disables and re-enables an MCP server", func() {
		cfg := config.DefaultConfig()
		serverRef := "test-plugin@test-marketplace:test-server"

		Expect(cfg.IsMCPServerDisabled(serverRef)).To(BeFalse(), "MCP server should not be disabled initially")

		Expect(cfg.DisableMCPServer(serverRef)).To(BeTrue(), "DisableMCPServer should return true for first disable")
		Expect(cfg.IsMCPServerDisabled(serverRef)).To(BeTrue(), "MCP server should be disabled")

		Expect(cfg.DisableMCPServer(serverRef)).To(BeFalse(), "DisableMCPServer should return false for already disabled server")

		Expect(cfg.EnableMCPServer(serverRef)).To(BeTrue(), "EnableMCPServer should return true")
		Expect(cfg.IsMCPServerDisabled(serverRef)).To(BeFalse(), "MCP server should not be disabled after enable")

		Expect(cfg.EnableMCPServer(serverRef)).To(BeFalse(), "EnableMCPServer should return false for already enabled server")
	})
})
