# Sandbox Authentication Workflow (CORRECTED)

This document describes the **actual** authentication workflow for `claudeup sandbox`, based on testing.

## TL;DR

### Profile Mode (Recommended)

**First run:** Interactive setup (2 prompts)
**Subsequent runs:** Fully automated ✓

```bash
# First time
./bin/claudeup sandbox --profile myprofile --secret ANTHROPIC_API_KEY

# Prompts:
# 1. "Do you want to use this API key?" → Select "Yes"
# 2. "Do you trust this workspace?" → Accept

# All future runs - no prompts!
./bin/claudeup sandbox --profile myprofile --secret ANTHROPIC_API_KEY
# Goes straight to Claude! ✓
```

### Ephemeral Mode

**Every run:** Requires both prompts ❌
Not suitable for automation.

---

## Complete Workflow

### Prerequisites

```bash
# 1. Build claudeup
go build -o bin/claudeup ./cmd/claudeup

# 2. Set API key in environment
export ANTHROPIC_API_KEY="sk-ant-..."

# 3. Create profile
mkdir -p ~/.claudeup/profiles
cat > ~/.claudeup/profiles/myprofile.json << 'EOF'
{
  "settings": {},
  "plugins": [],
  "sandbox": {
    "secrets": ["ANTHROPIC_API_KEY"]
  }
}
EOF
```

### First Run (Interactive Setup)

```bash
./bin/claudeup sandbox --profile myprofile --secret ANTHROPIC_API_KEY
```

**What happens:**

1. **Container starts**
   ```text
   Sandbox

   Profile: myprofile (persistent)
   Workdir: /Users/you/project → /workspace
   Secrets: 1 injected
   Entry: claude
   ```

2. **API Key Prompt appears**
   ```bash
   Detected a custom API key in your environment

   ANTHROPIC_API_KEY: sk-ant-...

   Do you want to use this API key?

   > 1. Yes
     2. No (recommended)
   ```

   **Action:** Select "1. Yes" and press Enter

3. **Workspace Trust Prompt**
   ```text
   Do you trust this workspace?

   /workspace

   > Accept
     Decline
   ```

   **Action:** Select "Accept" and press Enter

4. **Claude Launches!**
   ```text
   Welcome back <your name>!

   Claude Code v2.0.76
   Sonnet 4.5 · API Usage Billing
   /workspace

   > Try "refactor <filepath>"
   ```

5. **Authentication persisted**
   - Credentials saved to `~/.claudeup/sandboxes/myprofile/`
   - Workspace trust saved
   - Future runs skip both prompts!

### Subsequent Runs (Automated)

```bash
./bin/claudeup sandbox --profile myprofile --secret ANTHROPIC_API_KEY
```

**What happens:**

1. **Container starts** (same as before)
2. **Claude launches immediately** (no prompts!)
3. Ready to use ✓

---

## Testing the Workflow

### Verify First-Time Setup

```bash
# Start sandbox
./bin/claudeup sandbox --profile test --secret ANTHROPIC_API_KEY

# Accept both prompts
# Claude should start

# Test it works:
Hello! Can you respond?

# Exit
/exit
```

### Verify Persistence

```bash
# Run again with same profile
./bin/claudeup sandbox --profile test --secret ANTHROPIC_API_KEY

# Should go straight to Claude (no prompts!)

# Exit
/exit
```

### Check What Was Saved

```bash
# View profile state directory
ls -la ~/.claudeup/sandboxes/test/

# Should contain:
# .claude.json           - Claude settings
# .claude.json.backup    - Backup
# debug/                 - Debug logs
# plugins/               - Installed plugins
# projects/              - Project data
# todos/                 - Todo list data
```

---

## Using with Shell Access

For testing and debugging, use `--shell` to get bash access:

```bash
./bin/claudeup sandbox --profile test --secret ANTHROPIC_API_KEY --shell

# Inside container:
root@abc123:/workspace# ls -la /root/.claude/
root@abc123:/workspace# claude
# (Prompts appear here if first run)
# Ctrl+D to exit Claude, back to bash

root@abc123:/workspace# exit
```

---

## Troubleshooting

### "Auth conflict" Warning

When Claude starts, you might see:

```text
⚠️ Auth conflict: Using ANTHROPIC_API_KEY instead of Anthropic Console key.
Either unset ANTHROPIC_API_KEY, or run `claude /logout`.
```

**This is normal and can be ignored.** Claude is telling you that it detected both:
- The ANTHROPIC_API_KEY environment variable (which you injected)
- Potentially old OAuth credentials (from previous runs)

Claude will use the API key as intended. The warning is just informational.

### Prompts Appear Every Time

If prompts appear on every run (not just first time):

**Cause:** Not using a profile, or using `--ephemeral`

**Solution:**
```bash
# Make sure you're using --profile (NOT --ephemeral)
./bin/claudeup sandbox --profile myprofile --secret ANTHROPIC_API_KEY

# NOT this:
./bin/claudeup sandbox --ephemeral --secret ANTHROPIC_API_KEY
```

### "32 plugins failed to install"

This is expected. The profile's plugin configuration is **not** applied to the sandbox (see Issue #2 in sandbox-discrepancies.md).

The sandbox starts with vanilla Claude. Plugins must be manually installed inside the sandbox if needed.

---

## What Works vs What Doesn't

### ✅ What Works

| Feature | Status |
|---------|--------|
| API key injection via `--secret` | ✓ Works |
| Profile persistence (auth + workspace trust) | ✓ Works |
| No prompts on subsequent runs | ✓ Works |
| Working directory mount | ✓ Works |
| Additional mounts (`--mount`) | ✓ Works |
| Secret injection (`--secret`) | ✓ Works |
| Environment variables (`sandbox.env`) | ✓ Works |
| Custom Docker image (`--image`) | ✓ Works |
| Shell access (`--shell`) | ✓ Works |

### ❌ What Doesn't Work

| Feature | Status |
|---------|--------|
| Ephemeral mode without prompts | ✗ Broken |
| Profile Claude settings applied | ✗ Not implemented |
| Profile plugins installed | ✗ Not implemented |
| Profile marketplaces configured | ✗ Not implemented |
| Fully non-interactive (first run) | ✗ Requires manual prompts |

---

## Summary

**Profile mode works well** for persistent, authenticated sessions:
- One-time interactive setup (2 prompts)
- Fully automated afterwards
- State persists between runs

**Ephemeral mode is not suitable** for automation:
- Requires prompts every time
- No way to pre-configure
- Only useful for manual, interactive use

**Profile Claude settings are not applied:**
- Sandbox starts with vanilla Claude configuration
- Plugins, marketplaces, settings must be configured manually
- This is a significant limitation (see Issue #2)
