// ABOUTME: Integration tests for profile wizard functionality
// ABOUTME: Tests name prompting, validation, and wizard helpers
package profile_test

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"

	"github.com/claudeup/claudeup/v5/internal/profile"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// noGumLookPath simulates gum not being installed, forcing fallback paths.
func noGumLookPath(name string) (string, error) {
	return "", fmt.Errorf("executable file not found in $PATH")
}

// testWizardIO creates a WizardIO with piped input and no gum.
func testWizardIO(input string) (profile.WizardIO, *bytes.Buffer) {
	out := &bytes.Buffer{}
	in := strings.NewReader(input)
	return profile.NewWizardIO(in, out, &bytes.Buffer{}, noGumLookPath), out
}

// gumLookPath simulates gum being installed.
func gumLookPath(name string) (string, error) {
	return "/usr/bin/gum", nil
}

// gumWizardIO creates a WizardIO with gum available and a custom GumRun.
// Returns the WizardIO, the stdout buffer, and the stderr buffer for assertion.
func gumWizardIO(input string, runner func(args ...string) ([]byte, error)) (profile.WizardIO, *bytes.Buffer, *bytes.Buffer) {
	out := &bytes.Buffer{}
	errBuf := &bytes.Buffer{}
	in := strings.NewReader(input)
	wio := profile.NewWizardIO(in, out, errBuf, gumLookPath)
	wio.GumRun = runner
	return wio, out, errBuf
}

// makeExitErrorWithCode returns an *exec.ExitError with the given exit code.
// Fails the current Ginkgo spec if the shell command does not produce an ExitError.
func makeExitErrorWithCode(code int) *exec.ExitError {
	err := exec.Command("sh", "-c", fmt.Sprintf("exit %d", code)).Run()
	exitErr, ok := err.(*exec.ExitError)
	Expect(ok).To(BeTrue(), fmt.Sprintf(
		"exec.Command(\"sh\", \"-c\", \"exit %d\").Run() returned %T, not *exec.ExitError", code, err))
	return exitErr
}

