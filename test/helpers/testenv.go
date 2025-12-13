// ABOUTME: TestEnv provides isolated test environments for acceptance tests
// ABOUTME: Creates temp directories and runs CLI binary with environment overrides
package helpers

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/claudeup/claudeup/internal/profile"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// TestEnv represents an isolated test environment
type TestEnv struct {
	TempDir     string // Root temp directory
	ClaudeDir   string // Fake ~/.claude
	ClaudeupDir string // Fake ~/.claudeup
	ProfilesDir string // Fake ~/.claudeup/profiles
	ConfigFile  string // Fake ~/.claudeup/config.json
	Binary      string // Path to claudeup binary
}

// NewTestEnv creates a new isolated test environment
func NewTestEnv(binary string) *TestEnv {
	tempDir := GinkgoT().TempDir()

	env := &TestEnv{
		TempDir:     tempDir,
		ClaudeDir:   filepath.Join(tempDir, ".claude"),
		ClaudeupDir: filepath.Join(tempDir, ".claudeup"),
		ProfilesDir: filepath.Join(tempDir, ".claudeup", "profiles"),
		ConfigFile:  filepath.Join(tempDir, ".claudeup", "config.json"),
		Binary:      binary,
	}

	// Create directory structure
	Expect(os.MkdirAll(env.ClaudeDir, 0755)).To(Succeed())
	Expect(os.MkdirAll(env.ProfilesDir, 0755)).To(Succeed())

	return env
}

// Run executes the CLI with the given arguments
func (e *TestEnv) Run(args ...string) *Result {
	return e.RunWithInput("", args...)
}

// RunWithInput executes the CLI with stdin input
func (e *TestEnv) RunWithInput(input string, args ...string) *Result {
	cmd := exec.Command(e.Binary, args...)
	cmd.Env = append(os.Environ(),
		"HOME="+e.TempDir,
	)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if input != "" {
		cmd.Stdin = strings.NewReader(input)
	}

	err := cmd.Run()

	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = 1
		}
	}

	return &Result{
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		ExitCode: exitCode,
	}
}

// ProfileExists checks if a profile file exists
func (e *TestEnv) ProfileExists(name string) bool {
	_, err := os.Stat(filepath.Join(e.ProfilesDir, name+".json"))
	return err == nil
}

// CreateProfile creates a profile in the test environment
func (e *TestEnv) CreateProfile(p *profile.Profile) {
	data, err := json.MarshalIndent(p, "", "  ")
	Expect(err).NotTo(HaveOccurred())

	path := filepath.Join(e.ProfilesDir, p.Name+".json")
	Expect(os.WriteFile(path, data, 0644)).To(Succeed())
}

// LoadProfile loads a profile from the test environment
func (e *TestEnv) LoadProfile(name string) *profile.Profile {
	path := filepath.Join(e.ProfilesDir, name+".json")
	data, err := os.ReadFile(path)
	Expect(err).NotTo(HaveOccurred())

	var p profile.Profile
	Expect(json.Unmarshal(data, &p)).To(Succeed())
	return &p
}

// SetActiveProfile sets the active profile in config
func (e *TestEnv) SetActiveProfile(name string) {
	config := map[string]interface{}{
		"preferences": map[string]interface{}{
			"activeProfile": name,
		},
	}
	data, err := json.MarshalIndent(config, "", "  ")
	Expect(err).NotTo(HaveOccurred())
	Expect(os.WriteFile(e.ConfigFile, data, 0644)).To(Succeed())
}

// CreateClaudeSettings creates a fake claude.json settings file
func (e *TestEnv) CreateClaudeSettings() {
	settingsPath := filepath.Join(e.TempDir, ".claude.json")
	settings := map[string]interface{}{
		"mcpServers": map[string]interface{}{},
	}
	data, err := json.MarshalIndent(settings, "", "  ")
	Expect(err).NotTo(HaveOccurred())
	Expect(os.WriteFile(settingsPath, data, 0644)).To(Succeed())
}
