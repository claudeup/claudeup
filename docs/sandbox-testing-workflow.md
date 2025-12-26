# Sandbox Testing Workflow

Quick reference for testing sandbox behavior with proper commands.

## Basic Pattern

**Always use `--shell` for testing** to maintain access to the container:

```bash
./bin/claudeup sandbox [flags] --shell
```

Without `--shell`:
- Launches Claude directly
- Exit Claude → Exit container immediately
- Can't explore filesystem or verify behavior

With `--shell`:
- Drops to bash prompt
- Can run `claude` manually
- Can explore filesystem
- Exit shell when done

## Testing Authentication

### Test 1: API Key Prompt (First Run)

```bash
# Start with shell and API key injected
./bin/claudeup sandbox --profile test --secret ANTHROPIC_API_KEY --shell

# Inside container:
ls -la /root/.claude/           # Check initial state (should be empty)
claude                          # Run Claude manually
# Prompted: "Do you want to use this API key?"
# Select: 1. Yes
# Exit Claude: Ctrl+D

# Back in bash:
ls -la /root/.claude/           # Check what was created
cat /root/.claude/.credentials.json | jq 'keys'
exit                            # Exit container
```

### Test 2: Credentials Persistence (Second Run)

```bash
# Run same profile again
./bin/claudeup sandbox --profile test --secret ANTHROPIC_API_KEY --shell

# Inside container:
ls -la /root/.claude/           # Credentials should exist
claude                          # Run Claude
# Should NOT prompt - uses saved credentials
# Exit Claude: Ctrl+D

exit                            # Exit container
```

### Test 3: Check Host Persistence

```bash
# On host machine (not in container)
ls -la ~/.claudeup/sandboxes/test/
cat ~/.claudeup/sandboxes/test/.credentials.json | jq 'keys'

# Should match what you saw inside the container
```

## Testing Other Flags

### Test: Working Directory Mount

```bash
# Create test file
echo "test content" > test-file.txt

# Start sandbox
./bin/claudeup sandbox --shell

# Inside:
pwd                             # Should be /workspace
ls                              # Should show test-file.txt
cat test-file.txt               # Should show "test content"
exit
```

### Test: --no-mount Flag

```bash
./bin/claudeup sandbox --no-mount --shell

# Inside:
pwd                             # NOT /workspace
ls /workspace                   # Should not exist or be empty
exit
```

### Test: Additional Mounts

```bash
# Create test directory
mkdir -p /tmp/test-mount
echo "mounted data" > /tmp/test-mount/file.txt

# Run with mount
./bin/claudeup sandbox --mount /tmp/test-mount:/data --shell

# Inside:
ls /data                        # Should show file.txt
cat /data/file.txt              # Should show "mounted data"
exit
```

### Test: Read-Only Mounts

```bash
./bin/claudeup sandbox --mount /tmp/test-mount:/data:ro --shell

# Inside:
cat /data/file.txt              # Should work (read)
touch /data/newfile             # Should fail (read-only)
exit
```

### Test: Secret Injection

```bash
export MY_SECRET="secret-value-123"
./bin/claudeup sandbox --secret MY_SECRET --shell

# Inside:
echo $MY_SECRET                 # Should show "secret-value-123"
exit
```

### Test: Profile Mounts/Secrets

```bash
# Create profile with config
cat > ~/.claudeup/profiles/configured.json << 'EOF'
{
  "settings": {},
  "plugins": [],
  "sandbox": {
    "mounts": [
      {"host": "/tmp/profile-mount", "container": "/profile-data", "readOnly": false}
    ],
    "secrets": ["PROFILE_SECRET"],
    "env": {"FROM_PROFILE": "yes"}
  }
}
EOF

# Set profile secret
export PROFILE_SECRET="from-profile-env"

# Create profile mount directory
mkdir -p /tmp/profile-mount
echo "profile data" > /tmp/profile-mount/file.txt

# Run with profile
./bin/claudeup sandbox --profile configured --shell

# Inside:
echo $FROM_PROFILE              # Should show "yes"
echo $PROFILE_SECRET            # Should show "from-profile-env"
ls /profile-data                # Should show file.txt
exit
```

### Test: CLI vs Profile (Merging)

```bash
# Using the configured profile from above
export CLI_SECRET="from-cli"

./bin/claudeup sandbox \
  --profile configured \
  --secret CLI_SECRET \
  --mount /tmp/cli-mount:/cli-data \
  --shell

# Inside - should have BOTH profile and CLI configs:
echo $FROM_PROFILE              # From profile
echo $PROFILE_SECRET            # From profile
echo $CLI_SECRET                # From CLI
ls /profile-data                # From profile
ls /cli-data                    # From CLI
exit
```

### Test: Secret Exclusion

```bash
./bin/claudeup sandbox \
  --profile configured \
  --no-secret PROFILE_SECRET \
  --shell

# Inside:
echo $FROM_PROFILE              # Should work (env, not secret)
echo $PROFILE_SECRET            # Should be empty (excluded)
exit
```

### Test: Ephemeral Override

```bash
# First: Create state with profile
./bin/claudeup sandbox --profile ephemeral-test --shell
# Inside: touch /root/.claude/testfile
# exit

# Verify state exists on host
ls ~/.claudeup/sandboxes/ephemeral-test/testfile

# Second: Run with --ephemeral
./bin/claudeup sandbox --profile ephemeral-test --ephemeral --shell
# Inside: ls /root/.claude/testfile  # Should NOT exist (no mount)
# exit

# Verify: Original state still on host (not deleted)
ls ~/.claudeup/sandboxes/ephemeral-test/testfile
```

### Test: Custom Image

```bash
./bin/claudeup sandbox --image ubuntu:22.04 --shell

# Inside:
cat /etc/os-release             # Should show Ubuntu
which claude                    # Should not exist (not claudeup image)
exit
```

## Common Patterns

### Pattern: Quick Verification

```bash
# Start → Check → Exit
./bin/claudeup sandbox [flags] --shell
ls -la /path/to/check
exit
```

### Pattern: Manual Claude Testing

```bash
# Start with shell
./bin/claudeup sandbox --profile test --secret ANTHROPIC_API_KEY --shell

# Inside: Run Claude when ready
claude

# Exit Claude → Back to shell
# Can explore, run again, or exit container
```

### Pattern: State Inspection

```bash
# After running tests
ls -la ~/.claudeup/sandboxes/             # All profiles
ls -la ~/.claudeup/sandboxes/test/        # Specific profile
cat ~/.claudeup/sandboxes/test/.credentials.json | jq .
```

## Cleanup

```bash
# Remove profile
rm -rf ~/.claudeup/profiles/test.json
rm -rf ~/.claudeup/sandboxes/test

# Remove test data
rm -rf /tmp/test-mount
rm -rf /tmp/profile-mount
rm -rf /tmp/cli-mount
```
