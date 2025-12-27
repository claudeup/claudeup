// ABOUTME: Acceptance tests for scope-aware config drift cleanup with profile clean command
// ABOUTME: Tests drift detection and cleanup for plugins in .claudeup.json and .claudeup.local.json
package acceptance

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/claudeup/claudeup/test/helpers"
)

var _ = Describe("Profile clean command for config drift", func() {
	var (
		env        *helpers.TestEnv
		binaryPath string
		projectDir string
	)

	BeforeEach(func() {
		binaryPath = helpers.BuildBinary()
		env = helpers.NewTestEnv(binaryPath)

		projectDir = env.ProjectDir("test-project")
		err := os.MkdirAll(filepath.Join(projectDir, ".claude"), 0755)
		Expect(err).NotTo(HaveOccurred())

		// Create minimal installed_plugins.json
		installedPlugins := map[string]interface{}{
			"version": 2,
			"plugins": map[string]interface{}{},
		}
		helpers.WriteJSON(filepath.Join(env.ClaudeDir, "plugins", "installed_plugins.json"), installedPlugins)
	})

	AfterEach(func() {
		env.Cleanup()
	})

	Describe("Detecting config drift", func() {
		Context("when plugins in .claudeup.json are not installed", func() {
			BeforeEach(func() {
				// Create .claudeup.json with plugins
				projectConfig := map[string]interface{}{
					"version": "1",
					"profile": "test-profile",
					"plugins": []string{
						"missing-plugin-a@marketplace",
						"missing-plugin-b@marketplace",
					},
				}
				helpers.WriteJSON(filepath.Join(projectDir, ".claudeup.json"), projectConfig)

				// Create minimal settings
				settings := map[string]interface{}{
					"enabledPlugins": map[string]bool{},
				}
				helpers.WriteJSON(filepath.Join(env.ClaudeDir, "settings.json"), settings)
			})

			It("status should show config drift from project scope", func() {
				result := env.RunInDir(projectDir, "status")

				Expect(result.ExitCode).To(Equal(0))
				Expect(result.Stdout).To(ContainSubstring("Configuration Drift Detected"))
				Expect(result.Stdout).To(ContainSubstring("orphaned config entri"))
				Expect(result.Stdout).To(ContainSubstring("missing-plugin-a@marketplace"))
				Expect(result.Stdout).To(ContainSubstring("missing-plugin-b@marketplace"))
				Expect(result.Stdout).To(ContainSubstring("(project scope)"))
				Expect(result.Stdout).To(ContainSubstring("claudeup profile clean --scope"))
			})

			It("doctor should show config drift from project scope", func() {
				result := env.RunInDir(projectDir, "doctor")

				Expect(result.ExitCode).To(Equal(0))
				Expect(result.Stdout).To(ContainSubstring("orphaned config entri"))
				Expect(result.Stdout).To(ContainSubstring("missing-plugin-a@marketplace"))
				Expect(result.Stdout).To(ContainSubstring("missing-plugin-b@marketplace"))
				Expect(result.Stdout).To(ContainSubstring("(project scope)"))
				Expect(result.Stdout).To(ContainSubstring("claudeup profile clean"))
			})
		})

		Context("when plugins in .claudeup.local.json are not installed", func() {
			BeforeEach(func() {
				// Create .claudeup.local.json with plugins
				localConfig := map[string]interface{}{
					"version": "1",
					"profile": "local-profile",
					"plugins": []string{
						"local-missing-plugin@marketplace",
					},
				}
				helpers.WriteJSON(filepath.Join(projectDir, ".claudeup.local.json"), localConfig)

				// Create minimal settings
				settings := map[string]interface{}{
					"enabledPlugins": map[string]bool{},
				}
				helpers.WriteJSON(filepath.Join(env.ClaudeDir, "settings.json"), settings)
			})

			It("status should show config drift from local scope", func() {
				result := env.RunInDir(projectDir, "status")

				Expect(result.ExitCode).To(Equal(0))
				Expect(result.Stdout).To(ContainSubstring("Configuration Drift Detected"))
				Expect(result.Stdout).To(ContainSubstring("orphaned config entry"))
				Expect(result.Stdout).To(ContainSubstring("local-missing-plugin@marketplace"))
				Expect(result.Stdout).To(ContainSubstring("(local scope)"))
			})

			It("doctor should show config drift from local scope", func() {
				result := env.RunInDir(projectDir, "doctor")

				Expect(result.ExitCode).To(Equal(0))
				Expect(result.Stdout).To(ContainSubstring("orphaned config entry"))
				Expect(result.Stdout).To(ContainSubstring("local-missing-plugin@marketplace"))
				Expect(result.Stdout).To(ContainSubstring("(local scope)"))
			})
		})

		Context("when drift exists in both scopes", func() {
			BeforeEach(func() {
				// Create project config
				projectConfig := map[string]interface{}{
					"version": "1",
					"profile": "test-profile",
					"plugins": []string{
						"project-missing@marketplace",
					},
				}
				helpers.WriteJSON(filepath.Join(projectDir, ".claudeup.json"), projectConfig)

				// Create local config
				localConfig := map[string]interface{}{
					"version": "1",
					"profile": "local-profile",
					"plugins": []string{
						"local-missing@marketplace",
					},
				}
				helpers.WriteJSON(filepath.Join(projectDir, ".claudeup.local.json"), localConfig)

				// Create minimal settings
				settings := map[string]interface{}{
					"enabledPlugins": map[string]bool{},
				}
				helpers.WriteJSON(filepath.Join(env.ClaudeDir, "settings.json"), settings)
			})

			It("status should show drift from both scopes", func() {
				result := env.RunInDir(projectDir, "status")

				Expect(result.ExitCode).To(Equal(0))
				Expect(result.Stdout).To(ContainSubstring("Configuration Drift Detected"))
				Expect(result.Stdout).To(ContainSubstring("project-missing@marketplace"))
				Expect(result.Stdout).To(ContainSubstring("(project scope)"))
				Expect(result.Stdout).To(ContainSubstring("local-missing@marketplace"))
				Expect(result.Stdout).To(ContainSubstring("(local scope)"))
			})
		})
	})

	Describe("Cleaning config drift with profile clean", func() {
		Context("removing plugin from project scope", func() {
			BeforeEach(func() {
				// Create .claudeup.json with plugins
				projectConfig := map[string]interface{}{
					"version": "1",
					"profile": "test-profile",
					"plugins": []string{
						"plugin-a@marketplace",
						"plugin-b@marketplace",
						"plugin-c@marketplace",
					},
				}
				helpers.WriteJSON(filepath.Join(projectDir, ".claudeup.json"), projectConfig)
			})

			It("should remove plugin from .claudeup.json", func() {
				result := env.RunInDir(projectDir, "profile", "clean", "--scope", "project", "plugin-b@marketplace")

				Expect(result.ExitCode).To(Equal(0))
				Expect(result.Stdout).To(ContainSubstring("Removed plugin-b@marketplace from project scope"))
				Expect(result.Stdout).To(ContainSubstring(".claudeup.json"))

				// Verify the file was updated
				updatedConfig := helpers.LoadJSON(filepath.Join(projectDir, ".claudeup.json"))
				plugins := updatedConfig["plugins"].([]interface{})
				Expect(plugins).To(HaveLen(2))
				Expect(plugins[0]).To(Equal("plugin-a@marketplace"))
				Expect(plugins[1]).To(Equal("plugin-c@marketplace"))
			})

			It("should error if plugin not in config", func() {
				result := env.RunInDir(projectDir, "profile", "clean", "--scope", "project", "nonexistent@marketplace")

				Expect(result.ExitCode).NotTo(Equal(0))
				Expect(result.Stderr).To(ContainSubstring("not found in project scope config"))
			})
		})

		Context("removing plugin from local scope", func() {
			BeforeEach(func() {
				// Create .claudeup.local.json with plugins
				localConfig := map[string]interface{}{
					"version": "1",
					"profile": "local-profile",
					"plugins": []string{
						"local-a@marketplace",
						"local-b@marketplace",
					},
				}
				helpers.WriteJSON(filepath.Join(projectDir, ".claudeup.local.json"), localConfig)
			})

			It("should remove plugin from .claudeup.local.json", func() {
				result := env.RunInDir(projectDir, "profile", "clean", "--scope", "local", "local-a@marketplace")

				Expect(result.ExitCode).To(Equal(0))
				Expect(result.Stdout).To(ContainSubstring("Removed local-a@marketplace from local scope"))
				Expect(result.Stdout).To(ContainSubstring(".claudeup.local.json"))

				// Verify the file was updated
				updatedConfig := helpers.LoadJSON(filepath.Join(projectDir, ".claudeup.local.json"))
				plugins := updatedConfig["plugins"].([]interface{})
				Expect(plugins).To(HaveLen(1))
				Expect(plugins[0]).To(Equal("local-b@marketplace"))
			})
		})

		Context("error handling", func() {
			It("should require --scope flag", func() {
				result := env.RunInDir(projectDir, "profile", "clean", "plugin@marketplace")

				Expect(result.ExitCode).NotTo(Equal(0))
				Expect(result.Stderr).To(ContainSubstring("required flag"))
			})

			It("should validate scope value", func() {
				result := env.RunInDir(projectDir, "profile", "clean", "--scope", "invalid", "plugin@marketplace")

				Expect(result.ExitCode).NotTo(Equal(0))
				Expect(result.Stderr).To(ContainSubstring("invalid scope"))
				Expect(result.Stderr).To(ContainSubstring("must be 'project' or 'local'"))
			})

			It("should error if config file doesn't exist", func() {
				result := env.RunInDir(projectDir, "profile", "clean", "--scope", "project", "plugin@marketplace")

				Expect(result.ExitCode).NotTo(Equal(0))
				Expect(result.Stderr).To(ContainSubstring("no .claudeup.json file found"))
			})
		})
	})

	Describe("End-to-end drift workflow", func() {
		It("should detect drift and allow cleanup", func() {
			// 1. Create config with drifted plugins
			projectConfig := map[string]interface{}{
				"version": "1",
				"profile": "test-profile",
				"plugins": []string{
					"drifted-a@marketplace",
					"drifted-b@marketplace",
				},
			}
			helpers.WriteJSON(filepath.Join(projectDir, ".claudeup.json"), projectConfig)

			settings := map[string]interface{}{
				"enabledPlugins": map[string]bool{},
			}
			helpers.WriteJSON(filepath.Join(env.ClaudeDir, "settings.json"), settings)

			// 2. Status should show drift
			statusResult := env.RunInDir(projectDir, "status")
			Expect(statusResult.ExitCode).To(Equal(0))
			Expect(statusResult.Stdout).To(ContainSubstring("drifted-a@marketplace"))
			Expect(statusResult.Stdout).To(ContainSubstring("drifted-b@marketplace"))
			Expect(statusResult.Stdout).To(ContainSubstring("(project scope)"))

			// 3. Clean one plugin
			cleanResult := env.RunInDir(projectDir, "profile", "clean", "--scope", "project", "drifted-a@marketplace")
			Expect(cleanResult.ExitCode).To(Equal(0))

			// 4. Status should show only remaining drift
			statusResult2 := env.RunInDir(projectDir, "status")
			Expect(statusResult2.ExitCode).To(Equal(0))
			Expect(statusResult2.Stdout).NotTo(ContainSubstring("drifted-a@marketplace"))
			Expect(statusResult2.Stdout).To(ContainSubstring("drifted-b@marketplace"))

			// 5. Clean remaining plugin
			cleanResult2 := env.RunInDir(projectDir, "profile", "clean", "--scope", "project", "drifted-b@marketplace")
			Expect(cleanResult2.ExitCode).To(Equal(0))

			// 6. Status should show no drift
			statusResult3 := env.RunInDir(projectDir, "status")
			Expect(statusResult3.ExitCode).To(Equal(0))
			Expect(statusResult3.Stdout).NotTo(ContainSubstring("Configuration Drift Detected"))
			Expect(statusResult3.Stdout).NotTo(ContainSubstring("orphaned config"))
		})
	})
})
