// ABOUTME: Acceptance tests for plugin search command
// ABOUTME: Tests CLI behavior with real binary execution
package acceptance

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/claudeup/claudeup/v4/test/helpers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("plugin search", func() {
	var env *helpers.TestEnv

	BeforeEach(func() {
		env = helpers.NewTestEnv(binaryPath)
	})

	AfterEach(func() {
		env.Cleanup()
	})

	Describe("without plugin cache", func() {
		It("shows error when cache does not exist", func() {
			result := env.Run("plugin", "search", "tdd")

			Expect(result.ExitCode).To(Equal(1))
			Expect(result.Stderr).To(ContainSubstring("plugin cache not found"))
		})
	})

	Describe("with plugin cache", func() {
		var cacheDir string

		BeforeEach(func() {
			// Create plugin cache structure
			cacheDir = filepath.Join(env.ClaudeDir, "plugins", "cache")
			pluginDir := filepath.Join(cacheDir, "test-marketplace", "tdd-plugin", "1.0.0")
			Expect(os.MkdirAll(filepath.Join(pluginDir, ".claude-plugin"), 0755)).To(Succeed())
			Expect(os.MkdirAll(filepath.Join(pluginDir, "skills", "tdd-skill"), 0755)).To(Succeed())
			Expect(os.MkdirAll(filepath.Join(pluginDir, "commands", "commit"), 0755)).To(Succeed())
			Expect(os.MkdirAll(filepath.Join(pluginDir, "agents", "code-reviewer"), 0755)).To(Succeed())

			// Write plugin.json
			pluginJSON := map[string]interface{}{
				"name":        "tdd-plugin",
				"description": "Test-driven development tools",
				"version":     "1.0.0",
				"keywords":    []string{"testing", "tdd", "unit-tests"},
			}
			pluginData, err := json.MarshalIndent(pluginJSON, "", "  ")
			Expect(err).NotTo(HaveOccurred())
			Expect(os.WriteFile(filepath.Join(pluginDir, ".claude-plugin", "plugin.json"), pluginData, 0644)).To(Succeed())

			// Write SKILL.md with frontmatter
			skillMD := `---
name: tdd-skill
description: Test-driven development workflow
---

# TDD Skill

Use this for TDD.
`
			Expect(os.WriteFile(filepath.Join(pluginDir, "skills", "tdd-skill", "SKILL.md"), []byte(skillMD), 0644)).To(Succeed())

			// Register as installed plugin
			env.CreateInstalledPlugins(map[string]interface{}{
				"tdd-plugin@test-marketplace": []interface{}{
					map[string]interface{}{
						"version":     "1.0.0",
						"installedAt": "2025-01-01T00:00:00Z",
						"scope":       "user",
					},
				},
			})
		})

		Describe("finds plugins matching query", func() {
			It("finds plugin by keyword", func() {
				result := env.Run("plugin", "search", "tdd")

				Expect(result.ExitCode).To(Equal(0))
				Expect(result.Stdout).To(ContainSubstring("tdd-plugin@test-marketplace"))
				Expect(result.Stdout).To(ContainSubstring("1.0.0"))
			})

			It("finds plugin by skill name", func() {
				result := env.Run("plugin", "search", "tdd-skill")

				Expect(result.ExitCode).To(Equal(0))
				Expect(result.Stdout).To(ContainSubstring("tdd-plugin@test-marketplace"))
				Expect(result.Stdout).To(ContainSubstring("Skills"))
			})

			It("shows match count in results header", func() {
				result := env.Run("plugin", "search", "tdd")

				Expect(result.ExitCode).To(Equal(0))
				// Header format: "Search results for "X" (N)" followed by "N plugins"
				Expect(result.Stdout).To(MatchRegexp(`Search results for .* \(\d+\)`))
				Expect(result.Stdout).To(MatchRegexp(`\d+ plugins?`))
			})
		})

		Describe("filters by component type", func() {
			It("shows only skills when --type skills", func() {
				result := env.Run("plugin", "search", "tdd", "--type", "skills")

				Expect(result.ExitCode).To(Equal(0))
				Expect(result.Stdout).To(ContainSubstring("Skills"))
				Expect(result.Stdout).NotTo(ContainSubstring("Commands"))
			})

			It("shows only commands when --type commands", func() {
				result := env.Run("plugin", "search", "commit", "--type", "commands")

				Expect(result.ExitCode).To(Equal(0))
				Expect(result.Stdout).To(ContainSubstring("tdd-plugin"))
			})

			It("shows only agents when --type agents", func() {
				result := env.Run("plugin", "search", "reviewer", "--type", "agents")

				Expect(result.ExitCode).To(Equal(0))
				Expect(result.Stdout).To(ContainSubstring("tdd-plugin"))
			})
		})

		Describe("supports --by-component flag", func() {
			It("groups output by component type", func() {
				result := env.Run("plugin", "search", "tdd", "--by-component")

				Expect(result.ExitCode).To(Equal(0))
				Expect(result.Stdout).To(ContainSubstring("Skills"))
				Expect(result.Stdout).To(ContainSubstring("tdd-skill"))
			})

			It("shows component count in header", func() {
				result := env.Run("plugin", "search", "tdd", "--by-component")

				Expect(result.ExitCode).To(Equal(0))
				// Header format: "Search results for "X" (N)" followed by "N plugins"
				Expect(result.Stdout).To(MatchRegexp(`Search results for .* \(\d+\)`))
				Expect(result.Stdout).To(MatchRegexp(`\d+ plugins?`))
			})
		})

		Describe("supports JSON output", func() {
			It("returns valid JSON with --format json", func() {
				result := env.Run("plugin", "search", "tdd", "--format", "json")

				Expect(result.ExitCode).To(Equal(0))

				var jsonOutput map[string]interface{}
				err := json.Unmarshal([]byte(result.Stdout), &jsonOutput)
				Expect(err).NotTo(HaveOccurred())

				Expect(jsonOutput).To(HaveKey("query"))
				Expect(jsonOutput).To(HaveKey("totalPlugins"))
				Expect(jsonOutput).To(HaveKey("totalMatches"))
				Expect(jsonOutput).To(HaveKey("results"))
			})

			It("includes query in JSON output", func() {
				result := env.Run("plugin", "search", "tdd", "--format", "json")

				Expect(result.ExitCode).To(Equal(0))

				var jsonOutput map[string]interface{}
				err := json.Unmarshal([]byte(result.Stdout), &jsonOutput)
				Expect(err).NotTo(HaveOccurred())

				Expect(jsonOutput["query"]).To(Equal("tdd"))
			})

			It("includes match details in JSON results", func() {
				result := env.Run("plugin", "search", "tdd", "--format", "json")

				Expect(result.ExitCode).To(Equal(0))

				var jsonOutput struct {
					Results []struct {
						Plugin      string `json:"plugin"`
						Marketplace string `json:"marketplace"`
						Version     string `json:"version"`
						Matches     []struct {
							Type string `json:"type"`
							Name string `json:"name"`
						} `json:"matches"`
					} `json:"results"`
				}
				err := json.Unmarshal([]byte(result.Stdout), &jsonOutput)
				Expect(err).NotTo(HaveOccurred())

				Expect(jsonOutput.Results).NotTo(BeEmpty())
				Expect(jsonOutput.Results[0].Plugin).To(Equal("tdd-plugin"))
				Expect(jsonOutput.Results[0].Marketplace).To(Equal("test-marketplace"))
			})
		})

		Describe("shows no results message", func() {
			It("shows helpful message when no matches found", func() {
				result := env.Run("plugin", "search", "nonexistent-xyz-12345")

				Expect(result.ExitCode).To(Equal(0))
				Expect(result.Stdout).To(ContainSubstring("No results"))
				Expect(result.Stdout).To(ContainSubstring("--all"))
			})

			It("suggests broadening search", func() {
				result := env.Run("plugin", "search", "nonexistent-xyz-12345")

				Expect(result.ExitCode).To(Equal(0))
				Expect(result.Stdout).To(ContainSubstring("Broaden your search"))
			})
		})

		Describe("validates --type flag values", func() {
			It("rejects invalid type value", func() {
				result := env.Run("plugin", "search", "tdd", "--type", "invalid")

				Expect(result.ExitCode).To(Equal(1))
				Expect(result.Stderr).To(ContainSubstring("invalid --type"))
				Expect(result.Stderr).To(ContainSubstring("skills"))
				Expect(result.Stderr).To(ContainSubstring("commands"))
				Expect(result.Stderr).To(ContainSubstring("agents"))
			})

			It("accepts skills type", func() {
				result := env.Run("plugin", "search", "tdd", "--type", "skills")

				Expect(result.ExitCode).To(Equal(0))
			})

			It("accepts commands type", func() {
				result := env.Run("plugin", "search", "tdd", "--type", "commands")

				// May have 0 results but should not error
				Expect(result.ExitCode).To(Equal(0))
			})

			It("accepts agents type", func() {
				result := env.Run("plugin", "search", "tdd", "--type", "agents")

				// May have 0 results but should not error
				Expect(result.ExitCode).To(Equal(0))
			})
		})

		Describe("deduplicates multiple cached versions", func() {
			BeforeEach(func() {
				// Add older versions of the same plugin to the cache
				for _, ver := range []string{"0.9.0", "0.5.0"} {
					vDir := filepath.Join(cacheDir, "test-marketplace", "tdd-plugin", ver)
					Expect(os.MkdirAll(filepath.Join(vDir, ".claude-plugin"), 0755)).To(Succeed())
					Expect(os.MkdirAll(filepath.Join(vDir, "skills", "tdd-skill"), 0755)).To(Succeed())

					pluginJSON := map[string]interface{}{
						"name":        "tdd-plugin",
						"description": "Test-driven development tools",
						"version":     ver,
						"keywords":    []string{"testing", "tdd"},
					}
					pluginData, err := json.MarshalIndent(pluginJSON, "", "  ")
					Expect(err).NotTo(HaveOccurred())
					Expect(os.WriteFile(filepath.Join(vDir, ".claude-plugin", "plugin.json"), pluginData, 0644)).To(Succeed())

					skillMD := `---
name: tdd-skill
description: Test-driven development workflow
---

# TDD Skill
`
					Expect(os.WriteFile(filepath.Join(vDir, "skills", "tdd-skill", "SKILL.md"), []byte(skillMD), 0644)).To(Succeed())
				}
			})

			It("returns only one entry per plugin", func() {
				result := env.Run("plugin", "search", "tdd")

				Expect(result.ExitCode).To(Equal(0))
				// Name appears twice: once in search results, once in tree header.
				// Without dedup, 3 versions would produce 6 occurrences.
				Expect(strings.Count(result.Stdout, "tdd-plugin@test-marketplace")).To(Equal(2))
			})

			It("keeps the latest version", func() {
				result := env.Run("plugin", "search", "tdd", "--format", "json")

				Expect(result.ExitCode).To(Equal(0))

				var jsonOutput struct {
					Results []struct {
						Plugin  string `json:"plugin"`
						Version string `json:"version"`
					} `json:"results"`
				}
				err := json.Unmarshal([]byte(result.Stdout), &jsonOutput)
				Expect(err).NotTo(HaveOccurred())

				Expect(jsonOutput.Results).To(HaveLen(1))
				Expect(jsonOutput.Results[0].Version).To(Equal("1.0.0"))
			})
		})

		Describe("--all flag behavior", func() {
			BeforeEach(func() {
				// Create another plugin that is NOT installed
				uninstalledDir := filepath.Join(cacheDir, "other-marketplace", "uninstalled-plugin", "2.0.0")
				Expect(os.MkdirAll(filepath.Join(uninstalledDir, ".claude-plugin"), 0755)).To(Succeed())
				Expect(os.MkdirAll(filepath.Join(uninstalledDir, "skills", "test-automation"), 0755)).To(Succeed())

				pluginJSON := map[string]interface{}{
					"name":        "uninstalled-plugin",
					"description": "A plugin for testing automation",
					"version":     "2.0.0",
					"keywords":    []string{"testing", "automation"},
				}
				pluginData, err := json.MarshalIndent(pluginJSON, "", "  ")
				Expect(err).NotTo(HaveOccurred())
				Expect(os.WriteFile(filepath.Join(uninstalledDir, ".claude-plugin", "plugin.json"), pluginData, 0644)).To(Succeed())

				skillMD := `---
name: test-automation
description: Automated testing workflow
---

# Test Automation
`
				Expect(os.WriteFile(filepath.Join(uninstalledDir, "skills", "test-automation", "SKILL.md"), []byte(skillMD), 0644)).To(Succeed())
			})

			It("searches only installed plugins by default", func() {
				result := env.Run("plugin", "search", "automation")

				// Should not find the uninstalled plugin
				Expect(result.ExitCode).To(Equal(0))
				Expect(result.Stdout).NotTo(ContainSubstring("uninstalled-plugin"))
			})

			It("searches all cached plugins with --all", func() {
				result := env.Run("plugin", "search", "automation", "--all")

				Expect(result.ExitCode).To(Equal(0))
				Expect(result.Stdout).To(ContainSubstring("uninstalled-plugin"))
				Expect(result.Stdout).To(ContainSubstring("other-marketplace"))
			})
		})

		Describe("--marketplace flag", func() {
			BeforeEach(func() {
				// Create another plugin in a different marketplace
				otherDir := filepath.Join(cacheDir, "another-marketplace", "another-plugin", "1.0.0")
				Expect(os.MkdirAll(filepath.Join(otherDir, ".claude-plugin"), 0755)).To(Succeed())
				Expect(os.MkdirAll(filepath.Join(otherDir, "skills", "tdd-helper"), 0755)).To(Succeed())

				pluginJSON := map[string]interface{}{
					"name":        "another-plugin",
					"description": "Another TDD plugin",
					"version":     "1.0.0",
					"keywords":    []string{"tdd"},
				}
				pluginData, err := json.MarshalIndent(pluginJSON, "", "  ")
				Expect(err).NotTo(HaveOccurred())
				Expect(os.WriteFile(filepath.Join(otherDir, ".claude-plugin", "plugin.json"), pluginData, 0644)).To(Succeed())

				skillMD := `---
name: tdd-helper
description: TDD helper skill
---

# TDD Helper
`
				Expect(os.WriteFile(filepath.Join(otherDir, "skills", "tdd-helper", "SKILL.md"), []byte(skillMD), 0644)).To(Succeed())

				// Register both plugins as installed
				env.CreateInstalledPlugins(map[string]interface{}{
					"tdd-plugin@test-marketplace": []interface{}{
						map[string]interface{}{
							"version":     "1.0.0",
							"installedAt": "2025-01-01T00:00:00Z",
							"scope":       "user",
						},
					},
					"another-plugin@another-marketplace": []interface{}{
						map[string]interface{}{
							"version":     "1.0.0",
							"installedAt": "2025-01-01T00:00:00Z",
							"scope":       "user",
						},
					},
				})
			})

			It("limits search to specific marketplace", func() {
				result := env.Run("plugin", "search", "tdd", "--marketplace", "test-marketplace")

				Expect(result.ExitCode).To(Equal(0))
				Expect(result.Stdout).To(ContainSubstring("tdd-plugin@test-marketplace"))
				Expect(result.Stdout).NotTo(ContainSubstring("another-marketplace"))
			})
		})
	})

	Describe("requires query argument", func() {
		BeforeEach(func() {
			// Create minimal cache so we don't fail on "cache not found"
			cacheDir := filepath.Join(env.ClaudeDir, "plugins", "cache")
			Expect(os.MkdirAll(cacheDir, 0755)).To(Succeed())
		})

		It("fails when no query provided", func() {
			result := env.Run("plugin", "search")

			Expect(result.ExitCode).To(Equal(1))
			Expect(result.Stderr).To(ContainSubstring("accepts 1 arg"))
		})
	})
})
