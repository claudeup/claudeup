// ABOUTME: TestEnv provides isolated test environments for acceptance tests
// ABOUTME: Creates temp directories and runs CLI binary with environment overrides
package helpers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/claudeup/claudeup/v2/internal/profile"
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
	Expect(os.MkdirAll(filepath.Join(env.ClaudeDir, "plugins"), 0755)).To(Succeed())

	// Create empty marketplace and plugin registries so commands don't fail
	env.CreateKnownMarketplaces(map[string]interface{}{})
	env.CreateInstalledPlugins(map[string]interface{}{})
	env.CreateSettings(map[string]bool{})

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
		"CLAUDEUP_HOME="+e.ClaudeupDir,
		"CLAUDE_CONFIG_DIR="+e.ClaudeDir,
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

// GetActiveProfile returns the active profile name from config
func (e *TestEnv) GetActiveProfile() string {
	data, err := os.ReadFile(e.ConfigFile)
	if err != nil {
		return ""
	}

	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		return ""
	}

	if prefs, ok := config["preferences"].(map[string]interface{}); ok {
		if activeProfile, ok := prefs["activeProfile"].(string); ok {
			return activeProfile
		}
	}

	return ""
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

// RunWithEnv executes the CLI with additional environment variables
func (e *TestEnv) RunWithEnv(extraEnv map[string]string, args ...string) *Result {
	return e.RunWithEnvAndInput(extraEnv, "", args...)
}

