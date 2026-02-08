// ABOUTME: Acceptance tests for local view markdown rendering
// ABOUTME: Tests glamour rendering and --raw flag for local items
package acceptance

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/claudeup/claudeup/v4/test/helpers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("local view", func() {
	var env *helpers.TestEnv

	BeforeEach(func() {
		env = helpers.NewTestEnv(binaryPath)

		// Create library structure with items
		libraryDir := filepath.Join(env.ClaudeDir, ".library")

		agentsDir := filepath.Join(libraryDir, "agents")
		hooksDir := filepath.Join(libraryDir, "hooks")
		skillsDir := filepath.Join(libraryDir, "skills", "test-skill")
		Expect(os.MkdirAll(agentsDir, 0755)).To(Succeed())
		Expect(os.MkdirAll(hooksDir, 0755)).To(Succeed())
		Expect(os.MkdirAll(skillsDir, 0755)).To(Succeed())

		Expect(os.WriteFile(filepath.Join(agentsDir, "test-agent.md"),
			[]byte("# Test Agent\n\nAn agent for **testing**."), 0644)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(hooksDir, "format-check.sh"),
			[]byte("#!/bin/bash\necho 'checking format'"), 0644)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(skillsDir, "SKILL.md"),
			[]byte("# Test Skill\n\nA skill for **testing**."), 0644)).To(Succeed())

		// Create enabled.json so items are discoverable
		enabledConfig := map[string]map[string]bool{
			"agents": {"test-agent.md": true},
			"hooks":  {"format-check.sh": true},
			"skills": {"test-skill": true},
		}
		enabledData, err := json.MarshalIndent(enabledConfig, "", "  ")
		Expect(err).NotTo(HaveOccurred())
		Expect(os.WriteFile(filepath.Join(libraryDir, "enabled.json"), enabledData, 0644)).To(Succeed())
	})

	Describe("markdown rendering for agents", func() {
		It("renders markdown content", func() {
			result := env.Run("local", "view", "agents", "test-agent")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("Test Agent"))
			Expect(result.Stdout).To(ContainSubstring("testing"))
		})
	})

	Describe("markdown rendering for skills", func() {
		It("renders skill SKILL.md content", func() {
			result := env.Run("local", "view", "skills", "test-skill")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("Test Skill"))
			Expect(result.Stdout).To(ContainSubstring("testing"))
		})
	})

	Describe("--raw flag", func() {
		It("bypasses markdown rendering for agents", func() {
			result := env.Run("local", "view", "agents", "test-agent", "--raw")

			Expect(result.ExitCode).To(Equal(0))
			// Raw output should contain the markdown syntax
			Expect(result.Stdout).To(ContainSubstring("# Test Agent"))
			Expect(result.Stdout).To(ContainSubstring("**testing**"))
		})

		It("bypasses markdown rendering for skills", func() {
			result := env.Run("local", "view", "skills", "test-skill", "--raw")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("# Test Skill"))
			Expect(result.Stdout).To(ContainSubstring("**testing**"))
		})
	})

	Describe("non-markdown files", func() {
		It("shows raw content for shell scripts", func() {
			result := env.Run("local", "view", "hooks", "format-check")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("#!/bin/bash"))
			Expect(result.Stdout).To(ContainSubstring("echo 'checking format'"))
		})
	})
})
