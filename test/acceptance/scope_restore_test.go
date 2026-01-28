// ABOUTME: Acceptance tests for scope restore command
// ABOUTME: Tests CLI behavior for restoring scope backups
package acceptance

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/claudeup/claudeup/v3/test/helpers"
)

var _ = Describe("claudeup scope restore", func() {
	var (
		env        *helpers.TestEnv
		binaryPath string
	)

	BeforeEach(func() {
		binaryPath = helpers.BuildBinary()
		env = helpers.NewTestEnv(binaryPath)
	})

	AfterEach(func() {
		env.Cleanup()
	})

	Describe("scope restore user", func() {
		It("should restore from backup when one exists", func() {
			// Create backup directory and file
			backupDir := filepath.Join(env.TempDir, ".claudeup", "backups")
			err := os.MkdirAll(backupDir, 0755)
			Expect(err).NotTo(HaveOccurred())

			backupContent := `{"enabledPlugins":{"restored@test":true}}`
			err = os.WriteFile(filepath.Join(backupDir, "user-scope.json"), []byte(backupContent), 0644)
			Expect(err).NotTo(HaveOccurred())

			// Create current settings (different from backup)
			helpers.WriteJSON(filepath.Join(env.ClaudeDir, "settings.json"), map[string]interface{}{
				"enabledPlugins": map[string]bool{},
			})

			// Restore with --force
			result := env.Run("scope", "restore", "user", "--force")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("Restored user scope"))
		})

		It("should error when no backup exists", func() {
			result := env.Run("scope", "restore", "user")

			Expect(result.ExitCode).NotTo(Equal(0))
			Expect(result.Stderr).To(ContainSubstring("no backup found"))
		})
	})

	Describe("scope restore project", func() {
		It("should reject project scope with helpful message", func() {
			result := env.Run("scope", "restore", "project")

			Expect(result.ExitCode).NotTo(Equal(0))
			Expect(result.Stderr).To(ContainSubstring("not supported"))
			Expect(result.Stderr).To(ContainSubstring("git checkout"))
		})
	})

	Describe("scope restore local", func() {
		It("should restore from backup using project-specific hash", func() {
			// Create a project directory
			projectDir := filepath.Join(env.TempDir, "my-project")
			err := os.MkdirAll(filepath.Join(projectDir, ".claude"), 0755)
			Expect(err).NotTo(HaveOccurred())

			// Create local settings file
			localSettings := filepath.Join(projectDir, ".claude", "settings.local.json")
			err = os.WriteFile(localSettings, []byte(`{"enabledPlugins":{"local@test":true}}`), 0644)
			Expect(err).NotTo(HaveOccurred())

			// Clear with backup (this creates the hash-named backup)
			result := env.RunInDir(projectDir, "scope", "clear", "local", "--backup", "--force")
			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("Backup saved"))

			// Verify local settings was cleared
			_, err = os.Stat(localSettings)
			Expect(os.IsNotExist(err)).To(BeTrue())

			// Restore from backup
			result = env.RunInDir(projectDir, "scope", "restore", "local", "--force")
			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("Restored local scope"))

			// Verify content was restored
			content, err := os.ReadFile(localSettings)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(content)).To(ContainSubstring("local@test"))
		})

		It("should error when no backup exists for this project", func() {
			// Create a project directory (no backup exists)
			projectDir := filepath.Join(env.TempDir, "new-project")
			err := os.MkdirAll(projectDir, 0755)
			Expect(err).NotTo(HaveOccurred())

			result := env.RunInDir(projectDir, "scope", "restore", "local")

			Expect(result.ExitCode).NotTo(Equal(0))
			Expect(result.Stderr).To(ContainSubstring("no backup found"))
		})

		It("should use project-specific backup (different projects have different backups)", func() {
			// Create two project directories
			projectA := filepath.Join(env.TempDir, "project-a")
			projectB := filepath.Join(env.TempDir, "project-b")
			err := os.MkdirAll(filepath.Join(projectA, ".claude"), 0755)
			Expect(err).NotTo(HaveOccurred())
			err = os.MkdirAll(filepath.Join(projectB, ".claude"), 0755)
			Expect(err).NotTo(HaveOccurred())

			// Create different local settings for each project
			err = os.WriteFile(
				filepath.Join(projectA, ".claude", "settings.local.json"),
				[]byte(`{"enabledPlugins":{"pluginA@test":true}}`),
				0644,
			)
			Expect(err).NotTo(HaveOccurred())
			err = os.WriteFile(
				filepath.Join(projectB, ".claude", "settings.local.json"),
				[]byte(`{"enabledPlugins":{"pluginB@test":true}}`),
				0644,
			)
			Expect(err).NotTo(HaveOccurred())

			// Backup project A
			result := env.RunInDir(projectA, "scope", "clear", "local", "--backup", "--force")
			Expect(result.ExitCode).To(Equal(0))

			// Backup project B
			result = env.RunInDir(projectB, "scope", "clear", "local", "--backup", "--force")
			Expect(result.ExitCode).To(Equal(0))

			// Restore project A and verify it gets pluginA, not pluginB
			result = env.RunInDir(projectA, "scope", "restore", "local", "--force")
			Expect(result.ExitCode).To(Equal(0))

			content, err := os.ReadFile(filepath.Join(projectA, ".claude", "settings.local.json"))
			Expect(err).NotTo(HaveOccurred())
			Expect(string(content)).To(ContainSubstring("pluginA"))
			Expect(string(content)).NotTo(ContainSubstring("pluginB"))
		})
	})
})
