// ABOUTME: Integration tests for profile Snapshot functionality
// ABOUTME: Validates profile creation and display with various marketplace types
package integration

import (
	"encoding/json"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/claudeup/claudeup/v4/internal/profile"
)

// Test environment helpers

type snapshotTestEnv struct {
	claudeDir  string
	claudeJSON string
}

func setupSnapshotTestEnv() *snapshotTestEnv {
	tmpDir := GinkgoT().TempDir()

	claudeDir := filepath.Join(tmpDir, ".claude")
	pluginsDir := filepath.Join(claudeDir, "plugins")
	err := os.MkdirAll(pluginsDir, 0755)
	Expect(err).NotTo(HaveOccurred())

	claudeJSON := filepath.Join(tmpDir, ".claude.json")

	return &snapshotTestEnv{
		claudeDir:  claudeDir,
		claudeJSON: claudeJSON,
	}
}

func (e *snapshotTestEnv) createPluginRegistry(plugins map[string]interface{}) {
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

func (e *snapshotTestEnv) createMarketplaceRegistry(marketplaces map[string]interface{}) {
	e.writeJSON(filepath.Join(e.claudeDir, "plugins", "known_marketplaces.json"), marketplaces)
}

func (e *snapshotTestEnv) createClaudeJSON(data map[string]interface{}) {
	e.writeJSON(e.claudeJSON, data)
}

func (e *snapshotTestEnv) writeJSON(path string, data interface{}) {
	bytes, err := json.MarshalIndent(data, "", "  ")
	Expect(err).NotTo(HaveOccurred())
	err = os.WriteFile(path, bytes, 0644)
	Expect(err).NotTo(HaveOccurred())
}

var _ = Describe("SnapshotCapturesGitURLMarketplace", func() {
	var env *snapshotTestEnv

	BeforeEach(func() {
		env = setupSnapshotTestEnv()

		marketplaces := map[string]interface{}{
			"claude-code-plugins": map[string]interface{}{
				"source": map[string]interface{}{
					"source": "github",
					"repo":   "anthropics/claude-code",
				},
				"installLocation": "/path/to/claude-code-plugins",
			},
			"every-marketplace": map[string]interface{}{
				"source": map[string]interface{}{
					"source": "git",
					"url":    "https://github.com/EveryInc/compound-engineering-plugin.git",
				},
				"installLocation": "/path/to/every-marketplace",
			},
		}
		env.createMarketplaceRegistry(marketplaces)
		env.createPluginRegistry(map[string]interface{}{})
		env.createClaudeJSON(map[string]interface{}{"mcpServers": map[string]interface{}{}})
	})

	It("captures both github and git URL marketplaces", func() {
		p, err := profile.Snapshot("test-snapshot", env.claudeDir, env.claudeJSON, env.claudeDir)
		Expect(err).NotTo(HaveOccurred())

		Expect(p.Marketplaces).To(HaveLen(2))

		foundGithub := false
		foundGitURL := false
		for _, m := range p.Marketplaces {
			displayName := m.DisplayName()
			Expect(displayName).NotTo(BeEmpty(), "Empty display name for marketplace: source=%s", m.Source)

			if m.Source == "github" && m.Repo == "anthropics/claude-code" {
				foundGithub = true
				Expect(displayName).To(Equal("anthropics/claude-code"), "GitHub marketplace display name wrong")
			}

			if m.Source == "git" && m.URL == "https://github.com/EveryInc/compound-engineering-plugin.git" {
				foundGitURL = true
				Expect(displayName).To(Equal("https://github.com/EveryInc/compound-engineering-plugin.git"), "Git URL marketplace display name wrong")
			}
		}

		Expect(foundGithub).To(BeTrue(), "GitHub marketplace not found in snapshot")
		Expect(foundGitURL).To(BeTrue(), "Git URL marketplace not found in snapshot")
	})
})

var _ = Describe("SnapshotSaveLoadRoundTrip", func() {
	var env *snapshotTestEnv

	BeforeEach(func() {
		env = setupSnapshotTestEnv()

		marketplaces := map[string]interface{}{
			"git-marketplace": map[string]interface{}{
				"source": map[string]interface{}{
					"source": "git",
					"url":    "https://example.com/plugin.git",
				},
			},
		}
		env.createMarketplaceRegistry(marketplaces)
		env.createPluginRegistry(map[string]interface{}{})
		env.createClaudeJSON(map[string]interface{}{"mcpServers": map[string]interface{}{}})
	})

	It("preserves git URL marketplace through save and load", func() {
		p, err := profile.Snapshot("roundtrip-test", env.claudeDir, env.claudeJSON, env.claudeDir)
		Expect(err).NotTo(HaveOccurred())

		profilesDir := filepath.Join(env.claudeDir, "profiles")
		err = profile.Save(profilesDir, p)
		Expect(err).NotTo(HaveOccurred())

		loaded, err := profile.Load(profilesDir, "roundtrip-test")
		Expect(err).NotTo(HaveOccurred())

		Expect(loaded.Marketplaces).To(HaveLen(1))

		m := loaded.Marketplaces[0]
		Expect(m.Source).To(Equal("git"))
		Expect(m.URL).To(Equal("https://example.com/plugin.git"))
		Expect(m.DisplayName()).To(Equal("https://example.com/plugin.git"))
	})
})
