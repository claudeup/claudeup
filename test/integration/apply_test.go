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

	"github.com/claudeup/claudeup/internal/claude"
	"github.com/claudeup/claudeup/internal/profile"
	"github.com/claudeup/claudeup/internal/secrets"
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
	claudeDir  string
	claudeJSON string
}

func setupApplyTestEnv() *applyTestEnv {
	tmpDir := GinkgoT().TempDir()

	claudeDir := filepath.Join(tmpDir, ".claude")
	pluginsDir := filepath.Join(claudeDir, "plugins")
	err := os.MkdirAll(pluginsDir, 0755)
	Expect(err).NotTo(HaveOccurred())

	claudeJSON := filepath.Join(tmpDir, ".claude.json")

	env := &applyTestEnv{
		claudeDir:  claudeDir,
		claudeJSON: claudeJSON,
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

		result, err := profile.ApplyWithExecutor(p, env.claudeDir, env.claudeJSON, chain, executor)
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

		result, err := profile.ApplyWithExecutor(p, env.claudeDir, env.claudeJSON, chain, executor)
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

		result, err := profile.ApplyWithExecutor(p, env.claudeDir, env.claudeJSON, chain, executor)
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

		result, err := profile.ApplyWithExecutor(p, env.claudeDir, env.claudeJSON, chain, executor)
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

		result, err := profile.ApplyWithExecutor(p, env.claudeDir, env.claudeJSON, chain, executor)
		Expect(err).NotTo(HaveOccurred())

		Expect(executor.HasCommand("plugin", "marketplace", "add")).To(BeTrue(), "Expected marketplace add command. Commands: %v", executor.Commands)
		Expect(result.MarketplacesAdded).To(HaveLen(1))
	})
})

var _ = Describe("ApplyWithSecrets", func() {
	var env *applyTestEnv

	BeforeEach(func() {
		env = setupApplyTestEnv()

		os.Setenv("TEST_API_KEY", "secret-value-123")
		DeferCleanup(func() {
			os.Unsetenv("TEST_API_KEY")
		})
	})

	It("resolves secrets in MCP server args", func() {
		p := &profile.Profile{
			Name: "test",
			MCPServers: []profile.MCPServer{
				{
					Name:    "secret-mcp",
					Command: "npx",
					Args:    []string{"-y", "package", "$TEST_API_KEY"},
					Secrets: map[string]profile.SecretRef{
						"TEST_API_KEY": {
							Description: "Test API key",
							Sources: []profile.SecretSource{
								{Type: "env", Key: "TEST_API_KEY"},
							},
						},
					},
				},
			},
		}

		executor := NewMockExecutor()
		chain := secrets.NewChain(secrets.NewEnvResolver())

		result, err := profile.ApplyWithExecutor(p, env.claudeDir, env.claudeJSON, chain, executor)
		Expect(err).NotTo(HaveOccurred())

		Expect(result.MCPServersInstalled).To(HaveLen(1))

		found := false
		for _, cmd := range executor.Commands {
			for _, arg := range cmd {
				if arg == "secret-value-123" {
					found = true
					break
				}
			}
		}
		Expect(found).To(BeTrue(), "Expected resolved secret in command args. Commands: %v", executor.Commands)
	})
})

var _ = Describe("ApplyMissingSecretFails", func() {
	var env *applyTestEnv

	BeforeEach(func() {
		env = setupApplyTestEnv()
	})

	It("fails when secret cannot be resolved", func() {
		p := &profile.Profile{
			Name: "test",
			MCPServers: []profile.MCPServer{
				{
					Name:    "secret-mcp",
					Command: "npx",
					Args:    []string{"package", "$MISSING_SECRET"},
					Secrets: map[string]profile.SecretRef{
						"MISSING_SECRET": {
							Sources: []profile.SecretSource{
								{Type: "env", Key: "MISSING_SECRET"},
							},
						},
					},
				},
			},
		}

		executor := NewMockExecutor()
		chain := secrets.NewChain(secrets.NewEnvResolver())

		_, err := profile.ApplyWithExecutor(p, env.claudeDir, env.claudeJSON, chain, executor)
		Expect(err).To(HaveOccurred())
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

		_, err := profile.ApplyWithExecutor(p, env.claudeDir, env.claudeJSON, chain, executor)
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

		result, err := profile.ApplyWithExecutor(p, env.claudeDir, env.claudeJSON, chain, executor)
		Expect(err).NotTo(HaveOccurred())

		// Plugin is already disabled, should be in PluginsAlreadyRemoved
		Expect(result.PluginsAlreadyRemoved).To(HaveLen(1))
		Expect(result.PluginsAlreadyRemoved).To(ContainElement("plugin-a@marketplace"))
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

		result, err := profile.ApplyWithExecutor(p, env.claudeDir, env.claudeJSON, chain, executor)
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

		result, err := profile.ApplyWithExecutor(p, env.claudeDir, env.claudeJSON, chain, executor)
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

		result, err := profile.ApplyWithExecutor(p, env.claudeDir, env.claudeJSON, chain, executor)
		Expect(err).NotTo(HaveOccurred())

		Expect(result.Errors).To(HaveLen(1))
		Expect(result.PluginsAlreadyPresent).To(BeEmpty())
		Expect(result.PluginsInstalled).To(BeEmpty())
	})
})