// RunWithEnvAndInput executes the CLI with additional env vars and stdin input
func (e *TestEnv) RunWithEnvAndInput(extraEnv map[string]string, input string, args ...string) *Result {
	cmd := exec.Command(e.Binary, args...)
	cmd.Env = append(os.Environ(),
		"CLAUDEUP_HOME="+e.ClaudeupDir,
		"CLAUDE_CONFIG_DIR="+e.ClaudeDir,
	)

	for k, v := range extraEnv {
		cmd.Env = append(cmd.Env, k+"="+v)
	}

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

// CreateInstalledPlugins creates a fake installed_plugins.json and settings.json
func (e *TestEnv) CreateInstalledPlugins(plugins map[string]interface{}) {
	pluginsDir := filepath.Join(e.ClaudeDir, "plugins")
	Expect(os.MkdirAll(pluginsDir, 0755)).To(Succeed())

	// Create installed_plugins.json
	data := map[string]interface{}{
		"version": 2,
		"plugins": plugins,
	}
	jsonData, err := json.MarshalIndent(data, "", "  ")
	Expect(err).NotTo(HaveOccurred())
	Expect(os.WriteFile(filepath.Join(pluginsDir, "installed_plugins.json"), jsonData, 0644)).To(Succeed())

	// Create settings.json with all plugins enabled by default
	enabledPlugins := make(map[string]bool)
	for pluginName := range plugins {
		enabledPlugins[pluginName] = true
	}
	e.CreateSettings(enabledPlugins)
}

// CreateKnownMarketplaces creates a fake known_marketplaces.json
func (e *TestEnv) CreateKnownMarketplaces(marketplaces map[string]interface{}) {
	pluginsDir := filepath.Join(e.ClaudeDir, "plugins")
	Expect(os.MkdirAll(pluginsDir, 0755)).To(Succeed())

	jsonData, err := json.MarshalIndent(marketplaces, "", "  ")
	Expect(err).NotTo(HaveOccurred())
	Expect(os.WriteFile(filepath.Join(pluginsDir, "known_marketplaces.json"), jsonData, 0644)).To(Succeed())
}

// CreateMarketplaceIndex creates a fake .claude-plugin/marketplace.json for a marketplace
func (e *TestEnv) CreateMarketplaceIndex(installLocation string, name string, plugins []map[string]string) {
	indexDir := filepath.Join(installLocation, ".claude-plugin")
	Expect(os.MkdirAll(indexDir, 0755)).To(Succeed())

	index := map[string]interface{}{
		"name":    name,
		"plugins": plugins,
	}
	jsonData, err := json.MarshalIndent(index, "", "  ")
	Expect(err).NotTo(HaveOccurred())
	Expect(os.WriteFile(filepath.Join(indexDir, "marketplace.json"), jsonData, 0644)).To(Succeed())
}

// CreateSettings creates a fake settings.json with enabled plugins
func (e *TestEnv) CreateSettings(enabledPlugins map[string]bool) {
	settings := map[string]interface{}{
		"enabledPlugins": enabledPlugins,
	}
	jsonData, err := json.MarshalIndent(settings, "", "  ")
	Expect(err).NotTo(HaveOccurred())
	Expect(os.WriteFile(filepath.Join(e.ClaudeDir, "settings.json"), jsonData, 0644)).To(Succeed())
}

// IsPluginEnabled checks if a plugin is enabled in settings.json
func (e *TestEnv) IsPluginEnabled(pluginName string) bool {
	data, err := os.ReadFile(filepath.Join(e.ClaudeDir, "settings.json"))
	if err != nil {
		return false
	}

	var settings map[string]interface{}
	if err := json.Unmarshal(data, &settings); err != nil {
		return false
	}

	enabledPlugins, ok := settings["enabledPlugins"].(map[string]interface{})
	if !ok {
		return false
	}

	enabled, ok := enabledPlugins[pluginName].(bool)
	return ok && enabled
}

// CreatePluginMCPServers creates a .mcp.json file in a plugin directory
func (e *TestEnv) CreatePluginMCPServers(pluginPath string, servers map[string]interface{}) {
	mcpFile := map[string]interface{}{
		"mcpServers": servers,
	}
	jsonData, err := json.MarshalIndent(mcpFile, "", "  ")
	Expect(err).NotTo(HaveOccurred())
	Expect(os.WriteFile(filepath.Join(pluginPath, ".mcp.json"), jsonData, 0644)).To(Succeed())
}

// SetDisabledMCPServers configures disabled MCP servers in claudeup config
func (e *TestEnv) SetDisabledMCPServers(servers []string) {
	config := map[string]interface{}{
		"disabledMcpServers": servers,
		"preferences":        map[string]interface{}{},
	}
	jsonData, err := json.MarshalIndent(config, "", "  ")
	Expect(err).NotTo(HaveOccurred())
	Expect(os.WriteFile(e.ConfigFile, jsonData, 0644)).To(Succeed())
}

// ProjectDir creates and returns a project directory for testing scoped profiles
func (e *TestEnv) ProjectDir(name string) string {
	projectDir := filepath.Join(e.TempDir, "projects", name)
	Expect(os.MkdirAll(projectDir, 0755)).To(Succeed())
	return projectDir
}

// MCPJSONExists checks if .mcp.json exists in the given directory
func (e *TestEnv) MCPJSONExists(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, ".mcp.json"))
	return err == nil
}

// ClaudeupJSONExists checks if .claudeup.json exists in the given directory
func (e *TestEnv) ClaudeupJSONExists(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, ".claudeup.json"))
	return err == nil
}

// LoadMCPJSON loads .mcp.json from the given directory
func (e *TestEnv) LoadMCPJSON(dir string) map[string]interface{} {
	data, err := os.ReadFile(filepath.Join(dir, ".mcp.json"))
	Expect(err).NotTo(HaveOccurred())

	var config map[string]interface{}
	Expect(json.Unmarshal(data, &config)).To(Succeed())
	return config
}

// LoadClaudeupJSON loads .claudeup.json from the given directory
func (e *TestEnv) LoadClaudeupJSON(dir string) map[string]interface{} {
	data, err := os.ReadFile(filepath.Join(dir, ".claudeup.json"))
	Expect(err).NotTo(HaveOccurred())

	var config map[string]interface{}
	Expect(json.Unmarshal(data, &config)).To(Succeed())
	return config
}

// CreateClaudeupJSON creates a .claudeup.json file in the given directory
func (e *TestEnv) CreateClaudeupJSON(dir string, cfg map[string]interface{}) {
	data, err := json.MarshalIndent(cfg, "", "  ")
	Expect(err).NotTo(HaveOccurred())
	Expect(os.WriteFile(filepath.Join(dir, ".claudeup.json"), data, 0644)).To(Succeed())
}

