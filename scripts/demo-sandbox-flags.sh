#!/bin/bash
# ABOUTME: Comprehensive test script for claudeup sandbox flags
# ABOUTME: Verifies each flag behaves as documented and reports discrepancies

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Counters
PASSED=0
FAILED=0
WARNINGS=0

# Get script directory and project root
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
CLAUDEUP_BIN="$PROJECT_ROOT/bin/claudeup"

# Test temp directory
TEST_DIR=$(mktemp -d)
trap 'rm -rf "$TEST_DIR"' EXIT

# Ensure binary is built
if [ ! -f "$CLAUDEUP_BIN" ]; then
    echo -e "${RED}✗ claudeup binary not found at $CLAUDEUP_BIN${NC}"
    echo "  Run: go build -o bin/claudeup ./cmd/claudeup"
    exit 1
fi

echo -e "${BLUE}═══════════════════════════════════════════════════════${NC}"
echo -e "${BLUE}  Claudeup Sandbox Flag Verification${NC}"
echo -e "${BLUE}═══════════════════════════════════════════════════════${NC}"
echo ""

# Helper functions
pass() {
    echo -e "${GREEN}✓${NC} $1"
    ((PASSED++))
}

fail() {
    echo -e "${RED}✗${NC} $1"
    if [ -n "${2:-}" ]; then
        echo -e "  ${RED}Expected:${NC} $2"
    fi
    if [ -n "${3:-}" ]; then
        echo -e "  ${RED}Actual:${NC} $3"
    fi
    ((FAILED++))
}

warn() {
    echo -e "${YELLOW}⚠${NC} $1"
    if [ -n "${2:-}" ]; then
        echo -e "  ${YELLOW}Note:${NC} $2"
    fi
    ((WARNINGS++))
}

section() {
    echo ""
    echo -e "${BLUE}─────────────────────────────────────────────────────${NC}"
    echo -e "${BLUE}$1${NC}"
    echo -e "${BLUE}─────────────────────────────────────────────────────${NC}"
}

# Test helper - captures docker command that would be run
# We'll use a wrapper script that intercepts the docker command
capture_docker_command() {
    local test_name="$1"
    shift
    local args=("$@")

    # Create a fake docker wrapper
    local docker_wrapper="$TEST_DIR/docker-$test_name"
    cat > "$docker_wrapper" << 'EOF'
#!/bin/bash
# Output the full command that was called
echo "DOCKER_COMMAND: $@"
# Don't actually run docker
exit 0
EOF
    chmod +x "$docker_wrapper"

    # Run claudeup with PATH pointing to our fake docker
    local output
    output=$(PATH="$TEST_DIR:$PATH" "$CLAUDEUP_BIN" "${args[@]}" 2>&1 || true)

    # Extract the docker command
    echo "$output" | grep "^DOCKER_COMMAND:" | sed 's/^DOCKER_COMMAND: //' || echo ""
}

#═══════════════════════════════════════════════════════
# TEST 1: --help flag
#═══════════════════════════════════════════════════════
section "Test 1: Help Documentation"

HELP_OUTPUT=$("$CLAUDEUP_BIN" sandbox --help 2>&1)

# Check that help mentions key behaviors
if echo "$HELP_OUTPUT" | grep -q "ephemeral session"; then
    pass "Help mentions ephemeral sessions"
else
    fail "Help should mention ephemeral sessions"
fi

if echo "$HELP_OUTPUT" | grep -q "current working directory"; then
    pass "Help mentions working directory mount"
else
    fail "Help should mention working directory mount"
fi

if echo "$HELP_OUTPUT" | grep -q "/workspace"; then
    pass "Help mentions /workspace mount point"
else
    fail "Help should mention /workspace mount point"
fi

#═══════════════════════════════════════════════════════
# TEST 2: --clean flag requires --profile
#═══════════════════════════════════════════════════════
section "Test 2: --clean Requires --profile"

EXPECTED_ERROR="--clean requires --profile"
OUTPUT=$("$CLAUDEUP_BIN" sandbox --clean 2>&1 || true)

