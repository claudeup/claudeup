# Claudeup Test Scripts

Scripts for testing and verifying `claudeup sandbox` functionality.

## Quick Start

```bash
# 1. Build the binary
go build -o bin/claudeup ./cmd/claudeup

# 2. Set up test environment (extracts API key from Keychain)
source ./scripts/setup-test-env.sh

# 3. Run tests
./scripts/demo-sandbox-flags.sh          # Automated tests
./scripts/interactive-sandbox-test.sh    # Interactive tests
```

## Scripts

### `setup-test-env.sh`

Prepares the testing environment by extracting the ANTHROPIC_API_KEY from macOS Keychain.

**Usage:**
```bash
# Source it to export variables to your shell
source ./scripts/setup-test-env.sh

# Or run directly to check if API key is available
./scripts/setup-test-env.sh
```

**What it does:**
- Searches macOS Keychain for Claude's stored API key
- Falls back to `ANTHROPIC_API_KEY` environment variable if set
- Prompts for manual entry if not found
- Validates API key format
- Exports `ANTHROPIC_API_KEY` for use by tests

**Why this is needed:**
Claude Code stores credentials in macOS Keychain. Without the API key injected via environment variable, the `claude` CLI launches in interactive mode and prompts for login, which breaks automated testing.

---

### `demo-sandbox-flags.sh`

Automated test script that verifies sandbox flag behavior without requiring Docker.

**Usage:**
```bash
./scripts/demo-sandbox-flags.sh
```

**Tests:**
- Help text accuracy and completeness
- `--clean` requires `--profile` error handling
- `--clean` removes state directory correctly
- `--profile` creates state directory
- Flag parsing and validation

**Output:**
- ✓ Passed tests (green)
- ✗ Failed tests (red)
- ⚠ Warnings for tests requiring Docker (yellow)

---

### `interactive-sandbox-test.sh`

Interactive guide for manual testing of sandbox flags.

**Usage:**
```bash
./scripts/interactive-sandbox-test.sh
```

**Tests:**
- Automated tests (same as demo script)
- Guides for Docker-dependent tests
- Step-by-step verification with user confirmation
- Checks Docker availability

**Requirements:**
- `ANTHROPIC_API_KEY` must be set (run `setup-test-env.sh` first)
- Docker installed and running (for container tests)

---

### `build-sandbox-image.sh`

Builds the claudeup sandbox Docker image locally for testing.

**Usage:**
```bash
./scripts/build-sandbox-image.sh
```

**What it does:**
- Detects host architecture (amd64/arm64)
- Builds Go binaries for the detected platform
- Creates Docker image using `docker buildx`
- Tags image as `claudeup-sandbox:local`

**Use cases:**
- Testing sandbox image changes before pushing
- Local development of sandbox features
- Testing with custom sandbox configurations

**Testing with local image:**
```bash
# Build local image
./scripts/build-sandbox-image.sh

# Use it
./bin/claudeup sandbox --image claudeup-sandbox:local
```

---

## Testing Workflow

### 1. Initial Setup

```bash
# Build claudeup
go build -o bin/claudeup ./cmd/claudeup

# Set up environment
source ./scripts/setup-test-env.sh
```

### 2. Run Automated Tests

```bash
# Quick verification
./scripts/demo-sandbox-flags.sh
```

### 3. Manual Verification

For comprehensive testing, follow the **Sandbox Flag Verification Matrix** in `docs/sandbox-flag-verification.md`.

Each flag has:
- **Expected behavior** from documentation
- **Exact Docker command** that should be generated
- **Step-by-step verification procedure**
- **Checklist** to track test results

### 4. Testing with Docker

```bash
# Ensure Docker is running
docker info

# Test with default image
./bin/claudeup sandbox --secret ANTHROPIC_API_KEY --shell

# Test with local image
./scripts/build-sandbox-image.sh
./bin/claudeup sandbox --image claudeup-sandbox:local --secret ANTHROPIC_API_KEY --shell
```

---

## Common Test Scenarios

### Test: Default Ephemeral Mode

```bash
./bin/claudeup sandbox --secret ANTHROPIC_API_KEY
# Verify:
# - Claude starts without login prompt
# - Working directory mounted at /workspace
# - No state persists after exit
```

### Test: Persistent Profile

```bash
# Create test profile
mkdir -p ~/.claudeup/profiles
cat > ~/.claudeup/profiles/test.json << 'EOF'
{
  "settings": {},
  "plugins": [],
  "sandbox": {
    "secrets": ["ANTHROPIC_API_KEY"]
  }
}
EOF

# First run
./bin/claudeup sandbox --profile test
# Inside: touch /root/.claude/testfile
# Exit

# Second run
./bin/claudeup sandbox --profile test
# Inside: ls /root/.claude/testfile  # Should exist!
```

