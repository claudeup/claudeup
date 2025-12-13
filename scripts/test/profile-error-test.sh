#!/bin/bash
# Tests error conditions

echo "=== Profile Error Tests ==="

CONFIG_FILE=~/.claudeup/config.json

# Cleanup any leftover test profiles and backup config
rm -f ~/.claudeup/profiles/test-err.json 2>/dev/null
[ -f "$CONFIG_FILE" ] && cp "$CONFIG_FILE" "${CONFIG_FILE}.bak"

# Test 1: create -y with no active profile
echo "[1] Testing: create -y with no active profile errors"
# Clear active profile by editing config file
if [ -f "$CONFIG_FILE" ]; then
  cat "$CONFIG_FILE" | sed 's/"activeProfile": "[^"]*"/"activeProfile": ""/g' > /tmp/claudeup-config-tmp.json
  mv /tmp/claudeup-config-tmp.json "$CONFIG_FILE"
fi

if claudeup profile create test-err -y 2>&1 | grep -qi "no active profile"; then
  echo "✓ Pass"
else
  echo "✗ Fail - should have errored about no active profile"
  # Restore config
  [ -f "${CONFIG_FILE}.bak" ] && mv "${CONFIG_FILE}.bak" "$CONFIG_FILE"
  exit 1
fi

# Restore original config for next test
[ -f "${CONFIG_FILE}.bak" ] && cp "${CONFIG_FILE}.bak" "$CONFIG_FILE"

# Test 2: create --from nonexistent profile
echo "[2] Testing: create --from nonexistent profile errors"
rm -f ~/.claudeup/profiles/test-err.json 2>/dev/null
if claudeup profile create test-err --from nonexistent-xyz 2>&1 | grep -q "not found"; then
  echo "✓ Pass"
else
  echo "✗ Fail - should have errored about profile not found"
  [ -f "${CONFIG_FILE}.bak" ] && mv "${CONFIG_FILE}.bak" "$CONFIG_FILE"
  exit 1
fi

# Final cleanup
[ -f "${CONFIG_FILE}.bak" ] && mv "${CONFIG_FILE}.bak" "$CONFIG_FILE"
rm -f ~/.claudeup/profiles/test-err.json 2>/dev/null

echo "=== All error tests passed ==="
