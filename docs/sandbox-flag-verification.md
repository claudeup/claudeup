# Sandbox Flag Verification Matrix

This document lists all `claudeup sandbox` flags, their documented behavior, and provides a verification checklist.

## Test Setup

### Prerequisites

**CRITICAL:** Claude Code requires authentication via API key. The sandbox needs the API key injected to avoid interactive login prompts.

```bash
# Build the binary
go build -o bin/claudeup ./cmd/claudeup

# Get your API key from macOS Keychain (where Claude stores it)
# Option 1: Extract from Keychain
ANTHROPIC_API_KEY=$(security find-generic-password -s "Claude" -a "api_key" -w 2>/dev/null || echo "")

# Option 2: If the above doesn't work, use Claude's credentials directly
# Look for the key in ~/.claude/ or get it from console.anthropic.com

# Option 3: Export your API key manually
export ANTHROPIC_API_KEY="sk-ant-..."

# Verify API key is set
if [ -z "$ANTHROPIC_API_KEY" ]; then
  echo "ERROR: ANTHROPIC_API_KEY not set"
  echo "Get your key from: https://console.anthropic.com/settings/keys"
  exit 1
fi

# Create test profile that includes API key as a secret
mkdir -p ~/.claudeup/profiles
cat > ~/.claudeup/profiles/test-sandbox.json << 'EOF'
{
  "settings": {},
  "plugins": [],
  "sandbox": {
    "mounts": [
      {"host": "/tmp/test-mount", "container": "/test", "readOnly": false}
    ],
    "secrets": ["ANTHROPIC_API_KEY", "TEST_SECRET"],
    "env": {"PROFILE_VAR": "from-profile"}
  }
}
EOF

# Set test environment variable
export TEST_SECRET="secret-value"
```

**Why ANTHROPIC_API_KEY is required:**

- Claude Code stores credentials in macOS Keychain
- Without the API key, `claude` launches in interactive mode and prompts for login
- Automated tests need non-interactive operation
- The API key must be injected via `--secret` or profile configuration

## Flag Verification Matrix

### 1. Default Behavior (no flags)

**Documentation says:**

- Runs an ephemeral session (nothing persists)
- Current working directory mounted at `/workspace`
- Launches Claude CLI (not shell)

**How to verify:**

```bash
# Run sandbox (will drop into Claude)
# IMPORTANT: Must have ANTHROPIC_API_KEY in environment
export ANTHROPIC_API_KEY="sk-ant-..."  # or extract from keychain
./bin/claudeup sandbox --secret ANTHROPIC_API_KEY

# Inside container, check:
# 1. pwd should show /workspace
# 2. ls should show files from your host working directory
# 3. Should be in Claude CLI, not bash (no login prompt!)
# 4. Exit and verify ~/.claudeup/sandboxes/ has no directories (ephemeral)
```

**Expected Docker command:**

```bash
docker run -it --rm \
  -v $(pwd):/workspace \
  -e ANTHROPIC_API_KEY=sk-ant-... \
  --network bridge \
  ghcr.io/claudeup/claudeup-sandbox:latest
```

**Verification checklist:**

- [ ] Container starts
- [ ] Working directory is `/workspace`
- [ ] Host files visible in `/workspace`
- [ ] Claude CLI launches (not bash)
- [ ] After exit, no state persisted
- [ ] No directory created in `~/.claudeup/sandboxes/`

---

### 2. `--profile <name>`

**Documentation says:**

- Enables persistent session
- State survives between sessions
- Uses profile configuration

**How to verify:**

```bash
# First run
./bin/claudeup sandbox --profile test-sandbox
# Inside: touch /root/.claude/testfile
# Exit

# Second run
./bin/claudeup sandbox --profile test-sandbox
# Inside: ls /root/.claude/testfile (should exist)
```

**Expected Docker command:**

```bash
docker run -it --rm \
  -v $(pwd):/workspace \
  -v ~/.claudeup/sandboxes/test-sandbox:/root/.claude \
  -v /tmp/test-mount:/test \
  -e TEST_SECRET=secret-value \
  -e PROFILE_VAR=from-profile \
  --network bridge \
  ghcr.io/claudeup/claudeup-sandbox:latest
```

**Verification checklist:**

- [ ] State directory created at `~/.claudeup/sandboxes/test-sandbox`
- [ ] Files in `/root/.claude` persist between sessions
- [ ] Profile mounts are applied (`/test` is mounted)
- [ ] Profile env vars are set (`PROFILE_VAR`)
- [ ] Profile secrets are injected (`TEST_SECRET`)

---

### 3. `--ephemeral`

**Documentation says:**

