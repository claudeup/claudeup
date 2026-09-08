// ABOUTME: Integration tests for profile Apply flow
// ABOUTME: Uses mock executor to verify command sequences
package integration

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/claudeup/claudeup/v5/internal/claude"
	"github.com/claudeup/claudeup/v5/internal/profile"
	"github.com/claudeup/claudeup/v5/internal/secrets"
)

// MockExecutor records commands for verification
type MockExecutor struct {
	Commands [][]string
	Errors   map[string]error  // command prefix -> error to return
	Outputs  map[string]string // command prefix -> output to return
}

func NewMockExecutor() *MockExecutor {
	return &MockExecutor{
		Commands: [][]string{},
		Errors:   make(map[string]error),
		Outputs:  make(map[string]string),
	}
}

func (m *MockExecutor) Run(args ...string) error {
	m.Commands = append(m.Commands, args)

	// Check if we should return an error
	cmdKey := strings.Join(args[:min(3, len(args))], " ")
	if err, ok := m.Errors[cmdKey]; ok {
		return err
	}
	return nil
}

func (m *MockExecutor) RunWithOutput(args ...string) (string, error) {
	m.Commands = append(m.Commands, args)

	// Check if we should return an error or custom output
	cmdKey := strings.Join(args[:min(3, len(args))], " ")
	output := "✔ Success\n"
	if customOutput, ok := m.Outputs[cmdKey]; ok {
		output = customOutput
	}
	if err, ok := m.Errors[cmdKey]; ok {
		return output, err
	}
	return output, nil
}

func (m *MockExecutor) CommandCount() int {
	return len(m.Commands)
}

