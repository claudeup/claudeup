# Sandbox Flag Discrepancies

This document tracks discovered discrepancies between documented behavior and actual behavior of `claudeup sandbox` flags.

## CRITICAL: Authentication Does Not Work as Documented

**Discovered:** 2024-12-26

### Expected Behavior (from documentation)

The test documentation assumed:
```bash
export ANTHROPIC_API_KEY="sk-ant-..."
./bin/claudeup sandbox --secret ANTHROPIC_API_KEY
# Expected: Claude starts without login prompt
```

### Actual Behavior

**Claude CLI detects `ANTHROPIC_API_KEY` but prompts interactively instead of using it automatically!**

What actually happens:
1. Claude detects the `ANTHROPIC_API_KEY` environment variable ✓
2. Displays interactive prompt: "Do you want to use this API key?"
3. Defaults to "No (recommended)" - recommends OAuth instead
4. Waits for user to select Yes/No (blocks automation)
5. If "No" selected, proceeds with OAuth flow

**Evidence:**
```bash
Detected a custom API key in your environment

ANTHROPIC_API_KEY: sk-ant-...2PunlZAC9PA-oZJarQAA

Do you want to use this API key?

> 1. Yes
  2. No (recommended) ✓

Enter to confirm . Esc to cancel
```

**Why OAuth is "recommended":**
- OAuth tokens are organization-scoped
- API keys are personal/sensitive
- OAuth provides better audit trails
- But this breaks non-interactive automation!

### Impact

**UPDATE:** Authentication with API key appears to work, but gets stuck on workspace trust prompt!

Debug logs show:
```text
2025-12-26T14:57:05.153Z [DEBUG] Skipping Notification:auth_success hook execution - workspace trust not accepted
```

**Current status:**
- API key injection via `--secret ANTHROPIC_API_KEY` DOES work ✓
- Authentication succeeds ✓
- But Claude waits for "workspace trust" prompt
- Prompt may not be visible/rendered correctly in Docker container
- User can't proceed without accepting workspace trust
- Investigation ongoing...

**Ephemeral mode:** BROKEN for non-interactive use
- Requires both API key prompt AND workspace trust prompt
- No way to pre-configure workspace trust
- Makes automated testing impossible

**Profile mode:** Partially works
- First run: Must complete API key selection AND workspace trust
- Unclear if workspace trust persists between sessions
- Credentials may save to `~/.claudeup/sandboxes/<profile>/.credentials.json`
- Subsequent runs: TBD (still investigating)

### Root Cause

Claude Code authentication architecture:
- Does NOT check `ANTHROPIC_API_KEY` env var
- Does NOT support API key authentication via env vars
- ONLY uses OAuth tokens from `.credentials.json`
- `.credentials.json` must exist in `~/.claude/` directory

### Workarounds

#### For Profile Mode (Recommended)

```bash
# 1. Create profile
mkdir -p ~/.claudeup/profiles
cat > ~/.claudeup/profiles/myprofile.json << 'EOF'
{
  "settings": {},
  "plugins": []
}
EOF

# 2. First run - complete OAuth flow manually
./bin/claudeup sandbox --profile myprofile
# Complete OAuth in browser
# Credentials saved to ~/.claudeup/sandboxes/myprofile/.credentials.json

# 3. Subsequent runs - automatic authentication
./bin/claudeup sandbox --profile myprofile
# Works without prompts!
```

#### For Ephemeral Mode (No Good Solution)

**Option 1:** Copy credentials from host
```bash
# NOT RECOMMENDED: Exposes your credentials
./bin/claudeup sandbox \
  --mount ~/.claude/.credentials.json:/root/.claude/.credentials.json:ro \
  --ephemeral
```

**Option 2:** Pre-authenticated ephemeral profile
```bash
# Create a "shared" profile for ephemeral-like usage
./bin/claudeup sandbox --profile shared-ephemeral

# Use it for "ephemeral" sessions (but state persists)
# Delete profile state when truly done
```

**Option 3:** Use --shell and authenticate once per session
```bash
# Drop to shell, run claude manually, authenticate
./bin/claudeup sandbox --shell --ephemeral
# Inside: claude
# Complete OAuth flow each time
```

### Required Documentation Updates

1. **README.md** - Remove all references to `ANTHROPIC_API_KEY` for sandbox usage
2. **docs/sandbox-flag-verification.md** - Update test procedures to use profile-based auth
3. **scripts/setup-test-env.sh** - Add warning that API key doesn't work for sandbox
4. **scripts/README.md** - Document OAuth flow requirement

