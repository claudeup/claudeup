// ABOUTME: Acceptance tests for plugins command
// ABOUTME: Tests plugin listing, summary flag, and various plugin states
package acceptance

import (
	"os"
	"path/filepath"

	"github.com/claudeup/claudeup/test/helpers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("plugins", func() {
	var env *helpers.TestEnv

	BeforeEach(func() {
		env = helpers.NewTestEnv(binaryPath)
	})

	Describe("with no plugins", func() {
		It("shows empty plugin list", func() {
			result := env.Run("plugins")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("Installed Plugins"))
		})

		It("shows summary with zero count", func() {
			result := env.Run("plugins", "--summary")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("Plugin Summary"))
			Expect(result.Stdout).To(ContainSubstring("0 plugins"))
		})
	})

	Describe("with installed plugins", func() {
		var pluginPath string

		BeforeEach(func() {
			// Create a valid plugin directory
			pluginPath = filepath.Join(env.ClaudeDir, "plugins", "cache", "my-plugin")
			Expect(os.MkdirAll(pluginPath, 0755)).To(Succeed())

			env.CreateInstalledPlugins(map[string]interface{}{
				"my-plugin@acme-marketplace": []interface{}{
					map[string]interface{}{
						"version":      "1.2.3",
						"installPath":  pluginPath,
						"gitCommitSha": "abc123",
						"isLocal":      false,
						"installedAt":  "2024-01-15T10:30:00Z",
						"scope":        "user",
					},
				},
			})

			// Enable the plugin in settings
			env.CreateSettings(map[string]bool{
				"my-plugin@acme-marketplace": true,
			})
		})

		It("lists plugin name", func() {
			result := env.Run("plugins")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("my-plugin@acme-marketplace"))
		})

		It("shows plugin version", func() {
			result := env.Run("plugins")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("1.2.3"))
		})

		It("shows plugin status as enabled", func() {
			result := env.Run("plugins")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("enabled"))
		})

		It("shows plugin path", func() {
			result := env.Run("plugins")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring(pluginPath))
		})

		It("shows plugin type as cached", func() {
			result := env.Run("plugins")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("cached"))
		})
	})

	Describe("with local plugins", func() {
		var pluginPath string

		BeforeEach(func() {
			// Create a valid local plugin directory
			pluginPath = filepath.Join(env.ClaudeDir, "plugins", "marketplaces", "local-mp", "plugins", "local-plugin")
			Expect(os.MkdirAll(pluginPath, 0755)).To(Succeed())

			env.CreateInstalledPlugins(map[string]interface{}{
				"local-plugin@local-mp": []interface{}{
					map[string]interface{}{
						"version":     "0.1.0",
						"installPath": pluginPath,
						"isLocal":     true,
						"installedAt": "2024-01-15T10:30:00Z",
						"scope":       "user",
					},
				},
			})

			// Enable the plugin
			env.CreateSettings(map[string]bool{
				"local-plugin@local-mp": true,
			})
		})

		It("shows plugin type as local", func() {
			result := env.Run("plugins")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("local"))
		})
	})

	Describe("with stale plugins", func() {
		BeforeEach(func() {
			env.CreateInstalledPlugins(map[string]interface{}{
				"stale-plugin@acme": []interface{}{
					map[string]interface{}{
						"version":     "1.0.0",
						"installPath": "/nonexistent/path/to/plugin",
						"scope":       "user",
					},
				},
			})
		})

		It("shows plugin status as stale", func() {
			result := env.Run("plugins")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("stale"))
		})

		It("indicates path not found", func() {
			result := env.Run("plugins")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("path not found"))
		})
	})

	Describe("--summary flag", func() {
		var pluginPath1, pluginPath2 string

		BeforeEach(func() {
			// Create two valid plugin directories
			pluginPath1 = filepath.Join(env.ClaudeDir, "plugins", "cache", "plugin1")
			pluginPath2 = filepath.Join(env.ClaudeDir, "plugins", "marketplaces", "mp", "plugins", "plugin2")
			Expect(os.MkdirAll(pluginPath1, 0755)).To(Succeed())
			Expect(os.MkdirAll(pluginPath2, 0755)).To(Succeed())

			env.CreateInstalledPlugins(map[string]interface{}{
				"plugin1@acme": []interface{}{
					map[string]interface{}{
						"version":     "1.0.0",
						"installPath": pluginPath1,
						"isLocal":     false,
						"scope":       "user",
					},
				},
				"plugin2@example": []interface{}{
					map[string]interface{}{
						"version":     "2.0.0",
						"installPath": pluginPath2,
						"isLocal":     true,
						"scope":       "user",
					},
				},
			})

			// Enable both plugins
			env.CreateSettings(map[string]bool{
				"plugin1@acme":    true,
				"plugin2@example": true,
			})
		})

		It("shows summary header", func() {
			result := env.Run("plugins", "--summary")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("Plugin Summary"))
		})

		It("shows total count", func() {
			result := env.Run("plugins", "--summary")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("2 plugins"))
		})

		It("shows enabled count", func() {
			result := env.Run("plugins", "--summary")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("Enabled"))
			Expect(result.Stdout).To(ContainSubstring("2"))
		})

		It("shows by-type breakdown", func() {
			result := env.Run("plugins", "--summary")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("By Type:"))
			Expect(result.Stdout).To(ContainSubstring("Cached:"))
			Expect(result.Stdout).To(ContainSubstring("Local:"))
		})

		It("shows cached count", func() {
			result := env.Run("plugins", "--summary")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(MatchRegexp(`Cached:\s*1`))
		})

		It("shows local count", func() {
			result := env.Run("plugins", "--summary")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(MatchRegexp(`Local:\s*1`))
		})

		It("does not show individual plugin details", func() {
			result := env.Run("plugins", "--summary")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).NotTo(ContainSubstring("plugin1@acme"))
			Expect(result.Stdout).NotTo(ContainSubstring("plugin2@example"))
		})
	})

	Describe("with mixed plugin states", func() {
		var validPath string

		BeforeEach(func() {
			validPath = filepath.Join(env.ClaudeDir, "plugins", "cache", "valid-plugin")
			Expect(os.MkdirAll(validPath, 0755)).To(Succeed())

			env.CreateInstalledPlugins(map[string]interface{}{
				"valid-plugin@acme": []interface{}{
					map[string]interface{}{
						"version":     "1.0.0",
						"installPath": validPath,
						"scope":       "user",
					},
				},
				"stale-plugin@acme": []interface{}{
					map[string]interface{}{
						"version":     "1.0.0",
						"installPath": "/nonexistent",
						"scope":       "user",
					},
				},
			})

			// Enable the valid plugin in settings
			env.CreateSettings(map[string]bool{
				"valid-plugin@acme": true,
				"stale-plugin@acme": true,
			})
		})

		It("shows both enabled and stale in summary", func() {
			result := env.Run("plugins")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("enabled"))
			Expect(result.Stdout).To(ContainSubstring("stale"))
		})

		It("shows stale count in summary footer", func() {
			result := env.Run("plugins")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("1 stale"))
		})
	})

	Describe("help output", func() {
		It("shows help with --help flag", func() {
			result := env.Run("plugins", "--help")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("alias"))
			Expect(result.Stdout).To(ContainSubstring("--summary"))
			Expect(result.Stdout).To(ContainSubstring("Usage:"))
		})

		It("shows help with help command", func() {
			result := env.Run("help", "plugins")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("plugin list"))
		})

		It("describes --summary flag in help", func() {
			result := env.Run("plugins", "--help")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("summary"))
		})
	})

	Describe("multiple plugins sorting", func() {
		BeforeEach(func() {
			// Create paths
			pathA := filepath.Join(env.ClaudeDir, "plugins", "cache", "aaa-plugin")
			pathZ := filepath.Join(env.ClaudeDir, "plugins", "cache", "zzz-plugin")
			pathM := filepath.Join(env.ClaudeDir, "plugins", "cache", "mmm-plugin")
			Expect(os.MkdirAll(pathA, 0755)).To(Succeed())
			Expect(os.MkdirAll(pathZ, 0755)).To(Succeed())
			Expect(os.MkdirAll(pathM, 0755)).To(Succeed())

			env.CreateInstalledPlugins(map[string]interface{}{
				"zzz-plugin@acme": []interface{}{
					map[string]interface{}{
						"version":     "1.0.0",
						"installPath": pathZ,
						"scope":       "user",
					},
				},
				"aaa-plugin@acme": []interface{}{
					map[string]interface{}{
						"version":     "1.0.0",
						"installPath": pathA,
						"scope":       "user",
					},
				},
				"mmm-plugin@acme": []interface{}{
					map[string]interface{}{
						"version":     "1.0.0",
						"installPath": pathM,
						"scope":       "user",
					},
				},
			})

			// Enable all plugins
			env.CreateSettings(map[string]bool{
				"aaa-plugin@acme": true,
				"mmm-plugin@acme": true,
				"zzz-plugin@acme": true,
			})
		})

		It("lists plugins in alphabetical order", func() {
			result := env.Run("plugins")

			Expect(result.ExitCode).To(Equal(0))

			// Find indices of plugin names
			lines := splitLines(result.Stdout)
			aaaIdx := findLineContaining(lines, "aaa-plugin")
			mmmIdx := findLineContaining(lines, "mmm-plugin")
			zzzIdx := findLineContaining(lines, "zzz-plugin")

			Expect(aaaIdx).To(BeNumerically("<", mmmIdx))
			Expect(mmmIdx).To(BeNumerically("<", zzzIdx))
		})
	})

	Describe("--enabled flag", func() {
		var enabledPath, disabledPath string

		BeforeEach(func() {
			enabledPath = filepath.Join(env.ClaudeDir, "plugins", "cache", "enabled-plugin")
			disabledPath = filepath.Join(env.ClaudeDir, "plugins", "cache", "disabled-plugin")
			Expect(os.MkdirAll(enabledPath, 0755)).To(Succeed())
			Expect(os.MkdirAll(disabledPath, 0755)).To(Succeed())

			env.CreateInstalledPlugins(map[string]interface{}{
				"enabled-plugin@acme": []interface{}{
					map[string]interface{}{
						"version":     "1.0.0",
						"installPath": enabledPath,
						"scope":       "user",
					},
				},
				"disabled-plugin@acme": []interface{}{
					map[string]interface{}{
						"version":     "1.0.0",
						"installPath": disabledPath,
						"scope":       "user",
					},
				},
			})

			// Only enable one plugin
			env.CreateSettings(map[string]bool{
				"enabled-plugin@acme": true,
			})
		})

		It("shows only enabled plugins", func() {
			result := env.Run("plugin", "list", "--enabled")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("enabled-plugin@acme"))
			Expect(result.Stdout).NotTo(ContainSubstring("disabled-plugin@acme"))
		})

		It("shows filtered count in footer", func() {
			result := env.Run("plugin", "list", "--enabled")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("Showing: 1 enabled"))
			Expect(result.Stdout).To(ContainSubstring("of 2 total"))
		})
	})

	Describe("--disabled flag", func() {
		var enabledPath, disabledPath string

		BeforeEach(func() {
			enabledPath = filepath.Join(env.ClaudeDir, "plugins", "cache", "enabled-plugin")
			disabledPath = filepath.Join(env.ClaudeDir, "plugins", "cache", "disabled-plugin")
			Expect(os.MkdirAll(enabledPath, 0755)).To(Succeed())
			Expect(os.MkdirAll(disabledPath, 0755)).To(Succeed())

			env.CreateInstalledPlugins(map[string]interface{}{
				"enabled-plugin@acme": []interface{}{
					map[string]interface{}{
						"version":     "1.0.0",
						"installPath": enabledPath,
						"scope":       "user",
					},
				},
				"disabled-plugin@acme": []interface{}{
					map[string]interface{}{
						"version":     "1.0.0",
						"installPath": disabledPath,
						"scope":       "user",
					},
				},
			})

			// Only enable one plugin
			env.CreateSettings(map[string]bool{
				"enabled-plugin@acme": true,
			})
		})

		It("shows only disabled plugins", func() {
			result := env.Run("plugin", "list", "--disabled")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("disabled-plugin@acme"))
			Expect(result.Stdout).NotTo(ContainSubstring("enabled-plugin@acme"))
		})

		It("shows filtered count in footer", func() {
			result := env.Run("plugin", "list", "--disabled")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("Showing: 1 disabled"))
			Expect(result.Stdout).To(ContainSubstring("of 2 total"))
		})
	})

	Describe("mutually exclusive flags", func() {
		It("returns error when both --enabled and --disabled are used", func() {
			result := env.Run("plugin", "list", "--enabled", "--disabled")

			Expect(result.ExitCode).To(Equal(1))
			Expect(result.Stderr).To(ContainSubstring("--enabled and --disabled are mutually exclusive"))
		})
	})
})
