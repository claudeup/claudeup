// ABOUTME: Functions for reading Claude Code settings.json configuration
// ABOUTME: Provides access to enabled plugins and other user settings
package claude

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/claudeup/claudeup/v4/internal/events"
)

// canonicalKeyOrder defines the key order that Claude Code uses in settings.json.
// Keys are output in this order, with any unknown keys appended alphabetically at the end.
// This ordering is based on the official schema at https://www.schemastore.org/claude-code-settings.json
var canonicalKeyOrder = []string{
	"$schema",
	"apiKeyHelper",
	"autoUpdatesChannel",
	"awsCredentialExport",
	"awsAuthRefresh",
	"cleanupPeriodDays",
	"env",
	"attribution",
	"includeCoAuthoredBy",
	"plansDirectory",
	"respectGitignore",
	"permissions",
	"language",
	"model",
	"enableAllProjectMcpServers",
	"enabledMcpjsonServers",
	"disabledMcpjsonServers",
	"allowedMcpServers",
	"deniedMcpServers",
	"hooks",
	"disableAllHooks",
	"allowManagedHooksOnly",
	"statusLine",
	"fileSuggestion",
	"enabledPlugins",
	"extraKnownMarketplaces",
	"strictKnownMarketplaces",
	"skippedMarketplaces",
	"skippedPlugins",
	"forceLoginMethod",
	"forceLoginOrgUUID",
	"otelHeadersHelper",
	"outputStyle",
	"skipWebFetchPreflight",
	"sandbox",
	"spinnerTipsEnabled",
	"terminalProgressBarEnabled",
	"showTurnDuration",
	"alwaysThinkingEnabled",
	"companyAnnouncements",
	"pluginConfigs",
}

// nestedKeyOrders defines canonical key ordering for nested objects
var nestedKeyOrders = map[string][]string{
	"permissions": {
		"allow",
		"deny",
		"ask",
		"defaultMode",
		"disableBypassPermissionsMode",
		"additionalDirectories",
	},
	"hooks": {
		"PreToolUse",
		"PostToolUse",
		"PostToolUseFailure",
		"PermissionRequest",
		"Notification",
		"UserPromptSubmit",
		"Stop",
		"SubagentStart",
		"SubagentStop",
		"PreCompact",
		"Setup",
		"SessionStart",
		"SessionEnd",
	},
	"hookMatcher": {
		"matcher",
		"hooks",
	},
	"hookCommand": {
		"type",
		"command",
		"prompt",
		"timeout",
	},
	"sandbox": {
		"network",
		"ignoreViolations",
		"excludedCommands",
		"autoAllowBashIfSandboxed",
		"enableWeakerNestedSandbox",
		"allowUnsandboxedCommands",
		"enabled",
	},
	"attribution": {
		"commit",
		"pr",
	},
	"statusLine": {
		"type",
		"command",
	},
}

// marshalJSON marshals a value to JSON without HTML escaping.
// Go's json.Marshal escapes <, >, & for HTML safety, but settings values
// contain shell commands where these characters must be preserved literally.
func marshalJSON(value any) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(value); err != nil {
		return nil, err
	}
	// Encode appends a trailing newline; trim it
	return bytes.TrimRight(buf.Bytes(), "\n"), nil
}

// marshalCanonical marshals a map to JSON with keys in Claude Code's canonical order.
// Known keys appear in the order defined by canonicalKeyOrder, followed by
// any unknown keys in alphabetical order. Nested objects are also ordered canonically.
func marshalCanonical(data map[string]any) ([]byte, error) {
	b, err := marshalCanonicalWithIndent(data, "", canonicalKeyOrder)
	if err != nil {
		return nil, err
	}
	// Ensure trailing newline (POSIX text file convention)
	return append(b, '\n'), nil
}

