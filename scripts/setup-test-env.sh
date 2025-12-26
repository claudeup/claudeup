#!/bin/bash
# ABOUTME: Sets up environment for sandbox testing
# ABOUTME: Extracts API key from macOS Keychain and exports required variables

set -euo pipefail

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo "Setting up sandbox test environment..."
echo ""

# Try to extract API key from macOS Keychain
# Claude Code stores the API key in Keychain with service name "Claude"
echo "Attempting to extract ANTHROPIC_API_KEY from macOS Keychain..."

# Try multiple possible Keychain entries
API_KEY=""

# Try common service names Claude might use
for SERVICE in "Claude" "claude" "Anthropic" "anthropic" "Claude Code" "claude-code"; do
    for ACCOUNT in "api_key" "apikey" "ANTHROPIC_API_KEY" "anthropic_api_key"; do
        KEY=$(security find-generic-password -s "$SERVICE" -a "$ACCOUNT" -w 2>/dev/null || echo "")
        if [ -n "$KEY" ]; then
            API_KEY="$KEY"
            echo -e "${GREEN}✓ Found API key in Keychain (service: $SERVICE, account: $ACCOUNT)${NC}"
            break 2
        fi
    done
done

# If not found in Keychain, check if already in environment
if [ -z "$API_KEY" ] && [ -n "${ANTHROPIC_API_KEY:-}" ]; then
    API_KEY="$ANTHROPIC_API_KEY"
    echo -e "${GREEN}✓ Using ANTHROPIC_API_KEY from environment${NC}"
fi

# If still not found, prompt user
if [ -z "$API_KEY" ]; then
    echo -e "${YELLOW}⚠ Could not find API key in Keychain or environment${NC}"
    echo ""
    echo "Please obtain your API key from:"
    echo "  https://console.anthropic.com/settings/keys"
    echo ""
    echo "Then either:"
    echo "  1. Export it: export ANTHROPIC_API_KEY='sk-ant-...'"
    echo "  2. Add it to your shell profile (~/.zshrc or ~/.bashrc)"
    echo "  3. Pass it directly to this script"
    echo ""
    read -p "Enter API key (or press ENTER to skip): " MANUAL_KEY
    if [ -n "$MANUAL_KEY" ]; then
        API_KEY="$MANUAL_KEY"
        echo -e "${GREEN}✓ Using manually provided API key${NC}"
    else
        echo -e "${RED}✗ No API key available - tests will fail${NC}"
        exit 1
    fi
fi

# Validate API key format (should start with sk-ant-)
if [[ ! "$API_KEY" =~ ^sk-ant- ]]; then
    echo -e "${YELLOW}⚠ Warning: API key doesn't start with 'sk-ant-'${NC}"
    echo "  This might not be a valid Anthropic API key"
fi

# Export the API key
export ANTHROPIC_API_KEY="$API_KEY"

echo ""
echo -e "${GREEN}Environment setup complete!${NC}"
echo ""
echo "Exported variables:"
echo "  ANTHROPIC_API_KEY=sk-ant-***${API_KEY: -4}"
echo ""
echo "You can now run:"
echo "  ./bin/claudeup sandbox --secret ANTHROPIC_API_KEY"
echo ""
echo "Or source this script to set the variables in your current shell:"
echo "  source ./scripts/setup-test-env.sh"
echo ""