var _ = Describe("Wizard", func() {
	Describe("ValidateName", func() {
		It("accepts valid profile names", func() {
			err := profile.ValidateName("my-profile")
			Expect(err).To(BeNil())
		})

		It("rejects empty names", func() {
			err := profile.ValidateName("")
			Expect(err).To(MatchError("profile name cannot be empty"))
		})

		It("rejects reserved name 'current'", func() {
			err := profile.ValidateName("current")
			Expect(err).To(MatchError("'current' is a reserved name"))
		})

		It("rejects names with invalid characters", func() {
			err := profile.ValidateName("my profile!")
			Expect(err).To(MatchError(ContainSubstring("invalid characters")))
		})
	})

	Describe("GetAvailableMarketplaces", func() {
		It("returns embedded marketplaces", func() {
			marketplaces := profile.GetAvailableMarketplaces()

			// Should return at least one marketplace from embedded profiles
			Expect(marketplaces).NotTo(BeEmpty())

			// All marketplaces should have valid source type and identifier
			for _, m := range marketplaces {
				// Accept github, git, or directory as valid source types
				Expect(m.Source).To(BeElementOf("github", "git", "directory"))
				// Must have either Repo or URL set
				Expect(m.Repo != "" || m.URL != "").To(BeTrue(), "marketplace must have Repo or URL")
			}
		})

		It("includes marketplace display names", func() {
			marketplaces := profile.GetAvailableMarketplaces()

			for _, m := range marketplaces {
				Expect(m.DisplayName()).NotTo(BeEmpty())
			}
		})
	})

	Describe("PromptForName", func() {
		It("reads and validates a valid name from input", func() {
			wio, _ := testWizardIO("my-profile\n")

			name, err := profile.PromptForName(wio)
			Expect(err).NotTo(HaveOccurred())
			Expect(name).To(Equal("my-profile"))
		})

		It("returns error on EOF", func() {
			wio, _ := testWizardIO("")

			_, err := profile.PromptForName(wio)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to read input"))
		})

		It("re-prompts on invalid then accepts valid name", func() {
			// First line is invalid (has spaces), second is valid
			wio, out := testWizardIO("bad name!\nmy-profile\n")

			name, err := profile.PromptForName(wio)
			Expect(err).NotTo(HaveOccurred())
			Expect(name).To(Equal("my-profile"))
			// Should have printed an error for the invalid name
			Expect(out.String()).To(ContainSubstring("Error:"))
		})

		It("re-prompts on empty input then accepts valid name", func() {
			// First line is blank (empty name), second is valid
			wio, out := testWizardIO("\nmy-profile\n")

			name, err := profile.PromptForName(wio)
			Expect(err).NotTo(HaveOccurred())
			Expect(name).To(Equal("my-profile"))
			Expect(out.String()).To(ContainSubstring("Error: profile name cannot be empty"))
		})
	})

	Describe("SelectMarketplaces", func() {
		It("returns error if no marketplaces available", func() {
			wio, _ := testWizardIO("")
			selected, err := profile.SelectMarketplaces(wio, []profile.Marketplace{})
			Expect(err).To(MatchError("no marketplaces available"))
			Expect(selected).To(BeNil())
		})

		It("selects a marketplace by number via fallback", func() {
			marketplaces := []profile.Marketplace{
				{Source: "github", Repo: "owner/first"},
				{Source: "github", Repo: "owner/second"},
			}
			wio, out := testWizardIO("2\n")

			selected, err := profile.SelectMarketplaces(wio, marketplaces)
			Expect(err).NotTo(HaveOccurred())
			Expect(selected).To(HaveLen(1))
			Expect(selected[0].Repo).To(Equal("owner/second"))
			// Should have shown the numbered menu
			Expect(out.String()).To(ContainSubstring("1) owner/first"))
			Expect(out.String()).To(ContainSubstring("2) owner/second"))
		})

		It("selects multiple marketplaces by comma-separated numbers", func() {
			marketplaces := []profile.Marketplace{
				{Source: "github", Repo: "owner/first"},
				{Source: "github", Repo: "owner/second"},
				{Source: "github", Repo: "owner/third"},
			}
			wio, _ := testWizardIO("1,3\n")

			selected, err := profile.SelectMarketplaces(wio, marketplaces)
			Expect(err).NotTo(HaveOccurred())
			Expect(selected).To(HaveLen(2))
			Expect(selected[0].Repo).To(Equal("owner/first"))
			Expect(selected[1].Repo).To(Equal("owner/third"))
		})

		It("returns error on empty input", func() {
			marketplaces := []profile.Marketplace{
				{Source: "github", Repo: "owner/first"},
			}
			wio, _ := testWizardIO("\n")

			_, err := profile.SelectMarketplaces(wio, marketplaces)
			Expect(err).To(MatchError("no marketplaces selected"))
		})

		It("returns error on EOF", func() {
			marketplaces := []profile.Marketplace{
				{Source: "github", Repo: "owner/first"},
			}
			wio, _ := testWizardIO("")

			_, err := profile.SelectMarketplaces(wio, marketplaces)
			Expect(err).To(HaveOccurred())
		})

		It("returns error on out-of-range selection", func() {
			marketplaces := []profile.Marketplace{
				{Source: "github", Repo: "owner/first"},
				{Source: "github", Repo: "owner/second"},
			}
			wio, _ := testWizardIO("99\n")

			_, err := profile.SelectMarketplaces(wio, marketplaces)
			Expect(err).To(MatchError("invalid selection: 99"))
		})

		It("returns error on non-numeric selection", func() {
			marketplaces := []profile.Marketplace{
				{Source: "github", Repo: "owner/first"},
			}
			wio, _ := testWizardIO("abc\n")

			_, err := profile.SelectMarketplaces(wio, marketplaces)
			Expect(err).To(MatchError("invalid selection: abc"))
		})

		It("deduplicates repeated marketplace numbers", func() {
			marketplaces := []profile.Marketplace{
				{Source: "github", Repo: "owner/first"},
				{Source: "github", Repo: "owner/second"},
			}
			wio, _ := testWizardIO("1,1,2\n")

			selected, err := profile.SelectMarketplaces(wio, marketplaces)
			Expect(err).NotTo(HaveOccurred())
			Expect(selected).To(HaveLen(2))
			Expect(selected[0].Repo).To(Equal("owner/first"))
			Expect(selected[1].Repo).To(Equal("owner/second"))
		})
	})

	Describe("SelectPluginsForMarketplace", func() {
		It("returns error on EOF for category-based marketplace", func() {
			// wshobson/agents has categories — the fallback category selection
			// hits EOF and surfaces a "failed to read input" error.
			marketplace := profile.Marketplace{
				Source: "github",
				Repo:   "wshobson/agents",
			}
			wio, _ := testWizardIO("")

			_, err := profile.SelectPluginsForMarketplace(wio, marketplace)
			Expect(err).To(HaveOccurred())
		})

		It("selects categories then returns selected plugins", func() {
			marketplace := profile.Marketplace{
				Source: "github",
				Repo:   "wshobson/agents",
			}
			// Select category 1 (Core Development), then plugin 1 from refinement list
			wio, _ := testWizardIO("1\n1\n")

			plugins, err := profile.SelectPluginsForMarketplace(wio, marketplace)
			Expect(err).NotTo(HaveOccurred())
			Expect(plugins).To(HaveLen(1))
		})

		It("returns empty plugins when 'q' skips category selection", func() {
			marketplace := profile.Marketplace{
				Source: "github",
				Repo:   "wshobson/agents",
			}
			wio, _ := testWizardIO("q\n")

			plugins, err := profile.SelectPluginsForMarketplace(wio, marketplace)
			Expect(err).NotTo(HaveOccurred())
			Expect(plugins).To(BeEmpty())
		})

		It("uses flat selection for marketplaces without categories", func() {
			marketplace := profile.Marketplace{
				Source: "github",
				Repo:   "unknown/marketplace",
			}

			// Flat selection path — listPluginsFromMarketplace fails gracefully
			// for unknown marketplace, returns empty list
			wio, _ := testWizardIO("")
			plugins, err := profile.SelectPluginsForMarketplace(wio, marketplace)
			Expect(err).To(BeNil())
			Expect(plugins).To(BeEmpty())
		})
	})

	Describe("PromptForDescription", func() {
		It("accepts auto-generated description when user declines edit", func() {
			wio, _ := testWizardIO("n\n")

			desc, err := profile.PromptForDescription(wio, "Auto description")
			Expect(err).NotTo(HaveOccurred())
			Expect(desc).To(Equal("Auto description"))
		})

		It("returns auto-generated on EOF", func() {
			wio, _ := testWizardIO("")

			desc, err := profile.PromptForDescription(wio, "Auto description")
			Expect(err).NotTo(HaveOccurred())
			Expect(desc).To(Equal("Auto description"))
		})

		It("allows user to enter custom description", func() {
			wio, _ := testWizardIO("y\nMy custom description\n")

			desc, err := profile.PromptForDescription(wio, "Auto description")
			Expect(err).NotTo(HaveOccurred())
			Expect(desc).To(Equal("My custom description"))
		})

		It("returns error when user says yes but input ends", func() {
			wio, _ := testWizardIO("y\n")

			_, err := profile.PromptForDescription(wio, "Auto description")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to read input"))
		})

		It("uses auto-generated if user says yes but enters empty description", func() {
			wio, _ := testWizardIO("y\n\n")

			desc, err := profile.PromptForDescription(wio, "Auto description")
			Expect(err).NotTo(HaveOccurred())
			Expect(desc).To(Equal("Auto description"))
		})
	})

	Describe("Gum error classification", func() {
		Describe("editDescription via PromptForDescription", func() {
			It("warns on gum crash and falls back to placeholder", func() {
				crashErr := fmt.Errorf("gum: permission denied")
				runner := func(args ...string) ([]byte, error) {
					if args[0] == "confirm" {
						return nil, nil // user said "yes" to editing
					}
					// "write" command crashes
					return nil, crashErr
				}
				wio, _, errBuf := gumWizardIO("", runner)

				desc, err := profile.PromptForDescription(wio, "Auto description")
				Expect(err).NotTo(HaveOccurred())
				Expect(desc).To(Equal("Auto description"))
				Expect(errBuf.String()).To(ContainSubstring("Warning:"))
				Expect(errBuf.String()).To(ContainSubstring("permission denied"))
			})

			It("does not warn on user cancellation", func() {
				exitErr := makeExitErrorWithCode(1)
				runner := func(args ...string) ([]byte, error) {
					if args[0] == "confirm" {
						return nil, nil // user said "yes"
					}
					return nil, exitErr // user cancelled gum write
				}
				wio, _, errBuf := gumWizardIO("", runner)

				desc, err := profile.PromptForDescription(wio, "Auto description")
				Expect(err).NotTo(HaveOccurred())
				Expect(desc).To(Equal("Auto description"))
				Expect(errBuf.String()).To(BeEmpty())
			})
		})

		Describe("PromptForDescription confirm step", func() {
			It("warns on gum crash during confirmation", func() {
				crashErr := fmt.Errorf("gum: TTY required")
				runner := func(args ...string) ([]byte, error) {
					return nil, crashErr
				}
				wio, _, errBuf := gumWizardIO("", runner)

				desc, err := profile.PromptForDescription(wio, "Auto description")
				Expect(err).NotTo(HaveOccurred())
				Expect(desc).To(Equal("Auto description"))
				Expect(errBuf.String()).To(ContainSubstring("Warning:"))
				Expect(errBuf.String()).To(ContainSubstring("TTY required"))
			})

			It("does not warn when user says no", func() {
				exitErr := makeExitErrorWithCode(1)
				runner := func(args ...string) ([]byte, error) {
					return nil, exitErr // user said "no"
				}
				wio, _, errBuf := gumWizardIO("", runner)

				desc, err := profile.PromptForDescription(wio, "Auto description")
				Expect(err).NotTo(HaveOccurred())
				Expect(desc).To(Equal("Auto description"))
				Expect(errBuf.String()).To(BeEmpty())
			})
		})

		// refinePluginSelection tests are in internal/profile/wizard_test.go
		// (same package, can access unexported function)

		Describe("SelectMarketplaces", func() {
			It("warns on gum crash", func() {
				crashErr := fmt.Errorf("gum: broken pipe")
				runner := func(args ...string) ([]byte, error) {
					return nil, crashErr
				}
				wio, _, errBuf := gumWizardIO("", runner)

				marketplaces := []profile.Marketplace{
					{Source: "github", Repo: "owner/first"},
				}
				_, err := profile.SelectMarketplaces(wio, marketplaces)
				Expect(err).To(HaveOccurred())
				Expect(errBuf.String()).To(ContainSubstring("Warning:"))
				Expect(errBuf.String()).To(ContainSubstring("broken pipe"))
			})

			It("does not warn on user cancellation", func() {
				exitErr := makeExitErrorWithCode(1)
				runner := func(args ...string) ([]byte, error) {
					return nil, exitErr
				}
				wio, _, errBuf := gumWizardIO("", runner)

				marketplaces := []profile.Marketplace{
					{Source: "github", Repo: "owner/first"},
				}
				_, err := profile.SelectMarketplaces(wio, marketplaces)
				Expect(err).To(HaveOccurred())
				Expect(errBuf.String()).To(BeEmpty())
			})
		})

		Describe("selectCategories via SelectPluginsForMarketplace", func() {
			It("warns on gum crash during category selection", func() {
				crashErr := fmt.Errorf("gum: signal killed")
				runner := func(args ...string) ([]byte, error) {
					return nil, crashErr
				}
				wio, _, errBuf := gumWizardIO("", runner)

				marketplace := profile.Marketplace{
					Source: "github",
					Repo:   "wshobson/agents",
				}
				_, err := profile.SelectPluginsForMarketplace(wio, marketplace)
				Expect(err).To(HaveOccurred())
				Expect(errBuf.String()).To(ContainSubstring("Warning:"))
				Expect(errBuf.String()).To(ContainSubstring("signal killed"))
			})

			It("does not warn on user cancellation", func() {
				exitErr := makeExitErrorWithCode(1)
				runner := func(args ...string) ([]byte, error) {
					return nil, exitErr
				}
				wio, _, errBuf := gumWizardIO("", runner)

				marketplace := profile.Marketplace{
					Source: "github",
					Repo:   "wshobson/agents",
				}
				_, err := profile.SelectPluginsForMarketplace(wio, marketplace)
				Expect(err).To(HaveOccurred())
				Expect(errBuf.String()).To(BeEmpty())
			})
		})

		Describe("non-cancel ExitError (exit code 2)", func() {
			It("warns on ExitError with non-cancel exit code in editDescription", func() {
				exitErr := makeExitErrorWithCode(2)
				runner := func(args ...string) ([]byte, error) {
					if args[0] == "confirm" {
						return nil, nil // user said "yes" to editing
					}
					return nil, exitErr // gum write crashed with exit 2
				}
				wio, _, errBuf := gumWizardIO("", runner)

				desc, err := profile.PromptForDescription(wio, "Auto description")
				Expect(err).NotTo(HaveOccurred())
				Expect(desc).To(Equal("Auto description"))
				Expect(errBuf.String()).To(ContainSubstring("Warning:"))
			})

			It("warns on ExitError with non-cancel exit code in SelectMarketplaces", func() {
				exitErr := makeExitErrorWithCode(2)
				runner := func(args ...string) ([]byte, error) {
					return nil, exitErr
				}
				wio, _, errBuf := gumWizardIO("", runner)

				marketplaces := []profile.Marketplace{
					{Source: "github", Repo: "owner/first"},
				}
				_, err := profile.SelectMarketplaces(wio, marketplaces)
				Expect(err).To(HaveOccurred())
				Expect(errBuf.String()).To(ContainSubstring("Warning:"))
			})
		})

		Describe("happy path via PromptForDescription", func() {
			It("returns edited description when both gum steps succeed", func() {
				runner := func(args ...string) ([]byte, error) {
					if args[0] == "confirm" {
						return nil, nil // user said "yes"
					}
					// gum write returns edited description
					return []byte("My custom description"), nil
				}
				wio, _, errBuf := gumWizardIO("", runner)

				desc, err := profile.PromptForDescription(wio, "Auto description")
				Expect(err).NotTo(HaveOccurred())
				Expect(desc).To(Equal("My custom description"))
				Expect(errBuf.String()).To(BeEmpty())
			})
		})
	})
})
