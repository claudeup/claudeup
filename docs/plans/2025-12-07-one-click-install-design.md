# One-Click Install Script Design

## Overview

A single curl command that installs the claudeup CLI:

```bash
curl -fsSL https://claudeup.github.io/install.sh | bash
```

## Design Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Audience | Developers evaluating for personal use | Prioritize speed, minimal prompts |
| Docker image | Lazy load on first `sandbox` use | Faster install, not everyone needs sandbox |
| Install location | Auto-detect (~/.local/bin first, /usr/local/bin with sudo fallback) | No sudo when possible, standard fallback |
| Checksum verification | Skip by default, `--verify` flag available | Minimal friction, security opt-in |
| Version selection | `--version` flag, latest by default | Simple default, flexibility when needed |
| Upgrade behavior | Overwrite with message | Clear feedback without prompts |
| PATH handling | Print instructions only | Never touch shell configs |
| Hosting | GitHub Pages via `claudeup/claudeup.github.io` | Clean URL, separates distribution from code |

## User Experience

### Basic Install

```bash
curl -fsSL https://claudeup.github.io/install.sh | bash
```

Output:
```
Installed claudeup v1.2.0 to ~/.local/bin/claudeup
Run 'claudeup setup' to get started
```

### Optional Flags

```bash
# Install specific version
curl -fsSL https://claudeup.github.io/install.sh | bash -s -- --version v1.2.0

# Verify checksum
curl -fsSL https://claudeup.github.io/install.sh | bash -s -- --verify

# Combine flags
curl -fsSL https://claudeup.github.io/install.sh | bash -s -- --version v1.2.0 --verify
```

### Upgrade Output

```
Upgraded claudeup v1.1.0 → v1.2.0
```

### PATH Warning

```
~/.local/bin is not in your PATH. Add this to your shell config:
  export PATH="$HOME/.local/bin:$PATH"
```

## Script Logic

### Platform Detection

```bash
OS=$(uname -s | tr '[:upper:]' '[:lower:]')    # linux, darwin
ARCH=$(uname -m)                                 # x86_64, arm64, aarch64

case "$ARCH" in
  x86_64)  ARCH="amd64" ;;
  aarch64) ARCH="arm64" ;;
esac
```

### Download Strategy

1. Default: Fetch latest release tag via GitHub API (`/repos/claudeup/claudeup/releases/latest`)
2. With `--version`: Use specified tag directly
3. Binary URL: `https://github.com/claudeup/claudeup/releases/download/${VERSION}/claudeup-${OS}-${ARCH}`

### Install Location Logic

1. Check if ~/.local/bin exists and is writable
   - Yes: Install there, warn if not in PATH
   - No: Try /usr/local/bin with sudo
2. If both fail: Print error and exit

### Checksum Verification (--verify)

1. Download `checksums.txt` from release
2. Extract expected SHA256 for the binary
3. Compute actual checksum
4. Abort if mismatch

### Upgrade Detection

1. Check if binary exists before overwriting
2. Run `claudeup --version` to capture current version
3. After install, show appropriate message (installed/upgraded/reinstalled)

## Error Handling

| Scenario | Response |
|----------|----------|
| Network failure | "Failed to fetch latest version. Check your network connection." |
| Binary not found | "Failed to download claudeup. The release may not exist for your platform." |
| Windows | "This installer is for macOS and Linux. For Windows, download from: [URL]" |
| Unknown platform | "Unsupported platform: ${OS}-${ARCH}. See [URL] for available binaries." |
| Permission denied | "Could not install. Try running with sudo or create ~/.local/bin manually." |
| Checksum mismatch | "Checksum verification failed! [details] Aborting installation." |

### Required Dependencies

- `curl` - Downloads (standard on macOS/Linux)
- `sha256sum` or `shasum` - Verification (detect which is available)

## Repository Structure

Two repos under the `claudeup` org:

**claudeup/claudeup** (main repo)
- Source code
- GitHub Releases with binaries
- Docker image publishing

**claudeup/claudeup.github.io** (distribution)
- `install.sh` at root
- Future: Landing page (index.html)

## Implementation Plan

### Phase 1: Repo Migration

1. Create `claudeup/claudeup.github.io` repo
2. Transfer `malston/claude-pm` to `claudeup/claudeup`
3. Update GitHub secrets for releases and Docker

### Phase 2: Rebranding

1. Rename binary from `claude-pm` to `claudeup`
2. Update `release.yml` - binary names to `claudeup-*`
3. Update `docker.yml` - image name to `claudeup-sandbox`
4. Update Dockerfile, README, docs
5. Tag and release to verify

### Phase 3: Install Script

1. Write `install.sh`
2. Add to `claudeup/claudeup.github.io`
3. Test across platforms
4. Update README with new install instructions

### Phase 4: Polish (optional)

- Custom domain (claudeup.dev)
- Landing page
- CI validation of install script

## Testing Matrix

| OS | Arch | Location | Scenario |
|----|------|----------|----------|
| macOS | arm64 | ~/.local/bin | Fresh install |
| macOS | amd64 | ~/.local/bin | Fresh install |
| Linux | amd64 | ~/.local/bin | Fresh install |
| Linux | arm64 | ~/.local/bin | Fresh install |
| Linux | amd64 | /usr/local/bin | Sudo fallback |

Additional scenarios:
- Upgrade from older version
- Reinstall same version
- `--version` with valid/invalid tag
- `--verify` with good/bad checksum
- PATH warning trigger
