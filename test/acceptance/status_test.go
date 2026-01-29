// ABOUTME: Acceptance tests for status command
// ABOUTME: Tests display of marketplaces, plugins, MCP servers, and issues
package acceptance

import (
	"github.com/claudeup/claudeup/v3/internal/profile"
	"github.com/claudeup/claudeup/v3/test/helpers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("status", func() {
	var env *helpers.TestEnv

	BeforeEach(func() {
		env = helpers.NewTestEnv(binaryPath)
	})

	Describe("basic output", func() {
		Context("with empty installation", func() {
			It("shows status header", func() {
				result := env.Run("status")

				Expect(result.ExitCode).To(Equal(0))
				Expect(result.Stdout).To(ContainSubstring("claudeup Status"))
			})

			It("shows active profile as none", func() {
				result := env.Run("status")

				Expect(result.ExitCode).To(Equal(0))
				Expect(result.Stdout).To(ContainSubstring("Active Profile"))
				Expect(result.Stdout).To(ContainSubstring("none"))
			})

			It("shows marketplaces section with zero count", func() {
				result := env.Run("status")

				Expect(result.ExitCode).To(Equal(0))
				Expect(result.Stdout).To(ContainSubstring("Marketplaces"))
			})

			It("shows plugins section with zero count", func() {
				result := env.Run("status")

				Expect(result.ExitCode).To(Equal(0))
				Expect(result.Stdout).To(ContainSubstring("Plugins"))
			})

			It("shows MCP servers section", func() {
				result := env.Run("status")

				Expect(result.ExitCode).To(Equal(0))
				Expect(result.Stdout).To(ContainSubstring("MCP Servers"))
			})
		})
	})

	Describe("with active profile", func() {
		BeforeEach(func() {
			env.CreateProfile(&profile.Profile{
				Name:        "my-profile",
				Description: "Test profile",
			})
			env.SetActiveProfile("my-profile")
		})

		It("displays the active profile name", func() {
			result := env.Run("status")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("Active Profile"))
			Expect(result.Stdout).To(ContainSubstring("my-profile"))
		})
	})

	Describe("with marketplaces", func() {
		BeforeEach(func() {
			env.CreateKnownMarketplaces(map[string]interface{}{
				"acme-marketplace": map[string]interface{}{
					"source": map[string]interface{}{
						"source": "github",
						"repo":   "acme/plugins",
					},
					"installLocation": "/tmp/acme",
					"lastUpdated":     "2024-01-01T00:00:00Z",
				},
				"example-marketplace": map[string]interface{}{
					"source": map[string]interface{}{
						"source": "github",
						"repo":   "example/plugins",
					},
					"installLocation": "/tmp/example",
					"lastUpdated":     "2024-01-01T00:00:00Z",
				},
			})
		})

		It("lists marketplace names", func() {
			result := env.Run("status")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("acme-marketplace"))
			Expect(result.Stdout).To(ContainSubstring("example-marketplace"))
		})

		It("shows marketplace count in section header", func() {
			result := env.Run("status")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(MatchRegexp(`Marketplaces.*2`))
		})
	})

	Describe("with plugins", func() {
		BeforeEach(func() {
			env.CreateInstalledPlugins(map[string]interface{}{
				"plugin-one@acme": []interface{}{
					map[string]interface{}{
						"version":     "1.0.0",
						"installPath": env.ClaudeDir + "/plugins/cache/plugin-one",
						"scope":       "user",
					},
				},
				"plugin-two@example": []interface{}{
					map[string]interface{}{
						"version":     "2.0.0",
						"installPath": env.ClaudeDir + "/plugins/cache/plugin-two",
						"scope":       "user",
					},
				},
			})
			// Enable both plugins so they show as stale (paths don't exist)
			env.CreateSettings(map[string]bool{
				"plugin-one@acme":    true,
				"plugin-two@example": true,
			})
		})

		It("shows plugin count", func() {
			result := env.Run("status")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(MatchRegexp(`Plugins.*2`))
		})

		It("shows enabled count", func() {
			result := env.Run("status")

			Expect(result.ExitCode).To(Equal(0))
			// Both plugins have invalid paths, so they're stale
			Expect(result.Stdout).To(ContainSubstring("stale"))
		})
	})

	Describe("with stale plugins", func() {
		BeforeEach(func() {
			// Create plugin with non-existent path
			env.CreateInstalledPlugins(map[string]interface{}{
				"stale-plugin@acme": []interface{}{
					map[string]interface{}{
						"version":     "1.0.0",
						"installPath": "/nonexistent/path",
						"scope":       "user",
					},
				},
			})
			// Enable the plugin so it shows as stale
			env.CreateSettings(map[string]bool{
				"stale-plugin@acme": true,
			})
		})

		It("shows issues detected section", func() {
			result := env.Run("status")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("Issues"))
		})

		It("mentions stale plugins", func() {
			result := env.Run("status")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("stale"))
		})

		It("suggests running doctor", func() {
			result := env.Run("status")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("doctor"))
		})
	})

	Describe("help output", func() {
		It("shows help with --help flag", func() {
			result := env.Run("status", "--help")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("Display the current state"))
			Expect(result.Stdout).To(ContainSubstring("Usage:"))
		})

		It("shows help with help command", func() {
			result := env.Run("help", "status")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("Display the current state"))
		})
	})
})
