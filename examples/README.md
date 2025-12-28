# claudeup Examples

Hands-on tutorials for learning claudeup.

## Before You Start

### Optional: Version Control Your Claude Config

Your Claude configuration lives in `~/.claude/`. Version controlling it
lets you track changes over time and easily revert if needed:

```bash
cd ~/.claude
git init
git add -A
git commit -m "Initial Claude configuration"
```

The examples with `--real` mode will check for uncommitted changes
to help protect your work.

## Running Examples

Run commands from the repository root directory:

By default, examples run in an isolated temp directory (safe to experiment):

```bash
./examples/getting-started/01-check-installation.sh
```

Temp mode creates a fresh Claude environment - your real settings are not visible or affected.

To run against your actual Claude installation:

```bash
./examples/getting-started/01-check-installation.sh --real
```

For scripting or CI (no pauses):

```bash
./examples/getting-started/01-check-installation.sh --non-interactive
```

## Workflows

| Directory | Description |
|-----------|-------------|
| `getting-started/` | First steps with claudeup |
| `profile-management/` | Create and switch configurations |
| `plugin-management/` | Control your plugins |
| `troubleshooting/` | Diagnose and fix issues |
| `team-setup/` | Share configurations across projects |

The `lib/` directory contains shared utilities used by all example scripts.

## Flags

| Flag | Behavior |
|------|----------|
| (none) | Interactive mode with isolated temp directory |
| `--real` | Operate on actual `~/.claude/` with safety checks |
| `--non-interactive` | No pauses, for CI/scripting |
| `--help` | Show usage for the specific example |
