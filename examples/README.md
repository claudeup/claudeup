# claudeup Examples

Hands-on tutorials for learning claudeup. Each workflow directory has its own
README with details on audience, script descriptions, and suggested order.

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

Temp mode creates a fresh Claude environment -- your real settings are not visible or affected.

To run against your actual Claude installation:

```bash
./examples/getting-started/01-check-installation.sh --real
```

For scripting or CI (no pauses):

```bash
./examples/getting-started/01-check-installation.sh --non-interactive
```

## Workflows

Start with **Getting Started**, then explore based on what you need.

| Directory                                    | What it covers                                                            | README                                 |
| -------------------------------------------- | ------------------------------------------------------------------------- | -------------------------------------- |
| [`getting-started/`](getting-started/)       | Verify installation, explore profiles, apply your first profile           | [README](getting-started/README.md)    |
| [`profile-management/`](profile-management/) | Save, create, switch, clone, and compose profiles                         | [README](profile-management/README.md) |
| [`plugin-management/`](plugin-management/)   | List, install, enable, disable plugins; check for updates                 | [README](plugin-management/README.md)  |
| [`extensions/`](extensions/)                 | Manage extensions (agents, commands, skills, hooks, rules, output-styles) | [README](extensions/README.md)         |
| [`team-setup/`](team-setup/)                 | Understand scopes, layer personal and team profiles                       | [README](team-setup/README.md)         |
| [`troubleshooting/`](troubleshooting/)       | Diagnose issues, view event history, diff configuration changes           | [README](troubleshooting/README.md)    |

The `lib/` directory contains shared utilities used by all example scripts.

### Suggested learning path

```
getting-started/  -->  profile-management/  -->  team-setup/
                            |
                            v
                   plugin-management/
                            |
                            v
                       extensions/

         (any time)  troubleshooting/
```

1. **Getting Started** -- verify claudeup works and learn basic concepts
2. **Profile Management** -- the core workflow: save, create, switch, compose
3. **Team Setup** -- layer profiles across scopes for team collaboration
4. **Plugin Management** -- manage marketplace plugins and updates
5. **Extensions** -- manage extensions (agents, commands, skills, hooks, rules, output-styles)
6. **Troubleshooting** -- use anytime something goes wrong

## Flags

| Flag                | Behavior                                          |
| ------------------- | ------------------------------------------------- |
| (none)              | Interactive mode with isolated temp directory     |
| `--real`            | Operate on actual `~/.claude/` with safety checks |
| `--non-interactive` | No pauses, for CI/scripting                       |
| `--help`            | Show usage for the specific example               |

## Temp mode vs real mode

Most examples work in **temp mode** (the default), which creates an isolated
`~/.claude/` in `/tmp/`. This is safe for experimentation but means some
commands show placeholder output since there are no real plugins or
marketplaces to work with.

**Real mode** (`--real`) operates on your actual Claude configuration. It
requires interactive mode and checks for uncommitted changes in `~/.claude/`
before proceeding. Use this to see realistic output with your actual plugins.
