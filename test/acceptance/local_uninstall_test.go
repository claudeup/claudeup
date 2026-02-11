// ABOUTME: Acceptance tests for local uninstall command
// ABOUTME: Tests CLI behavior for removing items from local storage
package acceptance

import (
	"os"
	"path/filepath"

	"github.com/claudeup/claudeup/v5/test/helpers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("local uninstall", func() {
	var env *helpers.TestEnv

	BeforeEach(func() {
		env = helpers.NewTestEnv(binaryPath)

		// Create items in local storage
		rulesDir := filepath.Join(env.ClaudeupDir, "local", "rules")
		Expect(os.MkdirAll(rulesDir, 0755)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(rulesDir, "my-rule.md"), []byte("# Rule"), 0644)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(rulesDir, "other-rule.md"), []byte("# Other"), 0644)).To(Succeed())

		// Enable both via enabled.json
		enabledJSON := `{"rules":{"my-rule.md":true,"other-rule.md":true}}`
		Expect(os.WriteFile(filepath.Join(env.ClaudeupDir, "enabled.json"), []byte(enabledJSON), 0644)).To(Succeed())

		// Create symlinks in active directory
		activeDir := filepath.Join(env.ClaudeDir, "rules")
		Expect(os.MkdirAll(activeDir, 0755)).To(Succeed())
		Expect(os.Symlink(filepath.Join(rulesDir, "my-rule.md"), filepath.Join(activeDir, "my-rule.md"))).To(Succeed())
		Expect(os.Symlink(filepath.Join(rulesDir, "other-rule.md"), filepath.Join(activeDir, "other-rule.md"))).To(Succeed())
	})

	It("removes the item and shows success", func() {
		result := env.Run("local", "uninstall", "rules", "my-rule.md")

		Expect(result.ExitCode).To(Equal(0))
		Expect(result.Stdout).To(ContainSubstring("Removed"))
		Expect(result.Stdout).To(ContainSubstring("my-rule.md"))
	})

	It("removes the file from local storage", func() {
		env.Run("local", "uninstall", "rules", "my-rule.md")

		_, err := os.Stat(filepath.Join(env.ClaudeupDir, "local", "rules", "my-rule.md"))
		Expect(os.IsNotExist(err)).To(BeTrue())
	})

	It("removes the symlink from the active directory", func() {
		env.Run("local", "uninstall", "rules", "my-rule.md")

		_, err := os.Lstat(filepath.Join(env.ClaudeDir, "rules", "my-rule.md"))
		Expect(os.IsNotExist(err)).To(BeTrue())
	})

	It("does not affect other items", func() {
		env.Run("local", "uninstall", "rules", "my-rule.md")

		_, err := os.Stat(filepath.Join(env.ClaudeupDir, "local", "rules", "other-rule.md"))
		Expect(err).NotTo(HaveOccurred())
	})

	It("warns on not found items", func() {
		result := env.Run("local", "uninstall", "rules", "nonexistent.md")

		Expect(result.ExitCode).NotTo(Equal(0))
		Expect(result.Combined()).To(ContainSubstring("Not found"))
	})

	It("requires category and items arguments", func() {
		result := env.Run("local", "uninstall")
		Expect(result.ExitCode).NotTo(Equal(0))
	})
})
