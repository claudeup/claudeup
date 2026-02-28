// ABOUTME: Acceptance tests for profile status live effective configuration
// ABOUTME: Verifies status shows live settings across all scopes with tracking annotations
package acceptance

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/claudeup/claudeup/v5/test/helpers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("profile status", func() {
	var env *helpers.TestEnv

	BeforeEach(func() {
		env = helpers.NewTestEnv(binaryPath)
	})

	Describe("live effective configuration", func() {
		Context("with user-scope plugins only", func() {
			BeforeEach(func() {
				env.CreateSettings(map[string]bool{
					"plugin-a@marketplace":        true,
					"plugin-b@marketplace":        true,
					"disabled-plugin@marketplace": false,
				})
			})

			It("shows user-scope plugins", func() {
				result := env.Run("profile", "status")

				Expect(result.ExitCode).To(Equal(0))
				Expect(result.Stdout).To(ContainSubstring("User scope"))
				Expect(result.Stdout).To(ContainSubstring("plugin-a@marketplace"))
				Expect(result.Stdout).To(ContainSubstring("plugin-b@marketplace"))
			})

			It("shows disabled plugins", func() {
				result := env.Run("profile", "status")

				Expect(result.ExitCode).To(Equal(0))
				Expect(result.Stdout).To(ContainSubstring("Disabled"))
				Expect(result.Stdout).To(ContainSubstring("disabled-plugin@marketplace"))
			})

		})

		Context("with multi-scope plugins", func() {
			var projectDir string

			BeforeEach(func() {
				projectDir = env.ProjectDir("multi-scope-test")

				// User scope
				env.CreateSettings(map[string]bool{
					"user-plugin@marketplace": true,
				})

				// Project scope
				env.CreateProjectScopeSettings(projectDir, map[string]bool{
					"proj-plugin@marketplace": true,
				})
			})

			It("shows plugins from both scopes", func() {
				result := env.RunInDir(projectDir, "profile", "status")

				Expect(result.ExitCode).To(Equal(0))
				Expect(result.Stdout).To(ContainSubstring("User scope"))
				Expect(result.Stdout).To(ContainSubstring("user-plugin@marketplace"))
				Expect(result.Stdout).To(ContainSubstring("Project scope"))
				Expect(result.Stdout).To(ContainSubstring("proj-plugin@marketplace"))
			})
		})

		Context("with user-scope MCP servers", func() {
			BeforeEach(func() {
				claudeJSON := map[string]any{
					"mcpServers": map[string]any{
						"test-server": map[string]any{
							"command": "npx",
							"args":    []string{"test-server"},
						},
					},
				}
				data, err := json.MarshalIndent(claudeJSON, "", "  ")
				Expect(err).NotTo(HaveOccurred())
				Expect(os.WriteFile(filepath.Join(env.ClaudeDir, ".claude.json"), data, 0644)).To(Succeed())
			})

			It("shows MCP servers in user scope", func() {
				result := env.Run("profile", "status")

				Expect(result.ExitCode).To(Equal(0))
				Expect(result.Stdout).To(ContainSubstring("User scope"))
				Expect(result.Stdout).To(ContainSubstring("MCP Servers:"))
				Expect(result.Stdout).To(ContainSubstring("test-server"))
			})
		})

		Context("with project-scope MCP servers", func() {
			var projectDir string

			BeforeEach(func() {
				projectDir = env.ProjectDir("mcp-project-test")

				// Create .claude dir so project scope is checked
				Expect(os.MkdirAll(filepath.Join(projectDir, ".claude"), 0755)).To(Succeed())

				mcpJSON := map[string]any{
					"mcpServers": map[string]any{
						"project-server": map[string]any{
							"command": "node",
							"args":    []string{"server.js"},
						},
					},
				}
				data, err := json.MarshalIndent(mcpJSON, "", "  ")
				Expect(err).NotTo(HaveOccurred())
				Expect(os.WriteFile(filepath.Join(projectDir, ".mcp.json"), data, 0644)).To(Succeed())
			})

			It("shows MCP servers in project scope", func() {
				result := env.RunInDir(projectDir, "profile", "status")

				Expect(result.ExitCode).To(Equal(0))
				Expect(result.Stdout).To(ContainSubstring("Project scope"))
				Expect(result.Stdout).To(ContainSubstring("MCP Servers:"))
				Expect(result.Stdout).To(ContainSubstring("project-server"))
			})
		})

		Context("with user-scope extensions", func() {
			BeforeEach(func() {
				// Create extension source file in claudeupHome
				extAgentsDir := filepath.Join(env.ClaudeupDir, "ext", "agents")
				Expect(os.MkdirAll(extAgentsDir, 0755)).To(Succeed())
				Expect(os.WriteFile(filepath.Join(extAgentsDir, "test-agent.md"), []byte("# Test Agent"), 0644)).To(Succeed())

				// Create symlink in active directory
				activeAgentsDir := filepath.Join(env.ClaudeDir, "agents")
				Expect(os.MkdirAll(activeAgentsDir, 0755)).To(Succeed())
				Expect(os.Symlink(
					filepath.Join(extAgentsDir, "test-agent.md"),
					filepath.Join(activeAgentsDir, "test-agent.md"),
				)).To(Succeed())

				// Create enabled.json
				enabledConfig := map[string]map[string]bool{
					"agents": {"test-agent.md": true},
				}
				data, err := json.MarshalIndent(enabledConfig, "", "  ")
				Expect(err).NotTo(HaveOccurred())
				Expect(os.WriteFile(filepath.Join(env.ClaudeupDir, "enabled.json"), data, 0644)).To(Succeed())
			})

			It("shows extensions in user scope", func() {
				result := env.Run("profile", "status")

				Expect(result.ExitCode).To(Equal(0))
				Expect(result.Stdout).To(ContainSubstring("User scope"))
				Expect(result.Stdout).To(ContainSubstring("Extensions:"))
				Expect(result.Stdout).To(ContainSubstring("Agents:"))
				Expect(result.Stdout).To(ContainSubstring("test-agent.md"))
			})
		})

		Context("with MCP servers but no plugins", func() {
			BeforeEach(func() {
				claudeJSON := map[string]any{
					"mcpServers": map[string]any{
						"mcp-only-server": map[string]any{
							"command": "npx",
							"args":    []string{"mcp-server"},
						},
					},
				}
				data, err := json.MarshalIndent(claudeJSON, "", "  ")
				Expect(err).NotTo(HaveOccurred())
				Expect(os.WriteFile(filepath.Join(env.ClaudeDir, ".claude.json"), data, 0644)).To(Succeed())
			})

			It("shows scope section for MCP-only scope", func() {
				result := env.Run("profile", "status")

				Expect(result.ExitCode).To(Equal(0))
				Expect(result.Stdout).To(ContainSubstring("User scope"))
				Expect(result.Stdout).To(ContainSubstring("MCP Servers:"))
				Expect(result.Stdout).To(ContainSubstring("mcp-only-server"))
				Expect(result.Stdout).NotTo(ContainSubstring("No configuration"))
			})
		})

		Context("with project-scope extensions", func() {
			var projectDir string

			BeforeEach(func() {
				projectDir = env.ProjectDir("ext-project-test")

				// Create project-scoped agent (regular file, not symlink)
				agentsDir := filepath.Join(projectDir, ".claude", "agents")
				Expect(os.MkdirAll(agentsDir, 0755)).To(Succeed())
				Expect(os.WriteFile(filepath.Join(agentsDir, "project-agent.md"), []byte("# Project Agent"), 0644)).To(Succeed())
			})

			It("shows extensions in project scope", func() {
				result := env.RunInDir(projectDir, "profile", "status")

				Expect(result.ExitCode).To(Equal(0))
				Expect(result.Stdout).To(ContainSubstring("Project scope"))
				Expect(result.Stdout).To(ContainSubstring("Extensions:"))
				Expect(result.Stdout).To(ContainSubstring("Agents:"))
				Expect(result.Stdout).To(ContainSubstring("project-agent.md"))
			})
		})

		Context("with plugins, MCP servers, and extensions in one scope", func() {
			BeforeEach(func() {
				// User-scope plugin
				settingsJSON := map[string]any{
					"enabledPlugins": map[string]any{
						"test-plugin@test-marketplace": true,
					},
				}
				data, err := json.MarshalIndent(settingsJSON, "", "  ")
				Expect(err).NotTo(HaveOccurred())
				Expect(os.WriteFile(filepath.Join(env.ClaudeDir, "settings.json"), data, 0644)).To(Succeed())

				// User-scope MCP server
				claudeJSON := map[string]any{
					"mcpServers": map[string]any{
						"combo-server": map[string]any{
							"command": "npx",
							"args":    []string{"combo-mcp"},
						},
					},
				}
				data, err = json.MarshalIndent(claudeJSON, "", "  ")
				Expect(err).NotTo(HaveOccurred())
				Expect(os.WriteFile(filepath.Join(env.ClaudeDir, ".claude.json"), data, 0644)).To(Succeed())

				// User-scope extension
				extAgentsDir := filepath.Join(env.ClaudeupDir, "ext", "agents")
				Expect(os.MkdirAll(extAgentsDir, 0755)).To(Succeed())
				Expect(os.WriteFile(filepath.Join(extAgentsDir, "combo-agent.md"), []byte("# Combo"), 0644)).To(Succeed())

				activeAgentsDir := filepath.Join(env.ClaudeDir, "agents")
				Expect(os.MkdirAll(activeAgentsDir, 0755)).To(Succeed())
				Expect(os.Symlink(
					filepath.Join(extAgentsDir, "combo-agent.md"),
					filepath.Join(activeAgentsDir, "combo-agent.md"),
				)).To(Succeed())

				enabledConfig := map[string]map[string]bool{
					"agents": {"combo-agent.md": true},
				}
				data, err = json.MarshalIndent(enabledConfig, "", "  ")
				Expect(err).NotTo(HaveOccurred())
				Expect(os.WriteFile(filepath.Join(env.ClaudeupDir, "enabled.json"), data, 0644)).To(Succeed())
			})

			It("shows all content types under user scope", func() {
				result := env.Run("profile", "status")

				Expect(result.ExitCode).To(Equal(0))
				Expect(result.Stdout).To(ContainSubstring("User scope"))
				Expect(result.Stdout).To(ContainSubstring("Plugins:"))
				Expect(result.Stdout).To(ContainSubstring("test-plugin@test-marketplace"))
				Expect(result.Stdout).To(ContainSubstring("MCP Servers:"))
				Expect(result.Stdout).To(ContainSubstring("combo-server"))
				Expect(result.Stdout).To(ContainSubstring("Extensions:"))
				Expect(result.Stdout).To(ContainSubstring("combo-agent.md"))
			})
		})

		Context("with no configuration at any scope", func() {
			It("shows empty configuration message", func() {
				result := env.Run("profile", "status")

				Expect(result.ExitCode).To(Equal(0))
				Expect(result.Stdout).To(ContainSubstring("No configuration"))
			})
		})
	})
})