- Forces ephemeral mode
- Disables persistence even with `--profile`

**How to verify:**

```bash
# With profile but ephemeral
./bin/claudeup sandbox --profile test-sandbox --ephemeral
# Inside: touch /root/.claude/testfile
# Exit

# Verify state was NOT saved
ls ~/.claudeup/sandboxes/test-sandbox  # Should not exist or be empty
```

**Expected Docker command:**

```bash
docker run -it --rm \
  -v $(pwd):/workspace \
  -e ANTHROPIC_API_KEY=sk-ant-... \
  --network bridge \
  ghcr.io/claudeup/claudeup-sandbox:latest
```

**Verification checklist:**

- [ ] No state mount even with `--profile`
- [ ] Files in `/root/.claude` do NOT persist
- [ ] Profile config still applied (mounts, env, secrets)
- [ ] No directory created in `~/.claudeup/sandboxes/`

---

### 4. `--shell`

**Documentation says:**

- Drops to bash instead of Claude CLI

**How to verify:**

```bash
./bin/claudeup sandbox --shell
# Should land in bash prompt, not Claude
# Run: echo $0  (should show "bash")
```

**Expected Docker command:**

```bash
docker run -it --rm \
  -v $(pwd):/workspace \
  --network bridge \
  --entrypoint bash \
  ghcr.io/claudeup/claudeup-sandbox:latest
```

**Verification checklist:**

- [ ] Bash prompt appears (not Claude)
- [ ] Can run shell commands
- [ ] `$0` shows "bash"
- [ ] All mounts still work
- [ ] Can manually run `claude` if needed

---

### 5. `--mount <host:container[:ro]>`

**Documentation says:**

- Adds additional volume mounts
- Format: `host:container` or `host:container:ro`
- Supports `~` expansion

**How to verify:**

```bash
# Create test directory
mkdir -p /tmp/sandbox-mount-test
echo "test content" > /tmp/sandbox-mount-test/file.txt

# Run with mount
./bin/claudeup sandbox --mount /tmp/sandbox-mount-test:/mydata --shell

# Inside container:
cat /mydata/file.txt  # Should show "test content"
echo "new" > /mydata/newfile.txt  # Should work (read-write)
exit

# Verify file persisted on host
cat /tmp/sandbox-mount-test/newfile.txt

# Test read-only mount
./bin/claudeup sandbox --mount /tmp/sandbox-mount-test:/mydata:ro --shell
# Inside: touch /mydata/test  # Should fail (read-only)
```

**Expected Docker command:**

```bash
docker run -it --rm \
  -v $(pwd):/workspace \
  -v /tmp/sandbox-mount-test:/mydata \
  --network bridge \
  --entrypoint bash \
  ghcr.io/claudeup/claudeup-sandbox:latest
```

**Verification checklist:**

- [ ] Additional mount works
- [ ] Files readable in container
- [ ] Read-write mount allows writes
- [ ] Read-only mount (`:ro`) prevents writes
- [ ] `~` expands to home directory
- [ ] Multiple `--mount` flags work

---

### 6. `--no-mount`

**Documentation says:**

- Don't mount current working directory
- No `/workspace` mount

**How to verify:**

```bash
./bin/claudeup sandbox --no-mount --shell
# Inside: ls /workspace  # Should not exist or be empty
# Inside: pwd  # Should be something other than /workspace
```

**Expected Docker command:**

```bash
docker run -it --rm \
  --network bridge \
  --entrypoint bash \
  ghcr.io/claudeup/claudeup-sandbox:latest
```

**Verification checklist:**

- [ ] No `/workspace` directory mount
- [ ] Host files NOT visible in container
- [ ] Container starts in different directory
- [ ] Other mounts still work

---

### 7. `--secret <name>`

**Documentation says:**

- Injects secrets as environment variables
- Resolves from: env vars, 1Password, macOS Keychain

**How to verify:**

```bash
export MY_SECRET="test-value-123"
./bin/claudeup sandbox --secret MY_SECRET --shell

# Inside container:
echo $MY_SECRET  # Should show "test-value-123"
```

**Expected Docker command:**

```bash
docker run -it --rm \
  -v $(pwd):/workspace \
  -e MY_SECRET=test-value-123 \
  --network bridge \
  --entrypoint bash \
  ghcr.io/claudeup/claudeup-sandbox:latest
```

**Verification checklist:**