// marshalCanonicalWithIndent handles recursive canonical marshaling with proper indentation.
// It uses the provided keyOrder for the current level, and looks up nested key orders
// based on the parent key context.
func marshalCanonicalWithIndent(data map[string]any, indent string, keyOrder []string) ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteString("{\n")
	nextIndent := indent + "  "

	// Build set of canonical keys for quick lookup
	canonicalSet := make(map[string]bool)
	for _, key := range keyOrder {
		canonicalSet[key] = true
	}

	// Collect unknown keys
	var unknownKeys []string
	for key := range data {
		if !canonicalSet[key] {
			unknownKeys = append(unknownKeys, key)
		}
	}
	sort.Strings(unknownKeys)

	// Build ordered keys: canonical first, then unknown (alphabetically)
	var orderedKeys []string
	for _, key := range keyOrder {
		if _, exists := data[key]; exists {
			orderedKeys = append(orderedKeys, key)
		}
	}
	orderedKeys = append(orderedKeys, unknownKeys...)

	// Marshal each key-value pair
	for i, key := range orderedKeys {
		value := data[key]

		// Marshal key
		keyJSON, err := marshalJSON(key)
		if err != nil {
			return nil, err
		}

		// Marshal value with canonical ordering if it's a nested object
		valueJSON, err := marshalValueCanonical(value, nextIndent, key)
		if err != nil {
			return nil, err
		}

		buf.WriteString(nextIndent)
		buf.Write(keyJSON)
		buf.WriteString(": ")
		buf.Write(valueJSON)

		if i < len(orderedKeys)-1 {
			buf.WriteString(",")
		}
		buf.WriteString("\n")
	}

	buf.WriteString(indent)
	buf.WriteString("}")
	return buf.Bytes(), nil
}

// marshalValueCanonical marshals a value with canonical key ordering for nested objects.
// The parentKey is used to look up the appropriate nested key order.
func marshalValueCanonical(value any, indent string, parentKey string) ([]byte, error) {
	switch v := value.(type) {
	case map[string]any:
		// Determine the key order for this nested object
		keyOrder := getNestedKeyOrder(parentKey)
		return marshalCanonicalWithIndent(v, indent, keyOrder)

	case map[string]bool:
		// Convert to map[string]any so the canonical marshaler handles it
		m := make(map[string]any, len(v))
		for k, val := range v {
			m[k] = val
		}
		keyOrder := getNestedKeyOrder(parentKey)
		return marshalCanonicalWithIndent(m, indent, keyOrder)

	case []any:
		return marshalArrayCanonical(v, indent, parentKey)

	default:
		// For primitives and other types, use standard marshaling
		return marshalJSON(value)
	}
}

// marshalArrayCanonical marshals an array with canonical ordering for nested objects.
func marshalArrayCanonical(arr []any, indent string, parentKey string) ([]byte, error) {
	if len(arr) == 0 {
		return []byte("[]"), nil
	}

	var buf bytes.Buffer
	buf.WriteString("[\n")
	nextIndent := indent + "  "

	// Determine element key order based on parent context
	elementKeyOrder := getElementKeyOrder(parentKey)

	for i, elem := range arr {
		buf.WriteString(nextIndent)

		var elemJSON []byte
		var err error

		switch e := elem.(type) {
		case map[string]any:
			elemJSON, err = marshalCanonicalWithIndent(e, nextIndent, elementKeyOrder)
		default:
			elemJSON, err = marshalJSON(elem)
		}

		if err != nil {
			return nil, err
		}

		buf.Write(elemJSON)
		if i < len(arr)-1 {
			buf.WriteString(",")
		}
		buf.WriteString("\n")
	}

	buf.WriteString(indent)
	buf.WriteString("]")
	return buf.Bytes(), nil
}

// getNestedKeyOrder returns the canonical key order for a nested object.
func getNestedKeyOrder(parentKey string) []string {
	if order, ok := nestedKeyOrders[parentKey]; ok {
		return order
	}
	// For unknown nested objects, return nil (caller will fall back to alphabetical order)
	return nil
}

// getElementKeyOrder returns the key order for array elements based on the parent key.
func getElementKeyOrder(parentKey string) []string {
	// Hook event arrays contain hookMatcher objects
	if isHookEventType(parentKey) {
		return nestedKeyOrders["hookMatcher"]
	}
	// The "hooks" array within a hookMatcher contains hookCommand objects
	if parentKey == "hooks" {
		return nestedKeyOrders["hookCommand"]
	}
	return nil
}

