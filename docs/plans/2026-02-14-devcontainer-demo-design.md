# Devcontainer Demo Design

## Goal

Add an end-to-end team setup demo that uses claudeup-lab to create real
devcontainer environments for three team members (Alice, Bob, Charlie),
demonstrating profile stacking with true container isolation.

## Context

The isolated workspace demo (`02-isolated-workspace-demo.sh`) simulates
separate machines using `CLAUDE_CONFIG_DIR` and `CLAUDEUP_HOME` environment
variables. This demo goes further -- each team member gets a real Docker
container with an isolated filesystem, network, and process space.

The two demos are complementary:

- **Isolated workspace** -- lightweight, no Docker required, simulates
  isolation with env vars and scoped profile apply
- **Devcontainer** -- heavyweight, requires Docker, provides true container
  isolation via claudeup-lab

## Script

**File:** `examples/team-setup/04-devcontainer-demo.sh`

### Prerequisites

- Docker running
- `claudeup-lab` in PATH
- `claudeup` in PATH
- `git` available

The script runs `claudeup-lab doctor` upfront and exits early if
prerequisites are not met. The `--real` flag is not supported -- labs always
use isolated temp profiles via `CLAUDEUP_HOME`.

### Sections

1. **Prerequisites** -- run `claudeup-lab doctor`, verify tools
2. **Create fixture profiles** -- write `go-backend-team`, `alice-tools`, and
   `bob-tools` profiles to a temp `CLAUDEUP_HOME` (same profiles as the
   isolated workspace demo)
3. **Create a temp git repo** -- claudeup-lab requires a git project; create
   a minimal repo in the temp directory
4. **Alice starts her lab** -- `claudeup-lab start --project <temp-repo>
--base-profile go-backend-team --profile alice-tools --name alice-lab`
5. **Bob starts his lab** -- `claudeup-lab start --project <temp-repo>
--base-profile go-backend-team --profile bob-tools --name bob-lab`
6. **Charlie starts his lab** -- `claudeup-lab start --project <temp-repo>
--profile go-backend-team --name charlie-lab`
7. **Verify** -- `claudeup-lab list` shows all three labs;
   `claudeup-lab exec --lab <name> -- claudeup profile list` in each
   container to show active profiles
8. **Cleanup** -- `claudeup-lab rm --force` all three labs; remove temp dir
9. **Summary** -- key takeaways comparing containers to isolated workspaces

### Profile Stacking Pattern

```
Alice:   --base-profile go-backend-team --profile alice-tools
Bob:     --base-profile go-backend-team --profile bob-tools
Charlie: --profile go-backend-team
```

All three get the team configuration. Alice and Bob layer personal tools on
top. Charlie uses only the team profile.

### Error Handling

- Trap on ERR cleans up any labs that were created (calls `claudeup-lab rm
--force` for each lab name, ignoring errors for labs that do not exist)
- Temp directory preserved on error for debugging (same as other demos)
- `prompt_cleanup` at the end handles both labs and temp dir

### common.sh Integration

Uses the standard shared library: `section`, `step`, `info`, `success`,
`run_cmd`, `pause`, `resolve_claudeup_bin`, `parse_common_args`,
`trap_preserve_on_error`, `prompt_cleanup`.

Does not use `setup_environment` (same as the isolated workspace demo --
this script manages its own `CLAUDEUP_HOME`).

## Testing

Manual verification only (same as other example scripts):

```bash
# Run the demo
./examples/team-setup/04-devcontainer-demo.sh

# Non-interactive mode
./examples/team-setup/04-devcontainer-demo.sh --non-interactive
```

Requires Docker running and claudeup-lab installed.