- [ ] Env var secrets injected
- [ ] 1Password secrets work (`op://...`)
- [ ] Keychain secrets work
- [ ] Multiple `--secret` flags work
- [ ] Unresolvable secrets show warning (don't fail)

---

### 8. `--no-secret <name>`

**Documentation says:**

- Excludes secrets
- Overrides profile secrets

**How to verify:**

```bash
# Profile has TEST_SECRET configured
./bin/claudeup sandbox --profile test-sandbox --no-secret TEST_SECRET --shell

# Inside container:
echo $TEST_SECRET  # Should be empty (excluded)
echo $PROFILE_VAR  # Should work (not excluded)
```

**Expected Docker command:**

```bash
docker run -it --rm \
  -v $(pwd):/workspace \
  -v ~/.claudeup/sandboxes/test-sandbox:/root/.claude \
  -v /tmp/test-mount:/test \
  -e PROFILE_VAR=from-profile \
  --network bridge \
  --entrypoint bash \
  ghcr.io/claudeup/claudeup-sandbox:latest
```

**Verification checklist:**

- [ ] Excluded secret not injected
- [ ] Other profile secrets still work
- [ ] Can exclude multiple secrets
- [ ] CLI `--secret` overrides profile exclusion

---

### 9. `--clean --profile <name>`

**Documentation says:**

- Resets sandbox state for profile
- Removes state directory
- Requires `--profile`

**How to verify:**

```bash
# Create state
./bin/claudeup sandbox --profile test-sandbox --shell
# Inside: touch /root/.claude/testfile
# Exit

# Verify state exists
ls ~/.claudeup/sandboxes/test-sandbox/testfile

# Clean state
./bin/claudeup sandbox --clean --profile test-sandbox

# Verify state removed
ls ~/.claudeup/sandboxes/test-sandbox  # Should not exist

# Try without --profile
./bin/claudeup sandbox --clean  # Should error
```

**Expected behavior:**

- Removes `~/.claudeup/sandboxes/<profile>` directory
- Shows success message
- Errors if `--profile` not provided

**Verification checklist:**

- [ ] State directory deleted
- [ ] Success message shown
- [ ] Error without `--profile`
- [ ] Can re-create state after clean

---

### 10. `--image <image>`

**Documentation says:**

- Overrides default sandbox image
- Default: `ghcr.io/claudeup/claudeup-sandbox:latest`

**How to verify:**

```bash
# Use different image (e.g., ubuntu)
./bin/claudeup sandbox --image ubuntu:22.04 --shell

# Inside: cat /etc/os-release  # Should show Ubuntu
```

**Expected Docker command:**

```bash
docker run -it --rm \
  -v $(pwd):/workspace \
  --network bridge \
  --entrypoint bash \
  ubuntu:22.04
```

**Verification checklist:**

- [ ] Custom image used
- [ ] Image pulled if not available
- [ ] Claude not available in non-claudeup images
- [ ] All mount/env/secret flags still work

---

## Combined Flag Tests

### Test: Profile + CLI Overrides

```bash
./bin/claudeup sandbox \
  --profile test-sandbox \
  --mount /tmp/extra:/extra \
  --secret EXTRA_SECRET \
  --no-secret TEST_SECRET \
  --shell
```

**Expected behavior:**

- Profile mounts + CLI mounts both applied
- Profile env vars set
- CLI secrets injected
- Profile secrets excluded per `--no-secret`
- Persistent state enabled
- Bash shell (not Claude)

**Verification checklist:**

- [ ] Both `/test` (profile) and `/extra` (CLI) mounted
- [ ] `PROFILE_VAR` set from profile
- [ ] `EXTRA_SECRET` set from CLI
- [ ] `TEST_SECRET` NOT set (excluded)
- [ ] State persists to `~/.claudeup/sandboxes/test-sandbox`

---

### Test: Ephemeral + Profile Config

```bash
./bin/claudeup sandbox \
  --profile test-sandbox \
  --ephemeral \
  --shell
```

**Expected behavior:**

- Profile config applied (mounts, env, secrets)
- State does NOT persist (ephemeral overrides profile)

**Verification checklist:**

- [ ] Profile mounts work (`/test`)
- [ ] Profile env vars set
- [ ] Profile secrets injected
- [ ] State does NOT persist after exit
- [ ] No state directory created

---

## Known Issues

Document any discovered discrepancies here:

### Issue 1: [Flag name] - [Brief description]

**Expected:**
[What should happen]

**Actual:**
[What actually happens]

**Workaround:**
[If any]

---

## Cleanup

```bash
# Remove test profile
rm ~/.claudeup/profiles/test-sandbox.json

# Remove any sandbox state
rm -rf ~/.claudeup/sandboxes/test-sandbox

# Remove test mount directory
rm -rf /tmp/sandbox-mount-test
rm -rf /tmp/test-mount
```