### Recommended Fixes

#### Short term: Update Documentation

1. Document that profiles are REQUIRED for non-interactive use
2. Add "First Run" setup instructions for OAuth
3. Warn that ephemeral mode requires manual login each time

#### Long term: Add API Key Support to Sandbox

Consider adding a feature to `claudeup` to pre-configure API keys:

```bash
# Proposed feature
./bin/claudeup sandbox --api-key "$ANTHROPIC_API_KEY" --ephemeral

# Implementation would:
# 1. Create temp credentials file
# 2. Inject into container
# 3. Clean up after exit
```

Or modify Claude Code to check `ANTHROPIC_API_KEY` env var as fallback.

---

## Issue 2: Profile Settings Not Applied to Claude

**Discovered:** 2024-12-26

### Expected Behavior

When using `--profile <name>`, the profile's Claude settings should be applied:
```bash
./bin/claudeup sandbox --profile myprofile
```

Expected: Claude inside container should have:
- Profile's plugin configuration
- Profile's marketplace configuration
- Profile's Claude settings
- Profile's custom configurations

### Actual Behavior

**Profile settings are NOT applied to Claude in the sandbox!**

Only the **sandbox-specific** settings work:
- ✅ `sandbox.mounts` - Applied correctly
- ✅ `sandbox.secrets` - Applied correctly
- ✅ `sandbox.env` - Applied correctly
- ❌ `marketplaces` - NOT applied
- ❌ `plugins` - NOT applied (shows "32 plugins failed to install")
- ❌ `settings` - NOT applied

**Evidence:**

Profile configuration:
```json
{
  "name": "test",
  "marketplaces": [
    {
      "source": "github",
      "repo": "obra/superpowers-marketplace"
    }
  ],
  "plugins": [
    "superpowers@superpowers-marketplace"
  ]
}
```

Claude in sandbox showed:
```bash
32 plugins failed to install • /plugin for details
```

### Root Cause

The sandbox mounts `~/.claudeup/sandboxes/<profile>` to `/root/.claude`, creating an isolated Claude environment. The claudeup profile settings (from `~/.claudeup/profiles/<profile>.json`) are never injected into this isolated environment.

**What happens:**
1. Sandbox creates fresh state directory: `~/.claudeup/sandboxes/<profile>/`
2. Mounts it to container: `/root/.claude`
3. Claude initializes with empty/default configuration
4. Profile's `settings`, `marketplaces`, and `plugins` fields are ignored

**What works:**
- Only `sandbox.*` fields from the profile are used (for Docker configuration)
- Claude settings must be configured inside the container manually

### Impact

- **Cannot pre-configure Claude** with plugins, settings, or marketplaces via profiles
- **Each profile starts with vanilla Claude** configuration
- **Manual setup required** inside each sandbox session to install plugins/configure Claude
- **Defeats the purpose** of profiles for Claude configuration

### Workarounds

**Option 1: Manual configuration (doesn't persist in ephemeral mode)**
```bash
./bin/claudeup sandbox --profile myprofile --shell
# Inside:
claude
# Manually configure Claude, install plugins
# Exit
```

**Option 2: Pre-populate state directory**
```bash
# Not recommended - would need to:
# 1. Run sandbox once to create state dir
# 2. Manually copy Claude config files into state dir
# 3. Very fragile, Claude version-dependent
```

**Option 3: Use host's Claude config (NOT RECOMMENDED - security issue)**
```bash
# DANGEROUS - exposes your personal Claude config to sandbox
./bin/claudeup sandbox --mount ~/.claude:/root/.claude
```

### Recommended Fix

Add a feature to apply profile's Claude settings to the sandbox:

```bash
# Proposed behavior
./bin/claudeup sandbox --profile myprofile --apply-claude-settings

# Would:
# 1. Read profile's settings/plugins/marketplaces
# 2. Inject into container's .claude.json on first run
# 3. Pre-install plugins in sandbox state directory
# 4. Apply Claude settings automatically
```

Or automatically apply settings when `--profile` is used:
- Copy profile's Claude config to sandbox state on first run
- Merge with existing state on subsequent runs
- Make it work like users expect!

---

## Other Discovered Issues

(To be filled in as testing continues)

### Issue Template

```markdown
## Issue: [Flag Name] - [Brief Description]

**Expected:** [What should happen according to docs]

**Actual:** [What actually happens]

**Impact:** [How this affects users]

**Workaround:** [If any]

**Evidence:**
[Commands/output showing the issue]
```
