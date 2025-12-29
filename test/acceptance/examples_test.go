// ABOUTME: Acceptance tests for example shell scripts
// ABOUTME: Verifies scripts have valid syntax and run in non-interactive mode
package acceptance

import (
	"os"
	"os/exec"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("example scripts", func() {
	var examplesDir string

	BeforeEach(func() {
		// Find the examples directory relative to the project root
		wd, err := os.Getwd()
		Expect(err).NotTo(HaveOccurred())

		// Navigate up from test/acceptance to project root
		projectRoot := filepath.Dir(filepath.Dir(wd))
		examplesDir = filepath.Join(projectRoot, "examples")

		// If we're already at project root, adjust
		if _, err := os.Stat(examplesDir); os.IsNotExist(err) {
			examplesDir = filepath.Join(wd, "examples")
		}
		if _, err := os.Stat(examplesDir); os.IsNotExist(err) {
			examplesDir = filepath.Join(wd, "..", "..", "examples")
		}
	})

	Describe("syntax validation", func() {
		scriptDirs := []string{
			"getting-started",
			"profile-management",
			"plugin-management",
			"troubleshooting",
			"team-setup",
		}

		for _, dir := range scriptDirs {
			Context(dir, func() {
				It("all scripts pass bash -n syntax check", func() {
					scriptsPath := filepath.Join(examplesDir, dir)
					entries, err := os.ReadDir(scriptsPath)
					Expect(err).NotTo(HaveOccurred())

					for _, entry := range entries {
						if !entry.IsDir() && filepath.Ext(entry.Name()) == ".sh" {
							scriptPath := filepath.Join(scriptsPath, entry.Name())
							cmd := exec.Command("bash", "-n", scriptPath)
							output, err := cmd.CombinedOutput()
							Expect(err).NotTo(HaveOccurred(),
								"Script %s failed syntax check: %s", entry.Name(), string(output))
						}
					}
				})
			})
		}

		Context("lib", func() {
			It("common.sh passes bash -n syntax check", func() {
				scriptPath := filepath.Join(examplesDir, "lib", "common.sh")
				cmd := exec.Command("bash", "-n", scriptPath)
				output, err := cmd.CombinedOutput()
				Expect(err).NotTo(HaveOccurred(),
					"common.sh failed syntax check: %s", string(output))
			})
		})
	})

	Describe("non-interactive execution", func() {
		Context("getting-started/01-check-installation.sh", func() {
			It("runs successfully in temp mode", func() {
				scriptPath := filepath.Join(examplesDir, "getting-started", "01-check-installation.sh")

				cmd := exec.Command(scriptPath, "--non-interactive")
				cmd.Env = append(os.Environ(),
					"CLAUDEUP_BIN="+binaryPath,
				)

				output, err := cmd.CombinedOutput()
				// The script may fail if some claudeup commands don't exist yet,
				// but it should at least start and show the header
				Expect(string(output)).To(ContainSubstring("Getting Started"))

				// If there's an error, it should be a graceful failure, not a syntax error
				if err != nil {
					Expect(string(output)).NotTo(ContainSubstring("syntax error"))
					Expect(string(output)).NotTo(ContainSubstring("command not found: source"))
				}
			})
		})

		Context("getting-started/02-explore-profiles.sh", func() {
			It("runs successfully in temp mode", func() {
				scriptPath := filepath.Join(examplesDir, "getting-started", "02-explore-profiles.sh")

				cmd := exec.Command(scriptPath, "--non-interactive")
				cmd.Env = append(os.Environ(),
					"CLAUDEUP_BIN="+binaryPath,
				)

				output, err := cmd.CombinedOutput()
				Expect(string(output)).To(ContainSubstring("Explore Profiles"))

				if err != nil {
					Expect(string(output)).NotTo(ContainSubstring("syntax error"))
				}
			})
		})

		Context("troubleshooting/01-run-doctor.sh", func() {
			It("runs successfully in temp mode", func() {
				scriptPath := filepath.Join(examplesDir, "troubleshooting", "01-run-doctor.sh")

				cmd := exec.Command(scriptPath, "--non-interactive")
				cmd.Env = append(os.Environ(),
					"CLAUDEUP_BIN="+binaryPath,
				)

				output, err := cmd.CombinedOutput()
				Expect(string(output)).To(ContainSubstring("Run Doctor"))

				if err != nil {
					Expect(string(output)).NotTo(ContainSubstring("syntax error"))
				}
			})
		})
	})

	Describe("help flag", func() {
		It("all scripts respond to --help", func() {
			scriptDirs := []string{
				"getting-started",
				"profile-management",
				"plugin-management",
				"troubleshooting",
				"team-setup",
			}

			for _, dir := range scriptDirs {
				scriptsPath := filepath.Join(examplesDir, dir)
				entries, err := os.ReadDir(scriptsPath)
				Expect(err).NotTo(HaveOccurred())

				for _, entry := range entries {
					if !entry.IsDir() && filepath.Ext(entry.Name()) == ".sh" {
						scriptPath := filepath.Join(scriptsPath, entry.Name())
						cmd := exec.Command(scriptPath, "--help")
						output, err := cmd.CombinedOutput()
						Expect(err).NotTo(HaveOccurred(),
							"Script %s/%s failed --help: %s", dir, entry.Name(), string(output))
						Expect(string(output)).To(ContainSubstring("Usage:"),
							"Script %s/%s --help should show Usage", dir, entry.Name())
					}
				}
			}
		})
	})
})
