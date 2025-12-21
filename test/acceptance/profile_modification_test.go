// ABOUTME: Acceptance tests for profile modification detection
// ABOUTME: Tests status warnings, force flag, and modified indicators
package acceptance

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/claudeup/claudeup/internal/profile"
	"github.com/claudeup/claudeup/test/helpers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("profile modification detection", func() {
	var env *helpers.TestEnv

	BeforeEach(func() {
		env = helpers.NewTestEnv(binaryPath)
	})

	Describe("status command", func() {
		Context("when active profile has unsaved changes", func() {
			BeforeEach(func() {
				// Create and activate a profile
				env.CreateProfile(&profile.Profile{
					Name:        "test-profile",
					Description: "Test profile",
					Plugins:     []string{"plugin-a"},
					Marketplaces: []profile.Marketplace{
						{Repo: "acme/marketplace"},
					},
					MCPServers: []profile.MCPServer{},
				})
				env.SetActiveProfile("test-profile")

				// Create .claude.json to simulate current state
				claudeJSON := map[string]interface{}{
					"mcpServers": map[string]interface{}{},
				}
				data, _ := json.MarshalIndent(claudeJSON, "", "  ")
				os.WriteFile(filepath.Join(env.TempDir, ".claude.json"), data, 0644)

				// Create plugins directory and files for current state
				env.CreateInstalledPlugins(map[string]interface{}{
					"plugin-a": []interface{}{
						map[string]interface{}{
							"version":     "1.0.0",
							"installPath": "/fake/path",
							"scope":       "user",
						},
					},
					"plugin-b": []interface{}{
						map[string]interface{}{
							"version":     "1.0.0",
							"installPath": "/fake/path",
							"scope":       "user",
						},
					},
				})

				// Create marketplaces for current state (different from profile)
				env.CreateKnownMarketplaces(map[string]interface{}{
					"acme-marketplace": map[string]interface{}{
						"source": map[string]interface{}{
							"source": "github",
							"repo":   "acme/marketplace",
						},
						"installLocation": "/tmp/acme",
						"lastUpdated":     "2024-01-01T00:00:00Z",
					},
					"example-marketplace": map[string]interface{}{
						"source": map[string]interface{}{
							"source": "github",
							"repo":   "example/marketplace",
						},
						"installLocation": "/tmp/example",
						"lastUpdated":     "2024-01-01T00:00:00Z",
					},
				})
			})

			It("shows warning about unsaved changes", func() {
				result := env.Run("status")

				Expect(result.ExitCode).To(Equal(0))
				Expect(result.Stdout).To(ContainSubstring("System differs from profile"))
			})

			It("shows summary of changes", func() {
				result := env.Run("status")

				Expect(result.ExitCode).To(Equal(0))
				// Should mention plugins and marketplace changes
				Expect(result.Stdout).To(MatchRegexp("plugin.*not in profile|marketplace.*not in profile"))
			})

			It("suggests running profile save", func() {
				result := env.Run("status")

				Expect(result.ExitCode).To(Equal(0))
				Expect(result.Stdout).To(ContainSubstring("profile save"))
			})
		})

		Context("when active profile has no changes", func() {
			BeforeEach(func() {
				// Create profile matching current state exactly
				env.CreateProfile(&profile.Profile{
					Name:         "test-profile",
					Description:  "Test profile",
					Plugins:      []string{},
					Marketplaces: []profile.Marketplace{},
					MCPServers:   []profile.MCPServer{},
				})
				env.SetActiveProfile("test-profile")

				// Create .claude.json with matching state
				claudeJSON := map[string]interface{}{
					"mcpServers": map[string]interface{}{},
				}
				data, _ := json.MarshalIndent(claudeJSON, "", "  ")
				os.WriteFile(filepath.Join(env.TempDir, ".claude.json"), data, 0644)
			})

			It("does not show modification warning", func() {
				result := env.Run("status")

				Expect(result.ExitCode).To(Equal(0))
				Expect(result.Stdout).NotTo(ContainSubstring("has unsaved changes"))
				Expect(result.Stdout).NotTo(ContainSubstring("profile save"))
			})
		})
	})

	Describe("profile use command", func() {
		Context("when reapplying active profile with unsaved changes", func() {
			BeforeEach(func() {
				// Create and activate a profile
				env.CreateProfile(&profile.Profile{
					Name:         "test-profile",
					Description:  "Test profile",
					Plugins:      []string{"plugin-a"},
					Marketplaces: []profile.Marketplace{},
					MCPServers:   []profile.MCPServer{},
				})
				env.SetActiveProfile("test-profile")

				// Create current state with extra plugin (modification)
				env.CreateInstalledPlugins(map[string]interface{}{
					"plugin-a": []interface{}{
						map[string]interface{}{
							"version":     "1.0.0",
							"installPath": "/fake/path",
							"scope":       "user",
						},
					},
					"plugin-b": []interface{}{
						map[string]interface{}{
							"version":     "1.0.0",
							"installPath": "/fake/path",
							"scope":       "user",
						},
					},
				})

				// Enable plugin-b in settings (the "unsaved change")
				env.CreateSettings(map[string]bool{
					"plugin-b": true,
				})

				// Create .claude.json
				claudeJSON := map[string]interface{}{
					"mcpServers": map[string]interface{}{},
				}
				data, _ := json.MarshalIndent(claudeJSON, "", "  ")
				os.WriteFile(filepath.Join(env.TempDir, ".claude.json"), data, 0644)
			})

			It("allows reapplication (declarative - syncs state)", func() {
				result := env.Run("profile", "use", "test-profile", "-y")

				Expect(result.ExitCode).To(Equal(0))
				Expect(result.Stdout).To(ContainSubstring("Profile applied"))
			})

			It("removes the extra plugin to match profile", func() {
				result := env.Run("profile", "use", "test-profile", "-y")

				Expect(result.ExitCode).To(Equal(0))
				// Should have removed plugin-b (not in profile)
				Expect(result.Stdout).To(MatchRegexp("Removed.*plugin"))
			})
		})

		Context("when using --force flag with unsaved changes", func() {
			BeforeEach(func() {
				// Create and activate a profile
				env.CreateProfile(&profile.Profile{
					Name:         "test-profile",
					Description:  "Test profile",
					Plugins:      []string{},
					Marketplaces: []profile.Marketplace{},
					MCPServers:   []profile.MCPServer{},
				})
				env.SetActiveProfile("test-profile")

				// Create current state with modifications
				env.CreateInstalledPlugins(map[string]interface{}{
					"extra-plugin": []interface{}{
						map[string]interface{}{
							"version":     "1.0.0",
							"installPath": "/fake/path",
							"scope":       "user",
						},
					},
				})

				// Create .claude.json
				claudeJSON := map[string]interface{}{
					"mcpServers": map[string]interface{}{},
				}
				data, _ := json.MarshalIndent(claudeJSON, "", "  ")
				os.WriteFile(filepath.Join(env.TempDir, ".claude.json"), data, 0644)
			})

			It("allows reapplication", func() {
				result := env.Run("profile", "use", "test-profile", "--force")

				Expect(result.ExitCode).To(Equal(0))
				Expect(result.Stdout).To(ContainSubstring("Profile applied"))
			})
		})

		Context("when reapplying active profile with no changes", func() {
			BeforeEach(func() {
				// Create profile matching current state
				env.CreateProfile(&profile.Profile{
					Name:         "test-profile",
					Description:  "Test profile",
					Plugins:      []string{},
					Marketplaces: []profile.Marketplace{},
					MCPServers:   []profile.MCPServer{},
				})
				env.SetActiveProfile("test-profile")

				// Create .claude.json
				claudeJSON := map[string]interface{}{
					"mcpServers": map[string]interface{}{},
				}
				data, _ := json.MarshalIndent(claudeJSON, "", "  ")
				os.WriteFile(filepath.Join(env.TempDir, ".claude.json"), data, 0644)
			})

			It("allows reapplication without force flag", func() {
				result := env.Run("profile", "use", "test-profile")

				Expect(result.ExitCode).To(Equal(0))
				Expect(result.Stdout).To(ContainSubstring("No changes needed"))
			})
		})
	})

	Describe("profile list command", func() {
		Context("when active profile has unsaved changes", func() {
			BeforeEach(func() {
				// Create and activate a profile
				env.CreateProfile(&profile.Profile{
					Name:         "test-profile",
					Description:  "Test profile",
					Plugins:      []string{},
					Marketplaces: []profile.Marketplace{},
					MCPServers:   []profile.MCPServer{},
				})
				env.SetActiveProfile("test-profile")

				// Create current state with modifications
				env.CreateInstalledPlugins(map[string]interface{}{
					"new-plugin": []interface{}{
						map[string]interface{}{
							"version":     "1.0.0",
							"installPath": "/fake/path",
							"scope":       "user",
						},
					},
				})

				// Create .claude.json
				claudeJSON := map[string]interface{}{
					"mcpServers": map[string]interface{}{},
				}
				data, _ := json.MarshalIndent(claudeJSON, "", "  ")
				os.WriteFile(filepath.Join(env.TempDir, ".claude.json"), data, 0644)
			})

			It("shows (modified) indicator for active profile", func() {
				result := env.Run("profile", "list")

				Expect(result.ExitCode).To(Equal(0))
				Expect(result.Stdout).To(ContainSubstring("test-profile"))
				Expect(result.Stdout).To(ContainSubstring("(modified)"))
			})
		})

		Context("when active profile has no changes", func() {
			BeforeEach(func() {
				// Create profile matching current state
				env.CreateProfile(&profile.Profile{
					Name:         "test-profile",
					Description:  "Test profile",
					Plugins:      []string{},
					Marketplaces: []profile.Marketplace{},
					MCPServers:   []profile.MCPServer{},
				})
				env.SetActiveProfile("test-profile")

				// Create .claude.json
				claudeJSON := map[string]interface{}{
					"mcpServers": map[string]interface{}{},
				}
				data, _ := json.MarshalIndent(claudeJSON, "", "  ")
				os.WriteFile(filepath.Join(env.TempDir, ".claude.json"), data, 0644)
			})

			It("does not show (modified) indicator", func() {
				result := env.Run("profile", "list")

				Expect(result.ExitCode).To(Equal(0))
				Expect(result.Stdout).To(ContainSubstring("test-profile"))
				Expect(result.Stdout).NotTo(ContainSubstring("(modified)"))
			})
		})
	})
})