// isHookEventType returns true if the key is a hook event type.
func isHookEventType(key string) bool {
	hookEventTypes := nestedKeyOrders["hooks"]
	for _, eventType := range hookEventTypes {
		if key == eventType {
			return true
		}
	}
	return false
}

// Settings represents the Claude Code settings.json file structure
type Settings struct {
	EnabledPlugins map[string]bool        `json:"enabledPlugins"`
	raw            map[string]interface{} // Preserves all fields from settings.json
}

// LoadSettings reads the settings.json file from the Claude directory
func LoadSettings(claudeDir string) (*Settings, error) {
	// Check if Claude directory exists
	if _, err := os.Stat(claudeDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("Claude CLI not found (directory %s does not exist)", claudeDir)
	}

	settingsPath := filepath.Join(claudeDir, "settings.json")
	data, err := os.ReadFile(settingsPath)
	if os.IsNotExist(err) {
		// Claude installed but settings missing - suspicious
		return nil, &PathNotFoundError{
			Component:    "settings",
			ExpectedPath: settingsPath,
			ClaudeDir:    claudeDir,
		}
	}
	if err != nil {
		return nil, err
	}

	// Unmarshal into raw map first to preserve all fields
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}

	// Extract enabledPlugins with type safety
	settings := &Settings{
		raw:            raw,
		EnabledPlugins: make(map[string]bool),
	}

	if enabledPlugins, ok := raw["enabledPlugins"].(map[string]interface{}); ok {
		for key, val := range enabledPlugins {
			if enabled, ok := val.(bool); ok {
				settings.EnabledPlugins[key] = enabled
			}
		}
	}

	// Validate settings structure
	if err := validateSettings(settings); err != nil {
		return nil, err
	}

	return settings, nil
}

// IsPluginEnabled checks if a plugin is enabled in the settings
func (s *Settings) IsPluginEnabled(pluginName string) bool {
	enabled, exists := s.EnabledPlugins[pluginName]
	return exists && enabled
}

// EnablePlugin enables a plugin in the settings
func (s *Settings) EnablePlugin(pluginName string) {
	if s.EnabledPlugins == nil {
		s.EnabledPlugins = make(map[string]bool)
	}
	s.EnabledPlugins[pluginName] = true
}

// DisablePlugin disables a plugin in the settings
func (s *Settings) DisablePlugin(pluginName string) {
	if s.EnabledPlugins == nil {
		return
	}
	s.EnabledPlugins[pluginName] = false
}

// RemovePlugin removes a plugin from the settings entirely
func (s *Settings) RemovePlugin(pluginName string) {
	if s.EnabledPlugins == nil {
		return
	}
	delete(s.EnabledPlugins, pluginName)
}

// SaveSettings writes the settings back to settings.json
func SaveSettings(claudeDir string, settings *Settings) error {
	settingsPath := filepath.Join(claudeDir, "settings.json")

	// Initialize raw map if not present
	if settings.raw == nil {
		settings.raw = make(map[string]interface{})
	}

	// Update enabledPlugins in raw map
	settings.raw["enabledPlugins"] = settings.EnabledPlugins

	// Marshal with Claude Code's canonical key ordering
	data, err := marshalCanonical(settings.raw)
	if err != nil {
		return err
	}

	// Wrap file write with event tracking
	return events.GlobalTracker().RecordFileWrite(
		"settings update",
		settingsPath,
		"user",
		func() error {
			return os.WriteFile(settingsPath, data, 0644)
		},
	)
}

// SettingsPathForScope returns the settings.json path for a given scope
func SettingsPathForScope(scope string, claudeDir string, projectDir string) (string, error) {
	// Validate scope (allow empty string as alias for "user")
	if scope != "" {
		if err := ValidateScope(scope); err != nil {
			return "", err
		}
	}

	switch scope {
	case "user", "":
		return filepath.Join(claudeDir, "settings.json"), nil
	case "project":
		if projectDir == "" {
			return "", fmt.Errorf("project directory required for project scope")
		}
		// Project scope: ./.claude/settings.json
		return filepath.Join(projectDir, ".claude", "settings.json"), nil
	case "local":
		if projectDir == "" {
			return "", fmt.Errorf("project directory required for local scope")
		}
		// Local scope: ./.claude/settings.local.json (machine-specific, gitignored)
		return filepath.Join(projectDir, ".claude", "settings.local.json"), nil
	default:
		// This should never be reached due to ValidateScope above
		return "", fmt.Errorf("invalid scope: %s", scope)
	}
}