### Test: Additional Mounts

```bash
mkdir -p /tmp/test-mount
echo "test" > /tmp/test-mount/file.txt

./bin/claudeup sandbox \
  --secret ANTHROPIC_API_KEY \
  --mount /tmp/test-mount:/data \
  --shell

# Inside: cat /data/file.txt  # Should show "test"
```

### Test: Secret Injection

```bash
export MY_SECRET="test-value"
./bin/claudeup sandbox \
  --secret ANTHROPIC_API_KEY \
  --secret MY_SECRET \
  --shell

# Inside: echo $MY_SECRET  # Should show "test-value"
```

---

## Troubleshooting

### "docker is required" Error

**Problem:** Docker is not installed or not running.

**Solution:**
```bash
# Check Docker status
docker info

# Start Docker Desktop (macOS)
open -a Docker
```

### "ANTHROPIC_API_KEY not set" Warning

**Problem:** API key not found in Keychain or environment.

**Solution:**
```bash
# Extract from Keychain
source ./scripts/setup-test-env.sh

# Or set manually
export ANTHROPIC_API_KEY="sk-ant-..."
```

### Interactive Login Prompt in Container

**Problem:** Claude asks for login credentials inside container.

**Cause:** API key not injected properly.

**Solution:**
```bash
# Always pass API key as secret
./bin/claudeup sandbox --secret ANTHROPIC_API_KEY

# Or add to profile
cat > ~/.claudeup/profiles/myprofile.json << 'EOF'
{
  "sandbox": {
    "secrets": ["ANTHROPIC_API_KEY"]
  }
}
EOF
```

### Test Script Fails with "command not found"

**Problem:** Script lacks execute permissions.

**Solution:**
```bash
chmod +x ./scripts/*.sh
```

---

## Finding Discrepancies

When a flag doesn't behave as documented:

1. **Document the issue** in `docs/sandbox-flag-verification.md` under "Known Issues"
2. **Capture the actual behavior** - what Docker command was generated?
3. **Compare with expected** - check the verification matrix
4. **Create a minimal reproduction** - simplest command that shows the issue
5. **File a bug** - include reproduction steps and expected vs actual behavior

---

## Advanced Testing

### Capture Docker Commands

To see exactly what Docker command is generated without running it:

```bash
# Use a fake docker wrapper
cat > /tmp/fake-docker << 'EOF'
#!/bin/bash
echo "DOCKER COMMAND: $@"
exit 0
EOF
chmod +x /tmp/fake-docker

# Run with fake docker in PATH
PATH="/tmp:/fake-docker:$PATH" ./bin/claudeup sandbox --shell
```

### Test Profile Merging

Verify that CLI flags properly override/merge with profile configuration:

```bash
# Profile with secrets
cat > ~/.claudeup/profiles/merge-test.json << 'EOF'
{
  "sandbox": {
    "secrets": ["PROFILE_SECRET"],
    "env": {"FROM_PROFILE": "yes"}
  }
}
EOF

export PROFILE_SECRET="from-profile"
export CLI_SECRET="from-cli"

# Test: both secrets should be available
./bin/claudeup sandbox \
  --profile merge-test \
  --secret ANTHROPIC_API_KEY \
  --secret CLI_SECRET \
  --shell

# Inside:
# echo $PROFILE_SECRET  # Should work
# echo $CLI_SECRET      # Should work
# echo $FROM_PROFILE    # Should work
```

### Test Secret Exclusion

```bash
# Profile with secrets
cat > ~/.claudeup/profiles/exclude-test.json << 'EOF'
{
  "sandbox": {
    "secrets": ["SECRET1", "SECRET2", "ANTHROPIC_API_KEY"]
  }
}
EOF

export SECRET1="value1"
export SECRET2="value2"

# Exclude SECRET2
./bin/claudeup sandbox \
  --profile exclude-test \
  --no-secret SECRET2 \
  --shell

# Inside:
# echo $SECRET1           # Should be set
# echo $SECRET2           # Should be empty!
# echo $ANTHROPIC_API_KEY # Should be set
```

---

## See Also

- `docs/sandbox-flag-verification.md` - Comprehensive flag verification matrix
- `docker/Dockerfile` - Sandbox image definition
- `internal/sandbox/` - Sandbox implementation code
- `internal/commands/sandbox.go` - CLI command implementation
