#!/bin/bash
# ABOUTME: Interactive test script for verifying sandbox flag behavior
# ABOUTME: Guides user through manual verification of each flag

set -euo pipefail

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m'

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
CLAUDEUP_BIN="$PROJECT_ROOT/bin/claudeup"

# Ensure binary exists
if [ ! -f "$CLAUDEUP_BIN" ]; then
    echo "Error: claudeup binary not found"
    echo "Run: go build -o bin/claudeup ./cmd/claudeup"
    exit 1
fi

# Check for API key (required for Claude to run non-interactively)
if [ -z "${ANTHROPIC_API_KEY:-}" ]; then
    echo -e "${YELLOW}Warning: ANTHROPIC_API_KEY not set${NC}"
    echo ""
    echo "Claude requires an API key to run non-interactively."
    echo "Run: source ./scripts/setup-test-env.sh"
    echo ""
    read -p "Continue anyway? (y/n) " response
    if [[ ! "$response" =~ ^[Yy]$ ]]; then
        exit 0
    fi
else
    echo -e "${GREEN}✓ ANTHROPIC_API_KEY is set${NC}"
fi

echo -e "${BLUE}═══════════════════════════════════════════════════════${NC}"
echo -e "${BLUE}  Interactive Sandbox Flag Verification${NC}"
echo -e "${BLUE}═══════════════════════════════════════════════════════${NC}"
echo ""
echo "This script will guide you through testing sandbox flags."
echo ""
if [ -n "${ANTHROPIC_API_KEY:-}" ]; then
    echo -e "${GREEN}Setup:${NC} API key configured (last 4: ***${ANTHROPIC_API_KEY: -4})"
else
    echo -e "${YELLOW}Setup:${NC} No API key - Docker tests will fail"
    echo "       Run: source ./scripts/setup-test-env.sh"
fi
echo ""
echo "Press ENTER after each test to continue..."
echo ""

# Test 1: --clean requires --profile
echo -e "${BLUE}Test 1: --clean requires --profile${NC}"
echo ""
echo "Running: claudeup sandbox --clean"
echo "Expected: Error message '--clean requires --profile'"
echo ""
read -p "Press ENTER to run..."
$CLAUDEUP_BIN sandbox --clean 2>&1 | tail -1 || true
echo ""
read -p "Did you see the error '--clean requires --profile'? (y/n) " response
if [[ "$response" =~ ^[Yy]$ ]]; then
    echo -e "${GREEN}✓ PASS${NC}"
else
    echo -e "${YELLOW}✗ FAIL - Expected error not shown${NC}"
fi
echo ""

# Test 2: --profile creates state directory
echo -e "${BLUE}Test 2: --profile creates state directory${NC}"
echo ""
TEST_PROFILE="test-profile-$$"
echo "Creating test profile: $TEST_PROFILE"
mkdir -p "$HOME/.claudeup/profiles"
cat > "$HOME/.claudeup/profiles/$TEST_PROFILE.json" << 'EOF'
{
  "settings": {},
  "plugins": []
}
EOF

echo ""
echo "State directory should be created at: ~/.claudeup/sandboxes/$TEST_PROFILE"
echo ""
echo "We'll start the sandbox in the background and kill it after 2 seconds."
echo "This is enough time to verify the state directory is created."
echo ""
read -p "Press ENTER to test..."

# Start sandbox in background and kill after 2 seconds
# The state directory is created before docker runs, so this is safe
timeout 2 $CLAUDEUP_BIN sandbox --profile "$TEST_PROFILE" --shell > /dev/null 2>&1 &
SANDBOX_PID=$!

# Wait a moment for state dir creation
sleep 1

# Kill the sandbox if still running
kill $SANDBOX_PID 2>/dev/null || true
wait $SANDBOX_PID 2>/dev/null || true

echo ""
if [ -d "$HOME/.claudeup/sandboxes/$TEST_PROFILE" ]; then
    echo -e "${GREEN}✓ PASS - State directory created at ~/.claudeup/sandboxes/$TEST_PROFILE${NC}"
    rm -rf "$HOME/.claudeup/sandboxes/$TEST_PROFILE"