func (m *MockExecutor) HasCommand(prefix ...string) bool {
	for _, cmd := range m.Commands {
		if len(cmd) >= len(prefix) {
			match := true
			for i, p := range prefix {
				if cmd[i] != p {
					match = false
					break
				}
			}
			if match {
				return true
			}
		}
	}
	return false
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Test environment helpers

type applyTestEnv struct {
	claudeDir    string
	claudeupHome string
	claudeJSON   string
}

func setupApplyTestEnv() *applyTestEnv {
	tmpDir := GinkgoT().TempDir()

	claudeDir := filepath.Join(tmpDir, ".claude")
	claudeupHome := filepath.Join(tmpDir, ".claudeup")
	pluginsDir := filepath.Join(claudeDir, "plugins")
	err := os.MkdirAll(pluginsDir, 0755)
	Expect(err).NotTo(HaveOccurred())
	err = os.MkdirAll(claudeupHome, 0755)
	Expect(err).NotTo(HaveOccurred())

	claudeJSON := filepath.Join(tmpDir, ".claude.json")

	env := &applyTestEnv{
		claudeDir:    claudeDir,
		claudeupHome: claudeupHome,
		claudeJSON:   claudeJSON,
	}

	env.createPluginRegistry(map[string]interface{}{})
	env.createMarketplaceRegistry(map[string]interface{}{})
	env.createClaudeJSON(map[string]interface{}{})

	return env
}

func (e *applyTestEnv) createPluginRegistry(plugins map[string]interface{}) {
	// Convert to V2 format (plugins as arrays with scope)
	pluginsV2 := make(map[string]interface{})
	for name, meta := range plugins {
		metaMap, ok := meta.(map[string]interface{})
		if !ok {
			metaMap = make(map[string]interface{})
		}
		if _, hasScope := metaMap["scope"]; !hasScope {
			metaMap["scope"] = "user"
		}
		pluginsV2[name] = []interface{}{metaMap}
	}
	data := map[string]interface{}{
		"version": 2,
		"plugins": pluginsV2,
	}
	e.writeJSON(filepath.Join(e.claudeDir, "plugins", "installed_plugins.json"), data)
}

func (e *applyTestEnv) createMarketplaceRegistry(marketplaces map[string]interface{}) {
	e.writeJSON(filepath.Join(e.claudeDir, "plugins", "known_marketplaces.json"), marketplaces)
}

func (e *applyTestEnv) createSettings(enabledPlugins map[string]bool) {
	data := map[string]interface{}{
		"enabledPlugins": enabledPlugins,
	}
	e.writeJSON(filepath.Join(e.claudeDir, "settings.json"), data)
}

func (e *applyTestEnv) createClaudeJSON(data map[string]interface{}) {
	e.writeJSON(e.claudeJSON, data)
}

func (e *applyTestEnv) writeJSON(path string, data interface{}) {
	bytes, err := json.MarshalIndent(data, "", "  ")
	Expect(err).NotTo(HaveOccurred())
	err = os.WriteFile(path, bytes, 0644)
	Expect(err).NotTo(HaveOccurred())
}

var _ = Describe("ApplyInstallsPlugins", func() {
	var env *applyTestEnv

	BeforeEach(func() {
		env = setupApplyTestEnv()
	})

	It("installs plugins", func() {
		p := &profile.Profile{
			Name:    "test",
			Plugins: []string{"plugin-a@marketplace"},
		}

		executor := NewMockExecutor()
		chain := secrets.NewChain(secrets.NewEnvResolver())

		result, err := profile.ApplyWithExecutor(p, env.claudeDir, env.claudeJSON, env.claudeupHome, chain, executor)
		Expect(err).NotTo(HaveOccurred())

		Expect(executor.HasCommand("plugin", "install", "plugin-a@marketplace")).To(BeTrue(), "Expected plugin install command. Commands: %v", executor.Commands)
		Expect(result.PluginsInstalled).To(HaveLen(1))
	})
})

var _ = Describe("ApplyRemovesPlugins", func() {
	var env *applyTestEnv

	BeforeEach(func() {
		env = setupApplyTestEnv()

		env.createPluginRegistry(map[string]interface{}{
			"plugin-a@marketplace": map[string]interface{}{"version": "1.0"},
			"plugin-b@marketplace": map[string]interface{}{"version": "1.0"},
		})

		// Enable both plugins in settings.json
		env.createSettings(map[string]bool{
			"plugin-a@marketplace": true,
			"plugin-b@marketplace": true,
		})
	})

	It("removes plugins not in profile", func() {
		p := &profile.Profile{
			Name:    "test",
			Plugins: []string{"plugin-a@marketplace"},
		}

		executor := NewMockExecutor()
		chain := secrets.NewChain(secrets.NewEnvResolver())

		result, err := profile.ApplyWithExecutor(p, env.claudeDir, env.claudeJSON, env.claudeupHome, chain, executor)
		Expect(err).NotTo(HaveOccurred())

		// Verify plugin was removed from result
		Expect(result.PluginsRemoved).To(HaveLen(1))
		Expect(result.PluginsRemoved).To(ContainElement("plugin-b@marketplace"))

		// Verify settings.json was updated correctly
		settings, err := claude.LoadSettings(env.claudeDir)
		Expect(err).NotTo(HaveOccurred())
		Expect(settings.IsPluginEnabled("plugin-a@marketplace")).To(BeTrue(), "plugin-a should still be enabled")
		Expect(settings.IsPluginEnabled("plugin-b@marketplace")).To(BeFalse(), "plugin-b should be disabled")
	})
})

var _ = Describe("ApplyAddsMCPServers", func() {
	var env *applyTestEnv

	BeforeEach(func() {
		env = setupApplyTestEnv()
	})

	It("adds MCP servers", func() {
		p := &profile.Profile{
			Name: "test",
			MCPServers: []profile.MCPServer{
				{
					Name:    "test-mcp",
					Command: "npx",
					Args:    []string{"-y", "test-package"},
					Scope:   "user",
				},
			},
		}

		executor := NewMockExecutor()
		chain := secrets.NewChain(secrets.NewEnvResolver())

		result, err := profile.ApplyWithExecutor(p, env.claudeDir, env.claudeJSON, env.claudeupHome, chain, executor)
		Expect(err).NotTo(HaveOccurred())

		Expect(executor.HasCommand("mcp", "add", "test-mcp")).To(BeTrue(), "Expected mcp add command. Commands: %v", executor.Commands)
		Expect(result.MCPServersInstalled).To(HaveLen(1))
	})
})

var _ = Describe("ApplyRemovesMCPServers", func() {
	var env *applyTestEnv

	BeforeEach(func() {
		env = setupApplyTestEnv()

		env.createClaudeJSON(map[string]interface{}{
			"mcpServers": map[string]interface{}{
				"old-mcp": map[string]interface{}{
					"command": "node",
					"args":    []string{"server.js"},
				},
			},
		})
	})

	It("removes MCP servers not in profile", func() {
		p := &profile.Profile{
			Name:       "test",
			MCPServers: []profile.MCPServer{},
		}

		executor := NewMockExecutor()
		chain := secrets.NewChain(secrets.NewEnvResolver())

		result, err := profile.ApplyWithExecutor(p, env.claudeDir, env.claudeJSON, env.claudeupHome, chain, executor)
		Expect(err).NotTo(HaveOccurred())

		Expect(executor.HasCommand("mcp", "remove", "old-mcp")).To(BeTrue(), "Expected mcp remove command. Commands: %v", executor.Commands)
		Expect(result.MCPServersRemoved).To(HaveLen(1))
	})
})

var _ = Describe("ApplyAddsMarketplaces", func() {
	var env *applyTestEnv

	BeforeEach(func() {
		env = setupApplyTestEnv()
	})

	It("adds marketplaces", func() {
		p := &profile.Profile{
			Name: "test",
			Marketplaces: []profile.Marketplace{
				{Source: "github", Repo: "test-org/test-marketplace"},
			},
		}

		executor := NewMockExecutor()
		chain := secrets.NewChain(secrets.NewEnvResolver())

		result, err := profile.ApplyWithExecutor(p, env.claudeDir, env.claudeJSON, env.claudeupHome, chain, executor)
		Expect(err).NotTo(HaveOccurred())

		Expect(executor.HasCommand("plugin", "marketplace", "add")).To(BeTrue(), "Expected marketplace add command. Commands: %v", executor.Commands)
		Expect(result.MarketplacesAdded).To(HaveLen(1))
	})
})

// Secrets referenced from MCP server args must never be passed to
// `claude mcp add` in plaintext: the argv is visible to every local user via
// ps and /proc/<pid>/cmdline, and Claude Code stores it verbatim so the value
// would also sit in the MCP server's own command line on every launch.
// Instead, claudeup writes a ${KEY} placeholder that Claude Code expands from
// its parent environment at launch (issue #312).
var _ = Describe("ApplySecretPlaceholders", func() {
	const secretValue = "secret-value-123"

	var env *applyTestEnv

	// mcpAddArgs returns the argv of every `mcp add` command recorded by the
	// executor, joined so a whole-arg match is easy to assert on.
	mcpAddArgs := func(executor *MockExecutor, server string) []string {
		var found []string
		for _, cmd := range executor.Commands {
			if len(cmd) >= 3 && cmd[0] == "mcp" && cmd[1] == "add" && cmd[2] == server {
				found = append(found, strings.Join(cmd, " "))
			}
		}
		return found
	}

	// allArgs flattens every recorded command so a secret can be searched
	// for across the whole apply, not only the mcp add calls.
	allArgs := func(executor *MockExecutor) []string {
		var flat []string
		for _, cmd := range executor.Commands {
			flat = append(flat, cmd...)
		}
		return flat
	}

	secretServer := func(name, key string) profile.MCPServer {
		return profile.MCPServer{
			Name:    name,
			Command: "npx",
			Args:    []string{"-y", "package", "--token", "$" + key},
			Secrets: map[string]profile.SecretRef{
				key: {
					Description: "Test API key",
					Sources: []profile.SecretSource{
						{Type: "env", Key: key},
					},
				},
			},
		}
	}

	BeforeEach(func() {
		env = setupApplyTestEnv()

		os.Setenv("TEST_API_KEY", secretValue)
		DeferCleanup(func() {
			os.Unsetenv("TEST_API_KEY")
		})
	})

	It("passes a ${KEY} placeholder instead of the resolved value at user scope", func() {
		p := &profile.Profile{
			Name:       "test",
			MCPServers: []profile.MCPServer{secretServer("secret-mcp", "TEST_API_KEY")},
		}

		executor := NewMockExecutor()
		chain := secrets.NewChain(secrets.NewEnvResolver())

		result, err := profile.ApplyWithExecutor(p, env.claudeDir, env.claudeJSON, env.claudeupHome, chain, executor)
		Expect(err).NotTo(HaveOccurred())
		Expect(result.MCPServersInstalled).To(Equal([]string{"secret-mcp"}))
		Expect(result.Warnings).To(BeEmpty())

		cmds := mcpAddArgs(executor, "secret-mcp")
		Expect(cmds).To(HaveLen(1))
		Expect(cmds[0]).To(ContainSubstring(" --token ${TEST_API_KEY}"))
		Expect(allArgs(executor)).NotTo(ContainElement(secretValue),
			"resolved secret must not appear in any CLI argv. Commands: %v", executor.Commands)
	})

	It("does not fall back to the environment for a $VAR arg with no secrets entry", func() {
		p := &profile.Profile{
			Name: "test",
			MCPServers: []profile.MCPServer{
				{
					Name:    "plain-mcp",
					Command: "npx",
					Args:    []string{"-y", "package", "$TEST_API_KEY"},
				},
			},
		}

		executor := NewMockExecutor()
		chain := secrets.NewChain(secrets.NewEnvResolver())

		_, err := profile.ApplyWithExecutor(p, env.claudeDir, env.claudeJSON, env.claudeupHome, chain, executor)
		Expect(err).NotTo(HaveOccurred())

		cmds := mcpAddArgs(executor, "plain-mcp")
		Expect(cmds).To(HaveLen(1))
		Expect(cmds[0]).To(HaveSuffix(" ${TEST_API_KEY}"))
		Expect(allArgs(executor)).NotTo(ContainElement(secretValue))
	})

	It("warns but still installs with a placeholder when a secret cannot be resolved", func() {
		p := &profile.Profile{
			Name:       "test",
			MCPServers: []profile.MCPServer{secretServer("secret-mcp", "MISSING_SECRET")},
		}

		executor := NewMockExecutor()
		chain := secrets.NewChain(secrets.NewEnvResolver())

		result, err := profile.ApplyWithExecutor(p, env.claudeDir, env.claudeJSON, env.claudeupHome, chain, executor)
		Expect(err).NotTo(HaveOccurred())
		Expect(result.Errors).To(BeEmpty())
		Expect(result.MCPServersInstalled).To(Equal([]string{"secret-mcp"}))

		Expect(result.Warnings).To(HaveLen(1))
		Expect(result.Warnings[0].Error()).To(ContainSubstring("MISSING_SECRET"))

		cmds := mcpAddArgs(executor, "secret-mcp")
		Expect(cmds).To(HaveLen(1))
		Expect(cmds[0]).To(HaveSuffix(" --token ${MISSING_SECRET}"))
	})

	It("warns when the secret is found but the placeholder variable is not exported", func() {
		// Documented shape: the secrets map key (API_KEY) differs from the
		// env source (TEST_API_KEY). Claude Code expands ${API_KEY}, so the
		// exported TEST_API_KEY does not help and the preflight must say so.
		os.Unsetenv("API_KEY")
		p := &profile.Profile{
			Name: "test",
			MCPServers: []profile.MCPServer{
				{
					Name:    "secret-mcp",
					Command: "npx",
					Args:    []string{"--token", "$API_KEY"},
					Secrets: map[string]profile.SecretRef{
						"API_KEY": {Sources: []profile.SecretSource{{Type: "env", Key: "TEST_API_KEY"}}},
					},
				},
			},
		}

		executor := NewMockExecutor()
		chain := secrets.NewChain(secrets.NewEnvResolver())

		result, err := profile.ApplyWithExecutor(p, env.claudeDir, env.claudeJSON, env.claudeupHome, chain, executor)
		Expect(err).NotTo(HaveOccurred())
		Expect(result.MCPServersInstalled).To(Equal([]string{"secret-mcp"}))

		Expect(result.Warnings).To(HaveLen(1))
		Expect(result.Warnings[0].Error()).To(SatisfyAll(
			ContainSubstring(`"API_KEY"`),
			ContainSubstring("not exported"),
			Not(ContainSubstring(secretValue)),
		))

		cmds := mcpAddArgs(executor, "secret-mcp")
		Expect(cmds).To(HaveLen(1))
		Expect(cmds[0]).To(HaveSuffix(" --token ${API_KEY}"))
		Expect(allArgs(executor)).NotTo(ContainElement(secretValue))
	})

	Describe("across all scopes", func() {
		var (
			projectDir string
			p          *profile.Profile
		)

		BeforeEach(func() {
			projectDir = GinkgoT().TempDir()
			Expect(os.MkdirAll(filepath.Join(projectDir, ".claude"), 0755)).To(Succeed())

			p = &profile.Profile{
				Name: "all-scopes",
				PerScope: &profile.PerScopeSettings{
					User:    &profile.ScopeSettings{MCPServers: []profile.MCPServer{secretServer("user-mcp", "TEST_API_KEY")}},
					Project: &profile.ScopeSettings{MCPServers: []profile.MCPServer{secretServer("project-mcp", "TEST_API_KEY")}},
					Local:   &profile.ScopeSettings{MCPServers: []profile.MCPServer{secretServer("local-mcp", "TEST_API_KEY")}},
				},
			}
		})

		It("never places a resolved value in any executor command on the sequential path", func() {
			executor := NewMockExecutor()
			chain := secrets.NewChain(secrets.NewEnvResolver())

			result, err := profile.ApplyAllScopes(p, env.claudeDir, env.claudeJSON, projectDir, env.claudeupHome, chain, &profile.ApplyAllScopesOptions{
				Executor: executor,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.MCPServersInstalled).To(ConsistOf("user-mcp", "project-mcp", "local-mcp"))

			Expect(allArgs(executor)).NotTo(ContainElement(secretValue),
				"resolved secret must not appear in any CLI argv. Commands: %v", executor.Commands)
			for _, server := range []string{"user-mcp", "local-mcp"} {
				cmds := mcpAddArgs(executor, server)
				Expect(cmds).To(HaveLen(1), "expected one mcp add for %s", server)
				Expect(cmds[0]).To(HaveSuffix(" --token ${TEST_API_KEY}"))
			}

			// Project scope is file-based: the placeholder must be written to
			// .mcp.json args too, since Claude Code does not expand bare $VAR.
			data, err := os.ReadFile(filepath.Join(projectDir, ".mcp.json"))
			Expect(err).NotTo(HaveOccurred())
			Expect(string(data)).To(ContainSubstring(`"${TEST_API_KEY}"`))
			Expect(string(data)).NotTo(ContainSubstring(secretValue))
		})

		It("runs the secret preflight for project-scope servers written to .mcp.json", func() {
			os.Unsetenv("PROJECT_ONLY_KEY")
			projectOnly := &profile.Profile{
				Name: "project-only",
				PerScope: &profile.PerScopeSettings{
					Project: &profile.ScopeSettings{MCPServers: []profile.MCPServer{secretServer("project-mcp", "PROJECT_ONLY_KEY")}},
				},
			}

			executor := NewMockExecutor()
			chain := secrets.NewChain(secrets.NewEnvResolver())

			result, err := profile.ApplyAllScopes(projectOnly, env.claudeDir, env.claudeJSON, projectDir, env.claudeupHome, chain, &profile.ApplyAllScopesOptions{
				Executor: executor,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.MCPServersInstalled).To(Equal([]string{"project-mcp"}))

			Expect(result.Warnings).To(HaveLen(1))
			Expect(result.Warnings[0].Error()).To(ContainSubstring(`"PROJECT_ONLY_KEY"`))
		})

		It("never places a resolved value in any executor command on the concurrent path", func() {
			executor := NewMockExecutor()
			chain := secrets.NewChain(secrets.NewEnvResolver())

			result, err := profile.ApplyAllScopes(p, env.claudeDir, env.claudeJSON, projectDir, env.claudeupHome, chain, &profile.ApplyAllScopesOptions{
				Executor:     executor,
				Output:       GinkgoWriter,
				ShowProgress: true,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.MCPServersInstalled).To(ConsistOf("user-mcp", "project-mcp", "local-mcp"))

			Expect(allArgs(executor)).NotTo(ContainElement(secretValue),
				"resolved secret must not appear in any CLI argv. Commands: %v", executor.Commands)
			for _, server := range []string{"user-mcp", "local-mcp"} {
				cmds := mcpAddArgs(executor, server)
				Expect(cmds).To(HaveLen(1), "expected one mcp add for %s", server)
				Expect(cmds[0]).To(HaveSuffix(" --token ${TEST_API_KEY}"))
			}
		})
	})
})

var _ = Describe("ApplyCommandOrder", func() {
	var env *applyTestEnv

	BeforeEach(func() {
		env = setupApplyTestEnv()

		env.createPluginRegistry(map[string]interface{}{
			"old-plugin@marketplace": map[string]interface{}{"version": "1.0"},
		})
	})

	It("executes commands in correct order", func() {
		p := &profile.Profile{
			Name: "test",
			Marketplaces: []profile.Marketplace{
				{Source: "github", Repo: "new-marketplace"},
			},
			Plugins: []string{"new-plugin@marketplace"},
		}

		executor := NewMockExecutor()
		chain := secrets.NewChain(secrets.NewEnvResolver())

		_, err := profile.ApplyWithExecutor(p, env.claudeDir, env.claudeJSON, env.claudeupHome, chain, executor)
		Expect(err).NotTo(HaveOccurred())

		marketplaceIdx := -1
		installIdx := -1

		for i, cmd := range executor.Commands {
			if len(cmd) >= 2 {
				if cmd[0] == "plugin" && cmd[1] == "marketplace" {
					marketplaceIdx = i
				}
				if cmd[0] == "plugin" && cmd[1] == "install" {
					installIdx = i
				}
			}
		}

		Expect(marketplaceIdx).NotTo(Equal(-1), "Expected marketplace add command")
		Expect(installIdx).NotTo(Equal(-1), "Expected plugin install command")

		// Plugin removal happens via settings.json (not executor commands)
		// Only constraint: marketplace add must happen before plugin install
		Expect(marketplaceIdx).To(BeNumerically("<", installIdx), "Expected marketplace add before plugin install")
	})
})

var _ = Describe("ApplyPluginAlreadyUninstalled", func() {
	var env *applyTestEnv

	BeforeEach(func() {
		env = setupApplyTestEnv()

		// Plugin is installed but NOT enabled
		env.createPluginRegistry(map[string]interface{}{
			"plugin-a@marketplace": map[string]interface{}{"version": "1.0"},
		})

		// Create settings.json without the plugin enabled (already disabled)
		env.createSettings(map[string]bool{})
	})

	It("handles already disabled plugins gracefully", func() {
		p := &profile.Profile{
			Name:    "test",
			Plugins: []string{}, // Profile wants no plugins
		}

		executor := NewMockExecutor()
		chain := secrets.NewChain(secrets.NewEnvResolver())

		result, err := profile.ApplyWithExecutor(p, env.claudeDir, env.claudeJSON, env.claudeupHome, chain, executor)
		Expect(err).NotTo(HaveOccurred())

		// Plugin is already disabled (not in settings.json), so nothing to remove
		// Snapshot only reads enabled plugins, so this plugin won't be in the diff
		Expect(result.PluginsRemoved).To(BeEmpty())
		Expect(result.PluginsAlreadyRemoved).To(BeEmpty())
		Expect(result.Errors).To(BeEmpty())
	})
})

var _ = Describe("ApplyPluginAlreadyInstalled", func() {
	var env *applyTestEnv

	BeforeEach(func() {
		env = setupApplyTestEnv()
	})

	It("handles already installed plugins gracefully", func() {
		p := &profile.Profile{
			Name:    "test",
			Plugins: []string{"plugin-a@marketplace"},
		}

		executor := NewMockExecutor()
		executor.Errors["plugin install plugin-a@marketplace"] = fmt.Errorf("install failed")
		executor.Outputs["plugin install plugin-a@marketplace"] = "Error: plugin-a@marketplace is already installed"

		chain := secrets.NewChain(secrets.NewEnvResolver())

		result, err := profile.ApplyWithExecutor(p, env.claudeDir, env.claudeJSON, env.claudeupHome, chain, executor)
		Expect(err).NotTo(HaveOccurred())

		Expect(result.PluginsAlreadyPresent).To(HaveLen(1))
		Expect(result.Errors).To(BeEmpty())
	})
})

var _ = Describe("ApplyAllProfilePluginsAttempted", func() {
	var env *applyTestEnv

	BeforeEach(func() {
		env = setupApplyTestEnv()

		env.createPluginRegistry(map[string]interface{}{
			"plugin-a@marketplace": map[string]interface{}{"version": "1.0"},
		})
	})

	It("attempts to install all profile plugins", func() {
		p := &profile.Profile{
			Name:    "test",
			Plugins: []string{"plugin-a@marketplace", "plugin-b@marketplace"},
		}

		executor := NewMockExecutor()
		chain := secrets.NewChain(secrets.NewEnvResolver())

		result, err := profile.ApplyWithExecutor(p, env.claudeDir, env.claudeJSON, env.claudeupHome, chain, executor)
		Expect(err).NotTo(HaveOccurred())

		Expect(executor.HasCommand("plugin", "install", "plugin-a@marketplace")).To(BeTrue(), "Expected install attempt for plugin-a even though it's in JSON")
		Expect(executor.HasCommand("plugin", "install", "plugin-b@marketplace")).To(BeTrue(), "Expected install attempt for plugin-b")
		Expect(result.PluginsInstalled).To(HaveLen(2))
	})
})

var _ = Describe("ApplyPluginInstallRealError", func() {
	var env *applyTestEnv

	BeforeEach(func() {
		env = setupApplyTestEnv()
	})

	It("tracks real installation errors", func() {
		p := &profile.Profile{
			Name:    "test",
			Plugins: []string{"plugin-a@marketplace"},
		}

		executor := NewMockExecutor()
		executor.Errors["plugin install plugin-a@marketplace"] = fmt.Errorf("install failed")
		executor.Outputs["plugin install plugin-a@marketplace"] = "Error: network timeout while downloading plugin"

		chain := secrets.NewChain(secrets.NewEnvResolver())

		result, err := profile.ApplyWithExecutor(p, env.claudeDir, env.claudeJSON, env.claudeupHome, chain, executor)
		Expect(err).NotTo(HaveOccurred())

		Expect(result.Errors).To(HaveLen(1))
		Expect(result.PluginsAlreadyPresent).To(BeEmpty())
		Expect(result.PluginsInstalled).To(BeEmpty())
	})
})