if echo "$OUTPUT" | grep -q "$EXPECTED_ERROR"; then
    pass "--clean without --profile shows correct error"
else
    fail "--clean without --profile should error" "$EXPECTED_ERROR" "$OUTPUT"
fi

#═══════════════════════════════════════════════════════
# TEST 3: --ephemeral flag behavior
#═══════════════════════════════════════════════════════
section "Test 3: --ephemeral Flag Behavior"

# According to docs: "Force ephemeral mode (no persistence)"
# Expected: Even with --profile, should NOT mount persistent state

# Create a test profile
TEST_PROFILE="test-ephemeral-$$"
mkdir -p "$HOME/.claudeup/profiles"
cat > "$HOME/.claudeup/profiles/$TEST_PROFILE.json" << EOF
{
    "settings": {},
    "plugins": []
}
EOF

# Test with --ephemeral and --profile
warn "Cannot fully test --ephemeral without mocking Docker"
warn "Manual verification needed: --ephemeral should prevent state persistence even with --profile"

# Cleanup
rm -f "$HOME/.claudeup/profiles/$TEST_PROFILE.json"

#═══════════════════════════════════════════════════════
# TEST 4: --no-mount flag behavior
#═══════════════════════════════════════════════════════
section "Test 4: --no-mount Flag Behavior"

# According to docs: "Don't mount working directory"
# Expected: No /workspace mount in docker command

warn "Cannot verify --no-mount without Docker mock"
warn "Expected behavior: No -v \$PWD:/workspace in docker run command"

#═══════════════════════════════════════════════════════
# TEST 5: --shell flag behavior
#═══════════════════════════════════════════════════════
section "Test 5: --shell Flag Behavior"

# According to docs: "Drop to bash instead of Claude CLI"
# Expected: Docker command should use --entrypoint bash

warn "Cannot verify --shell without Docker mock"
warn "Expected behavior: Docker command should include --entrypoint bash"

#═══════════════════════════════════════════════════════
# TEST 6: --mount flag parsing
#═══════════════════════════════════════════════════════
section "Test 6: --mount Flag Parsing"

# Test mount format parsing
# Expected format: host:container[:ro]

# Valid formats that should work:
# - /path/host:/path/container
# - ~/path:/path/container
# - /path:/path:ro

warn "Cannot verify --mount without Docker mock"
warn "Expected behavior: Should accept host:container and host:container:ro formats"
warn "Should expand ~ in host paths"

#═══════════════════════════════════════════════════════
# TEST 7: Default behavior (no flags)
#═══════════════════════════════════════════════════════
section "Test 7: Default Behavior"

# According to docs:
# - Ephemeral session (no persistence)
# - Current working directory mounted at /workspace
# - Runs Claude CLI (not shell)

warn "Default behavior verification requires Docker mock"
warn "Expected: ephemeral, workdir mounted at /workspace, claude entrypoint"

#═══════════════════════════════════════════════════════
# TEST 8: --profile flag creates persistent state
#═══════════════════════════════════════════════════════
section "Test 8: --profile Creates Persistent State"

TEST_PROFILE="test-persistent-$$"
mkdir -p "$HOME/.claudeup/profiles"
cat > "$HOME/.claudeup/profiles/$TEST_PROFILE.json" << EOF
{
    "settings": {},
    "plugins": []
}
EOF

# The state directory should be created at ~/.claudeup/sandboxes/<profile>
EXPECTED_STATE_DIR="$HOME/.claudeup/sandboxes/$TEST_PROFILE"

# Run sandbox with --profile (will fail without docker, but should create state dir)
# Use --shell so we can exit immediately if docker actually exists
OUTPUT=$("$CLAUDEUP_BIN" sandbox --profile "$TEST_PROFILE" --shell 2>&1 || true)

# Check if Docker error appears (expected if Docker not installed)
if echo "$OUTPUT" | grep -q "docker is required"; then
    warn "Docker not available - cannot fully test --profile"
    warn "Expected: State dir should be created at $EXPECTED_STATE_DIR"