// WriteFile writes arbitrary content to a file in the given directory
func (e *TestEnv) WriteFile(dir, filename, content string) {
	Expect(os.WriteFile(filepath.Join(dir, filename), []byte(content), 0644)).To(Succeed())
}

// LoadProjectsRegistry loads the projects.json registry
func (e *TestEnv) LoadProjectsRegistry() map[string]interface{} {
	path := filepath.Join(e.ClaudeupDir, "projects.json")
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil
	}
	Expect(err).NotTo(HaveOccurred())

	var registry map[string]interface{}
	Expect(json.Unmarshal(data, &registry)).To(Succeed())
	return registry
}

// RegisterProject adds a project to the projects.json registry (local scope)
func (e *TestEnv) RegisterProject(projectDir, profileName string) {
	path := filepath.Join(e.ClaudeupDir, "projects.json")

	// Normalize path to handle macOS /var -> /private/var symlinks
	// This matches what os.Getwd() returns when CLI runs from the directory
	normalizedDir := projectDir
	if resolved, err := filepath.EvalSymlinks(projectDir); err == nil {
		normalizedDir = resolved
	}

	registry := map[string]interface{}{
		"version": "1",
		"projects": map[string]interface{}{
			normalizedDir: map[string]interface{}{
				"profile":   profileName,
				"appliedAt": "2025-01-01T00:00:00Z",
			},
		},
	}

	data, err := json.MarshalIndent(registry, "", "  ")
	Expect(err).NotTo(HaveOccurred())
	Expect(os.WriteFile(path, data, 0644)).To(Succeed())
}

// RunInDir executes the CLI with a specific working directory
func (e *TestEnv) RunInDir(dir string, args ...string) *Result {
	cmd := exec.Command(e.Binary, args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"CLAUDEUP_HOME="+e.ClaudeupDir,
		"CLAUDE_CONFIG_DIR="+e.ClaudeDir,
	)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

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

// RunInDirWithInput executes the CLI with a specific working directory and stdin input
func (e *TestEnv) RunInDirWithInput(dir, input string, args ...string) *Result {
	cmd := exec.Command(e.Binary, args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"CLAUDEUP_HOME="+e.ClaudeupDir,
		"CLAUDE_CONFIG_DIR="+e.ClaudeDir,
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

// BuildBinary builds the claudeup binary and returns its path
func BuildBinary() string {
	binPath := filepath.Join(GinkgoT().TempDir(), "claudeup")

	// Find the project root by looking for go.mod
	projectRoot, err := findProjectRoot()
	Expect(err).NotTo(HaveOccurred())

	// Use absolute path for source
	sourcePath := filepath.Join(projectRoot, "cmd", "claudeup")

	cmd := exec.Command("go", "build", "-o", binPath, sourcePath)
	Expect(cmd.Run()).To(Succeed())
	return binPath
}

// findProjectRoot walks up the directory tree to find go.mod
func findProjectRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		goModPath := filepath.Join(dir, "go.mod")
		if _, err := os.Stat(goModPath); err == nil {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("could not find go.mod in any parent directory")
		}
		dir = parent
	}
}

// WriteJSON writes data as JSON to the specified path
func WriteJSON(path string, data interface{}) {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	Expect(err).NotTo(HaveOccurred())
	Expect(os.MkdirAll(filepath.Dir(path), 0755)).To(Succeed())
	Expect(os.WriteFile(path, jsonData, 0644)).To(Succeed())
}

// LoadJSON reads and parses a JSON file
func LoadJSON(path string) map[string]interface{} {
	data, err := os.ReadFile(path)
	Expect(err).NotTo(HaveOccurred())

	var result map[string]interface{}
	Expect(json.Unmarshal(data, &result)).To(Succeed())
	return result
}

// Cleanup removes the test environment (automatically called by GinkgoT().TempDir())
func (e *TestEnv) Cleanup() {
	// Temp dir is automatically cleaned up by Ginkgo
}
