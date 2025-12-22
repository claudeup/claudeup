// ABOUTME: Acceptance tests for scope restore command
// ABOUTME: Tests CLI behavior for restoring scope backups
package acceptance

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/claudeup/claudeup/test/helpers"
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
})