else
    echo -e "${YELLOW}✗ FAIL - State directory NOT created${NC}"
fi
rm -f "$HOME/.claudeup/profiles/$TEST_PROFILE.json"
echo ""

# Test 3: --clean removes state directory
echo -e "${BLUE}Test 3: --clean removes state directory${NC}"
echo ""
TEST_PROFILE="test-clean-$$"
echo "Creating test profile and state: $TEST_PROFILE"
mkdir -p "$HOME/.claudeup/profiles"
mkdir -p "$HOME/.claudeup/sandboxes/$TEST_PROFILE"
cat > "$HOME/.claudeup/profiles/$TEST_PROFILE.json" << 'EOF'
{
  "settings": {},
  "plugins": []
}
EOF
touch "$HOME/.claudeup/sandboxes/$TEST_PROFILE/testfile"

echo "Created state directory with test file"
echo ""
read -p "Press ENTER to run --clean..."
$CLAUDEUP_BIN sandbox --clean --profile "$TEST_PROFILE"
echo ""

if [ ! -d "$HOME/.claudeup/sandboxes/$TEST_PROFILE" ]; then
    echo -e "${GREEN}✓ PASS - State directory removed${NC}"
else
    echo -e "${YELLOW}✗ FAIL - State directory still exists${NC}"
    rm -rf "$HOME/.claudeup/sandboxes/$TEST_PROFILE"
fi
rm -f "$HOME/.claudeup/profiles/$TEST_PROFILE.json"
echo ""

# Test 4: Help text verification
echo -e "${BLUE}Test 4: Help text accuracy${NC}"
echo ""
echo "Checking help text for key information..."
echo ""

HELP=$($CLAUDEUP_BIN sandbox --help 2>&1)

checks=(
    "ephemeral session:mentions ephemeral mode"
    "current working directory:mentions workdir mount"
    "/workspace:mentions mount point"
    "--profile:has profile flag"
    "--shell:has shell flag"
    "--mount:has mount flag"
    "--no-mount:has no-mount flag"
    "--secret:has secret flag"
    "--no-secret:has no-secret flag"
    "--clean:has clean flag"
    "--image:has image flag"
    "--ephemeral:has ephemeral flag"
)

for check in "${checks[@]}"; do
    pattern="${check%%:*}"
    desc="${check##*:}"
    # Use -F to treat pattern as fixed string (not regex) and -- to prevent pattern being interpreted as grep option
    if echo "$HELP" | grep -qiF -- "$pattern"; then
        echo -e "  ${GREEN}✓${NC} $desc"
    else
        echo -e "  ${YELLOW}✗${NC} $desc - missing '$pattern'"
    fi
done

echo ""
echo -e "${BLUE}═══════════════════════════════════════════════════════${NC}"
echo -e "${BLUE}  Tests requiring Docker${NC}"
echo -e "${BLUE}═══════════════════════════════════════════════════════${NC}"
echo ""
echo "The following tests require Docker to be installed and running."
echo "They will guide you through interactive verification."
echo ""

# Check Docker availability
if ! docker info > /dev/null 2>&1; then
    echo -e "${YELLOW}Docker is not available or not running.${NC}"
    echo ""
    echo "The following tests require Docker:"
    echo "  - Default behavior (ephemeral, workdir mount)"
    echo "  - --profile persistence"
    echo "  - --ephemeral override"
    echo "  - --shell entrypoint"
    echo "  - --mount additional mounts"
    echo "  - --no-mount behavior"
    echo "  - --secret injection"
    echo "  - --image override"
    echo ""
    echo "See docs/sandbox-flag-verification.md for manual test procedures."
    exit 0
fi

echo -e "${GREEN}Docker is available!${NC}"
echo ""
echo "See docs/sandbox-flag-verification.md for detailed manual test procedures"
echo "for Docker-based tests."
echo ""
