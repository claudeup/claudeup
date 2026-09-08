// ABOUTME: Acceptance tests for the --strict flag on profile apply
// ABOUTME: Verifies missing extensions abort the apply before anything is changed
package acceptance

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/claudeup/claudeup/v5/test/helpers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("profile apply --strict", func() {
	var env *helpers.TestEnv

	// writeProfile writes an arbitrary profile document to the profiles directory.
	writeProfile := func(name string, doc map[string]any) {
		data, err := json.MarshalIndent(doc, "", "  ")
		Expect(err).NotTo(HaveOccurred())
		Expect(os.WriteFile(filepath.Join(env.ProfilesDir, name+".json"), data, 0644)).To(Succeed())
	}

	// userRuleSymlinkExists reports whether the user-scope symlink for a rule was created.
	userRuleSymlinkExists := func(name string) bool {
		_, err := os.Lstat(filepath.Join(env.ClaudeDir, "rules", name))
		return err == nil
	}

	BeforeEach(func() {
		env = helpers.NewTestEnv(binaryPath)

		// Extension storage holds exactly one rule.
		rulesDir := filepath.Join(env.ClaudeupDir, "ext", "rules")
		Expect(os.MkdirAll(rulesDir, 0755)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(rulesDir, "present.md"), []byte("# present"), 0644)).To(Succeed())
	})

	Context("legacy profile referencing a missing extension", func() {
		BeforeEach(func() {
			// A plugin is included so the apply has a diff and reaches the extension step.
			writeProfile("strict-legacy", map[string]any{
				"name":    "strict-legacy",
				"plugins": []string{"plugin-a@market"},
				"extensions": map[string]any{
					"rules": []string{"present.md", "missing.md"},
				},
			})
		})

		It("fails without applying anything when --strict is set", func() {
			result := env.Run("profile", "apply", "strict-legacy", "-y", "--strict")

			Expect(result.ExitCode).NotTo(Equal(0))
			Expect(result.Combined()).To(ContainSubstring("rules/missing.md"))
			Expect(result.Combined()).NotTo(ContainSubstring("Profile applied"))

			// Nothing was touched: the present rule was not enabled either.
			Expect(userRuleSymlinkExists("present.md")).To(BeFalse())
			Expect(env.IsPluginEnabled("plugin-a@market")).To(BeFalse())
		})

		It("warns and continues without --strict", func() {
			result := env.Run("profile", "apply", "strict-legacy", "-y")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Combined()).To(ContainSubstring("not found, skipping"))
			Expect(result.Combined()).To(ContainSubstring("Profile applied"))
			Expect(userRuleSymlinkExists("present.md")).To(BeTrue())
		})

		It("runs the check before --replace clears the scope", func() {
			env.CreateSettings(map[string]bool{"keep@market": true})

			result := env.Run("profile", "apply", "strict-legacy", "-y", "--strict", "--replace")

			Expect(result.ExitCode).NotTo(Equal(0))
			Expect(result.Combined()).To(ContainSubstring("rules/missing.md"))
			Expect(result.Combined()).NotTo(ContainSubstring("Cleared user scope"))
			Expect(env.IsPluginEnabled("keep@market")).To(BeTrue())
		})
	})

	Context("legacy profile whose extensions are all present", func() {
		BeforeEach(func() {
			writeProfile("strict-ok", map[string]any{
				"name":    "strict-ok",
				"plugins": []string{"plugin-a@market"},
				"extensions": map[string]any{
					"rules": []string{"present.md"},
				},
			})
		})

		It("applies normally with --strict", func() {
			result := env.Run("profile", "apply", "strict-ok", "-y", "--strict")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Combined()).To(ContainSubstring("Profile applied"))
			Expect(userRuleSymlinkExists("present.md")).To(BeTrue())
		})

		It("still previews with --strict --dry-run", func() {
			result := env.Run("profile", "apply", "strict-ok", "-y", "--strict", "--dry-run")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Combined()).To(ContainSubstring("Dry run"))
			Expect(result.Combined()).NotTo(ContainSubstring("Profile applied"))
			Expect(userRuleSymlinkExists("present.md")).To(BeFalse())
		})
	})

	Context("--strict combined with --dry-run on a missing extension", func() {
		BeforeEach(func() {
			writeProfile("strict-dry", map[string]any{
				"name":    "strict-dry",
				"plugins": []string{"plugin-a@market"},
				"extensions": map[string]any{
					"rules": []string{"missing.md"},
				},
			})
		})

		It("fails on the missing extension instead of previewing", func() {
			result := env.Run("profile", "apply", "strict-dry", "-y", "--strict", "--dry-run")

			Expect(result.ExitCode).NotTo(Equal(0))
			Expect(result.Combined()).To(ContainSubstring("rules/missing.md"))
			Expect(result.Combined()).NotTo(ContainSubstring("Dry run"))
			Expect(userRuleSymlinkExists("present.md")).To(BeFalse())
		})
	})

	Context("stack profile whose included profile references a missing extension", func() {
		BeforeEach(func() {
			writeProfile("strict-base", map[string]any{
				"name":    "strict-base",
				"plugins": []string{"plugin-a@market"},
				"extensions": map[string]any{
					"rules": []string{"present.md", "missing.md"},
				},
			})
			writeProfile("strict-stack", map[string]any{
				"name":     "strict-stack",
				"includes": []string{"strict-base"},
			})
		})

		It("fails after resolving includes when --strict is set", func() {
			result := env.Run("profile", "apply", "strict-stack", "-y", "--strict")

			Expect(result.ExitCode).NotTo(Equal(0))
			Expect(result.Combined()).To(ContainSubstring("rules/missing.md"))
			Expect(result.Combined()).NotTo(ContainSubstring("Profile applied"))
			Expect(userRuleSymlinkExists("present.md")).To(BeFalse())
			Expect(env.IsPluginEnabled("plugin-a@market")).To(BeFalse())
		})

		It("warns and continues without --strict", func() {
			result := env.Run("profile", "apply", "strict-stack", "-y")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Combined()).To(ContainSubstring("Profile applied"))
		})
	})

	Context("multi-scope profile referencing a missing project-scope extension", func() {
		var projectDir string

		BeforeEach(func() {
			projectDir = env.ProjectDir("strict-project")
			writeProfile("strict-multi", map[string]any{
				"name": "strict-multi",
				"perScope": map[string]any{
					"user": map[string]any{
						"plugins": []string{"plugin-a@market"},
					},
					"project": map[string]any{
						"extensions": map[string]any{
							"rules": []string{"missing.md"},
						},
					},
				},
			})
		})

		It("fails before writing any scope settings when --strict is set", func() {
			result := env.RunInDir(projectDir, "profile", "apply", "strict-multi", "-y", "--strict")

			Expect(result.ExitCode).NotTo(Equal(0))
			Expect(result.Combined()).To(ContainSubstring("rules/missing.md"))
			Expect(result.Combined()).NotTo(ContainSubstring("Profile applied"))

			_, err := os.Stat(filepath.Join(projectDir, ".claude", "settings.json"))
			Expect(os.IsNotExist(err)).To(BeTrue(), "project settings.json should not have been written")
			Expect(env.IsPluginEnabled("plugin-a@market")).To(BeFalse())
		})

		It("warns and continues without --strict", func() {
			result := env.RunInDir(projectDir, "profile", "apply", "strict-multi", "-y")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Combined()).To(ContainSubstring("not found, skipping"))
			Expect(result.Combined()).To(ContainSubstring("Profile applied"))
		})
	})
})