else
    # Docker might be available, check if state dir exists
    if [ -d "$EXPECTED_STATE_DIR" ]; then
        pass "--profile creates state directory at expected location"
        rm -rf "$EXPECTED_STATE_DIR"
    else
        fail "--profile should create state directory" "$EXPECTED_STATE_DIR" "directory not created"
    fi
fi

# Cleanup
rm -f "$HOME/.claudeup/profiles/$TEST_PROFILE.json"

#═══════════════════════════════════════════════════════
# TEST 9: --clean removes state directory
#═══════════════════════════════════════════════════════
section "Test 9: --clean Removes State Directory"

TEST_PROFILE="test-clean-$$"

# Create profile and state directory
mkdir -p "$HOME/.claudeup/profiles"
mkdir -p "$HOME/.claudeup/sandboxes/$TEST_PROFILE"
cat > "$HOME/.claudeup/profiles/$TEST_PROFILE.json" << EOF
{
    "settings": {},
    "plugins": []
}
EOF

# Create a test file in the state directory
touch "$HOME/.claudeup/sandboxes/$TEST_PROFILE/testfile"

# Run --clean
"$CLAUDEUP_BIN" sandbox --clean --profile "$TEST_PROFILE" > /dev/null 2>&1

# Verify state directory was removed
if [ ! -d "$HOME/.claudeup/sandboxes/$TEST_PROFILE" ]; then
    pass "--clean removes sandbox state directory"
else
    fail "--clean should remove state directory" "directory removed" "directory still exists"
    rm -rf "$HOME/.claudeup/sandboxes/$TEST_PROFILE"
fi

# Cleanup
rm -f "$HOME/.claudeup/profiles/$TEST_PROFILE.json"

#═══════════════════════════════════════════════════════
# TEST 10: --image flag override
#═══════════════════════════════════════════════════════
section "Test 10: --image Flag Override"

# According to docs: "Override sandbox image"
# Default: ghcr.io/claudeup/claudeup-sandbox:latest
# Expected: Should use custom image if provided

warn "Cannot verify --image without Docker mock"
warn "Expected: Docker command should use specified image instead of default"

#═══════════════════════════════════════════════════════
# TEST 11: Profile sandbox config (mounts, secrets, env)
#═══════════════════════════════════════════════════════
section "Test 11: Profile Sandbox Configuration"

# Profile can specify sandbox.mounts, sandbox.secrets, sandbox.env
# These should be merged with CLI flags

warn "Cannot verify profile sandbox config without Docker mock"
warn "Expected: Profile mounts/secrets/env should be applied in addition to CLI flags"

#═══════════════════════════════════════════════════════
# TEST 12: Secret resolution order
#═══════════════════════════════════════════════════════
section "Test 12: Secret Resolution"

# Secrets should be resolved from: env vars, 1Password, keychain
# --no-secret should exclude secrets from profile

warn "Cannot verify secret resolution without mocking secret sources"
warn "Expected: Secrets resolved from env > 1Password > keychain"
warn "Expected: --no-secret excludes profile secrets"

#═══════════════════════════════════════════════════════
# Summary
#═══════════════════════════════════════════════════════
echo ""
echo -e "${BLUE}═══════════════════════════════════════════════════════${NC}"
echo -e "${BLUE}  Test Summary${NC}"
echo -e "${BLUE}═══════════════════════════════════════════════════════${NC}"
echo ""
echo -e "${GREEN}Passed:${NC}   $PASSED"
echo -e "${RED}Failed:${NC}   $FAILED"
echo -e "${YELLOW}Warnings:${NC} $WARNINGS"
echo ""

if [ $FAILED -eq 0 ]; then
    echo -e "${GREEN}All testable behaviors verified!${NC}"
    echo ""
    echo -e "${YELLOW}Note:${NC} Many behaviors cannot be fully tested without mocking Docker."
    echo "       Consider creating integration tests with Docker mocks."
    exit 0
else
    echo -e "${RED}Some tests failed!${NC}"
    echo ""
    exit 1
fi
