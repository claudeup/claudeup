// ABOUTME: Acceptance tests for mcp list command
// ABOUTME: Tests MCP server listing including disabled server display
package acceptance

import (
	"os"
	"path/filepath"

	"github.com/claudeup/claudeup/v4/test/helpers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("mcp list", func() {
	var env *helpers.TestEnv

	BeforeEach(func() {
		env = helpers.NewTestEnv(binaryPath)
	})

	Describe("with no plugins", func() {
		It("shows empty message", func() {
			result := env.Run("mcp", "list")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("No MCP servers found"))
		})
	})

	Describe("with MCP servers", func() {
		var pluginPath string

		BeforeEach(func() {
			// Create plugin directory
			pluginPath = filepath.Join(env.ClaudeDir, "plugins", "cache", "test-plugin")
			Expect(os.MkdirAll(pluginPath, 0755)).To(Succeed())

			// Create installed plugins registry
			env.CreateInstalledPlugins(map[string]interface{}{
				"test-plugin@acme-marketplace": []interface{}{
					map[string]interface{}{
						"version":     "1.0.0",
						"installPath": pluginPath,
						"scope":       "user",
					},
				},
			})

			// Enable the plugin in settings
			env.CreateSettings(map[string]bool{
				"test-plugin@acme-marketplace": true,
			})

			// Create plugin manifest with MCP servers
			env.CreatePluginMCPServers(pluginPath, map[string]interface{}{
				"server-a": map[string]interface{}{
					"command": "node",
					"args":    []string{"server-a.js"},
				},
				"server-b": map[string]interface{}{
					"command": "node",
					"args":    []string{"server-b.js"},
				},
			})
		})

		It("lists all enabled servers", func() {
			result := env.Run("mcp", "list")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("server-a"))
			Expect(result.Stdout).To(ContainSubstring("server-b"))
		})

		It("shows server count in header", func() {
			result := env.Run("mcp", "list")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("2"))
		})
	})

	Describe("with disabled MCP servers", func() {
		var pluginPath string

		BeforeEach(func() {
			// Create plugin directory
			pluginPath = filepath.Join(env.ClaudeDir, "plugins", "cache", "test-plugin")
			Expect(os.MkdirAll(pluginPath, 0755)).To(Succeed())

			// Create installed plugins registry
			env.CreateInstalledPlugins(map[string]interface{}{
				"test-plugin@acme-marketplace": []interface{}{
					map[string]interface{}{
						"version":     "1.0.0",
						"installPath": pluginPath,
						"scope":       "user",
					},
				},
			})

			// Enable the plugin in settings
			env.CreateSettings(map[string]bool{
				"test-plugin@acme-marketplace": true,
			})

			// Create plugin manifest with MCP servers
			env.CreatePluginMCPServers(pluginPath, map[string]interface{}{
				"server-a": map[string]interface{}{
					"command": "node",
					"args":    []string{"server-a.js"},
				},
				"server-b": map[string]interface{}{
					"command": "node",
					"args":    []string{"server-b.js"},
				},
			})

			// Disable server-b
			env.SetDisabledMCPServers([]string{
				"test-plugin@acme-marketplace:server-b",
			})
		})

		It("shows disabled servers with indicator", func() {
			result := env.Run("mcp", "list")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("server-b"))
			Expect(result.Stdout).To(ContainSubstring("disabled"))
		})

		It("shows both enabled and disabled counts in header", func() {
			result := env.Run("mcp", "list")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("1 enabled"))
			Expect(result.Stdout).To(ContainSubstring("1 disabled"))
		})

		It("shows enabled servers with success indicator", func() {
			result := env.Run("mcp", "list")

			Expect(result.ExitCode).To(Equal(0))
			// server-a should have success checkmark, not disabled
			Expect(result.Stdout).To(ContainSubstring("server-a"))
			Expect(result.Stdout).NotTo(MatchRegexp(`server-a.*disabled`))
		})
	})
})