// LoadSettingsForScope reads settings from a specific scope
func LoadSettingsForScope(scope string, claudeDir string, projectDir string) (*Settings, error) {
	path, err := SettingsPathForScope(scope, claudeDir, projectDir)
	if err != nil {
		return nil, err
	}

	// If file doesn't exist, return empty settings (not an error)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return &Settings{
			EnabledPlugins: make(map[string]bool),
			raw:            make(map[string]interface{}),
		}, nil
	}

	// Read and parse
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// Unmarshal into raw map first to preserve all fields
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}

	// Extract enabledPlugins with type safety
	settings := &Settings{
		raw:            raw,
		EnabledPlugins: make(map[string]bool),
	}

	if enabledPlugins, ok := raw["enabledPlugins"].(map[string]interface{}); ok {
		for key, val := range enabledPlugins {
			if enabled, ok := val.(bool); ok {
				settings.EnabledPlugins[key] = enabled
			}
		}
	}

	return settings, nil
}

// SaveSettingsForScope writes settings to a specific scope
func SaveSettingsForScope(scope string, claudeDir string, projectDir string, settings *Settings) error {
	path, err := SettingsPathForScope(scope, claudeDir, projectDir)
	if err != nil {
		return err
	}

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// Initialize raw map if not present
	if settings.raw == nil {
		settings.raw = make(map[string]interface{})
	}

	// Update enabledPlugins in raw map
	settings.raw["enabledPlugins"] = settings.EnabledPlugins

	// Marshal with Claude Code's canonical key ordering
	data, err := marshalCanonical(settings.raw)
	if err != nil {
		return err
	}

	// Normalize empty scope to "user"
	normalizedScope := scope
	if normalizedScope == "" {
		normalizedScope = "user"
	}

	// Wrap file write with event tracking
	return events.GlobalTracker().RecordFileWrite(
		"settings update",
		path,
		normalizedScope,
		func() error {
			return os.WriteFile(path, data, 0644)
		},
	)
}

// MergeHooks merges new hooks into settings, deduplicating by command string.
// Hooks are added without removing existing ones.
// newHooks format: map[eventType][]hookEntry where hookEntry has "type" and "command" keys.
func (s *Settings) MergeHooks(newHooks map[string][]map[string]interface{}) error {
	if s.raw == nil {
		s.raw = make(map[string]interface{})
	}

	// Get or create hooks map
	var hooks map[string]interface{}
	if existing, ok := s.raw["hooks"].(map[string]interface{}); ok {
		hooks = existing
	} else {
		hooks = make(map[string]interface{})
		s.raw["hooks"] = hooks
	}

	// For each event type in newHooks
	for eventType, entries := range newHooks {
		// Get existing hooks for this event type
		var existingEntries []interface{}
		if existing, ok := hooks[eventType].([]interface{}); ok {
			existingEntries = existing
		}

		// Collect all existing commands for deduplication
		existingCommands := make(map[string]bool)
		for _, entry := range existingEntries {
			if entryMap, ok := entry.(map[string]interface{}); ok {
				if hooksList, ok := entryMap["hooks"].([]interface{}); ok {
					for _, hook := range hooksList {
						if hookMap, ok := hook.(map[string]interface{}); ok {
							if cmd, ok := hookMap["command"].(string); ok {
								existingCommands[cmd] = true
							}
						}
					}
				}
			}
		}

		// Filter new hooks to only include non-duplicates
		var newHooksList []interface{}
		for _, entry := range entries {
			if cmd, ok := entry["command"].(string); ok {
				if !existingCommands[cmd] {
					newHooksList = append(newHooksList, entry)
					existingCommands[cmd] = true
				}
			}
		}

		if len(newHooksList) > 0 {
			// Add as a new entry with no matcher (applies to all)
			newEntry := map[string]interface{}{
				"hooks": newHooksList,
			}
			existingEntries = append(existingEntries, newEntry)
			hooks[eventType] = existingEntries
		}
	}

	return nil
}
