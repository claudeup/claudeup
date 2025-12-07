# One-Click Install Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Create a curl-based install script at `https://claudeup.github.io/install.sh` that installs the claudeup CLI binary.

**Architecture:** Rebrand from `claude-pm` to `claudeup`, update CI workflows, then write a bash install script that detects OS/arch, downloads the appropriate binary from GitHub Releases, and installs it.

**Tech Stack:** Bash (install script), Go (binary), GitHub Actions (CI), GitHub Pages (hosting)

---

## Task 1: Rename Go Module and Binary

**Files:**
- Modify: `go.mod:1`
- Modify: `cmd/claude-pm/main.go` → rename to `cmd/claudeup/main.go`

**Step 1: Update go.mod module path**

Change module from `github.com/malston/claude-pm` to `github.com/claudeup/claudeup`:

```go
module github.com/claudeup/claudeup
```

**Step 2: Rename cmd directory**

```bash
git mv cmd/claude-pm cmd/claudeup
```

**Step 3: Update all import paths**

Find and replace all imports from `github.com/malston/claude-pm` to `github.com/claudeup/claudeup`.

**Step 4: Verify build works**

Run: `go build -o bin/claudeup ./cmd/claudeup`
Expected: Binary builds successfully

**Step 5: Run tests**

Run: `go test ./...`
Expected: All tests pass

**Step 6: Commit**

```bash
git add -A
git commit -m "refactor: rename module to github.com/claudeup/claudeup"
```

---

## Task 2: Update Release Workflow

**Files:**
- Modify: `.github/workflows/release.yml`

**Step 1: Update binary names in release workflow**

Change all `claude-pm-*` to `claudeup-*`:

```yaml
    - name: Build binaries
      env:
        VERSION: ${{ github.ref_name }}
      run: |
        GOOS=linux GOARCH=amd64 go build -ldflags "-X main.version=$VERSION" -o bin/claudeup-linux-amd64 ./cmd/claudeup
        GOOS=linux GOARCH=arm64 go build -ldflags "-X main.version=$VERSION" -o bin/claudeup-linux-arm64 ./cmd/claudeup
        GOOS=darwin GOARCH=amd64 go build -ldflags "-X main.version=$VERSION" -o bin/claudeup-darwin-amd64 ./cmd/claudeup
        GOOS=darwin GOARCH=arm64 go build -ldflags "-X main.version=$VERSION" -o bin/claudeup-darwin-arm64 ./cmd/claudeup
        GOOS=windows GOARCH=amd64 go build -ldflags "-X main.version=$VERSION" -o bin/claudeup-windows-amd64.exe ./cmd/claudeup
        cd bin && sha256sum * > checksums.txt

    - name: Create Release
      uses: softprops/action-gh-release@v2
      with:
        files: |
          bin/claudeup-linux-amd64
          bin/claudeup-linux-arm64
          bin/claudeup-darwin-amd64
          bin/claudeup-darwin-arm64
          bin/claudeup-windows-amd64.exe
          bin/checksums.txt
        generate_release_notes: true
```

**Step 2: Commit**

```bash
git add .github/workflows/release.yml
git commit -m "ci: update release workflow for claudeup binary names"
```

---

## Task 3: Update Docker Workflow

**Files:**
- Modify: `.github/workflows/docker.yml`
- Modify: `docker/Dockerfile`

**Step 1: Update Docker workflow image name**

Change `IMAGE_NAME` from `${{ github.repository }}-sandbox` to explicit name:

```yaml
env:
  REGISTRY: ghcr.io
  IMAGE_NAME: claudeup/claudeup-sandbox
```

**Step 2: Update binary build path in workflow**

```yaml
    - name: Build claudeup binary for Linux
      env:
        VERSION: ${{ github.ref_name }}
      run: |
        GOOS=linux GOARCH=amd64 go build -ldflags "-X main.version=$VERSION" -o docker/claudeup ./cmd/claudeup
```

**Step 3: Update Dockerfile**

Change `claude-pm` references to `claudeup`:

```dockerfile
# ABOUTME: Dockerfile for claudeup sandbox environment.
# ABOUTME: Runs Claude Code CLI in an isolated container with claudeup for plugin management.

FROM ubuntu:24.04

ENV DEBIAN_FRONTEND=noninteractive

RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    curl \
    git \
    && rm -rf /var/lib/apt/lists/*

RUN curl -fsSL https://claude.ai/install.sh | bash \
    && ln -s /root/.local/bin/claude /usr/local/bin/claude

COPY claudeup /usr/local/bin/claudeup
RUN chmod +x /usr/local/bin/claudeup

WORKDIR /workspace

ENV CLAUDE_CONFIG_DIR=/root/.claude

ENTRYPOINT ["claude"]
```

**Step 4: Commit**

```bash
git add .github/workflows/docker.yml docker/Dockerfile
git commit -m "ci: update docker workflow for claudeup naming"
```

---

## Task 4: Update .gitignore

**Files:**
- Modify: `.gitignore`

**Step 1: Update binary patterns**

Change `/claude-pm` patterns to `/claudeup`:

```gitignore
# Build artifacts
/claudeup
/claudeup-*
docker/claudeup
```

**Step 2: Commit**

```bash
git add .gitignore
git commit -m "chore: update gitignore for claudeup binary names"
```

---

## Task 5: Update README

**Files:**
- Modify: `README.md`

**Step 1: Update README with new name and install instructions**

```markdown
# claudeup

A CLI tool for managing Claude Code plugins, profiles, and sandboxed environments.

## Install

```bash
# One-liner install (macOS/Linux)
curl -fsSL https://claudeup.github.io/install.sh | bash

# Or from source
go install github.com/claudeup/claudeup/cmd/claudeup@latest
```

## Get Started

```bash
# First-time setup - installs Claude CLI and applies a profile
claudeup setup

# Or setup with a specific profile
claudeup setup --profile frontend
```
```

Update all other `claude-pm` references in README to `claudeup`.

**Step 2: Commit**

```bash
git add README.md
git commit -m "docs: rebrand README to claudeup"
```

---

## Task 6: Update Documentation Files

**Files:**
- Modify: `docs/commands.md`
- Modify: `docs/profiles.md`
- Modify: `docs/sandbox.md`
- Modify: `docs/troubleshooting.md`

**Step 1: Update all doc files**

Replace all `claude-pm` with `claudeup` in documentation.

**Step 2: Commit**

```bash
git add docs/
git commit -m "docs: rebrand documentation to claudeup"
```

---

## Task 7: Write Install Script

**Files:**
- Create: `install.sh` (will be copied to claudeup.github.io repo)

**Step 1: Create install.sh**

```bash
#!/bin/bash
# ABOUTME: One-click installer for claudeup CLI.
# ABOUTME: Downloads the appropriate binary for the current platform from GitHub Releases.

set -euo pipefail

REPO="claudeup/claudeup"
BINARY_NAME="claudeup"

# Parse arguments
VERSION=""
VERIFY=false

while [[ $# -gt 0 ]]; do
    case $1 in
        --version)
            VERSION="$2"
            shift 2
            ;;
        --verify)
            VERIFY=true
            shift
            ;;
        *)
            echo "Unknown option: $1"
            echo "Usage: curl -fsSL https://claudeup.github.io/install.sh | bash -s -- [--version VERSION] [--verify]"
            exit 1
            ;;
    esac
done

# Detect platform
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case "$ARCH" in
    x86_64)  ARCH="amd64" ;;
    aarch64) ARCH="arm64" ;;
esac

# Check supported platform
if [[ "$OS" != "linux" && "$OS" != "darwin" ]]; then
    echo "This installer is for macOS and Linux."
    echo "For Windows, download from: https://github.com/$REPO/releases"
    exit 1
fi

# Get version
if [[ -z "$VERSION" ]]; then
    VERSION=$(curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/')
    if [[ -z "$VERSION" ]]; then
        echo "Failed to fetch latest version. Check your network connection."
        exit 1
    fi
fi

BINARY_URL="https://github.com/$REPO/releases/download/$VERSION/${BINARY_NAME}-${OS}-${ARCH}"

# Check for existing installation
EXISTING_VERSION=""
if command -v "$BINARY_NAME" &> /dev/null; then
    EXISTING_VERSION=$("$BINARY_NAME" --version 2>/dev/null | head -1 || echo "")
fi

# Create temp directory
TMP_DIR=$(mktemp -d)
trap "rm -rf $TMP_DIR" EXIT

# Download binary
echo "Downloading $BINARY_NAME $VERSION for $OS-$ARCH..."
if ! curl -fsSL "$BINARY_URL" -o "$TMP_DIR/$BINARY_NAME"; then
    echo "Failed to download $BINARY_NAME."
    echo "The release may not exist for your platform: $OS-$ARCH"
    echo "Check available releases at: https://github.com/$REPO/releases"
    exit 1
fi

chmod +x "$TMP_DIR/$BINARY_NAME"

# Verify checksum if requested
if [[ "$VERIFY" == true ]]; then
    echo "Verifying checksum..."
    CHECKSUMS_URL="https://github.com/$REPO/releases/download/$VERSION/checksums.txt"
    if ! curl -fsSL "$CHECKSUMS_URL" -o "$TMP_DIR/checksums.txt"; then
        echo "Failed to download checksums file."
        exit 1
    fi

    EXPECTED=$(grep "${BINARY_NAME}-${OS}-${ARCH}$" "$TMP_DIR/checksums.txt" | awk '{print $1}')

    # Use shasum on macOS, sha256sum on Linux
    if command -v sha256sum &> /dev/null; then
        ACTUAL=$(sha256sum "$TMP_DIR/$BINARY_NAME" | awk '{print $1}')
    elif command -v shasum &> /dev/null; then
        ACTUAL=$(shasum -a 256 "$TMP_DIR/$BINARY_NAME" | awk '{print $1}')
    else
        echo "Neither sha256sum nor shasum found. Cannot verify checksum."
        exit 1
    fi

    if [[ "$EXPECTED" != "$ACTUAL" ]]; then
        echo "Checksum verification failed!"
        echo "Expected: $EXPECTED"
        echo "Got:      $ACTUAL"
        echo "This could indicate a corrupted download or tampering."
        echo "Aborting installation."
        exit 1
    fi
    echo "Checksum verified."
fi

# Determine install location
INSTALL_DIR=""
USED_SUDO=false

if [[ -d "$HOME/.local/bin" && -w "$HOME/.local/bin" ]]; then
    INSTALL_DIR="$HOME/.local/bin"
elif [[ -w "/usr/local/bin" ]]; then
    INSTALL_DIR="/usr/local/bin"
else
    # Try to create ~/.local/bin
    if mkdir -p "$HOME/.local/bin" 2>/dev/null; then
        INSTALL_DIR="$HOME/.local/bin"
    else
        # Fall back to sudo
        echo "Installing to /usr/local/bin (requires sudo)..."
        if sudo mv "$TMP_DIR/$BINARY_NAME" "/usr/local/bin/$BINARY_NAME"; then
            INSTALL_DIR="/usr/local/bin"
            USED_SUDO=true
        else
            echo "Could not install to ~/.local/bin or /usr/local/bin."
            echo "Try running with sudo or create ~/.local/bin manually."
            exit 1
        fi
    fi
fi

# Install binary (if not already done via sudo)
if [[ "$USED_SUDO" == false ]]; then
    mv "$TMP_DIR/$BINARY_NAME" "$INSTALL_DIR/$BINARY_NAME"
fi

# Report result
NEW_VERSION=$("$INSTALL_DIR/$BINARY_NAME" --version 2>/dev/null | head -1 || echo "$VERSION")

if [[ -n "$EXISTING_VERSION" ]]; then
    if [[ "$EXISTING_VERSION" == "$NEW_VERSION" ]]; then
        echo "Reinstalled $BINARY_NAME $NEW_VERSION"
    else
        echo "Upgraded $BINARY_NAME $EXISTING_VERSION → $NEW_VERSION"
    fi
else
    echo "Installed $BINARY_NAME $NEW_VERSION to $INSTALL_DIR/$BINARY_NAME"
fi

# Check PATH
if [[ "$INSTALL_DIR" == "$HOME/.local/bin" ]]; then
    if [[ ":$PATH:" != *":$HOME/.local/bin:"* ]]; then
        echo ""
        echo "~/.local/bin is not in your PATH. Add this to your shell config:"
        echo "  export PATH=\"\$HOME/.local/bin:\$PATH\""
    fi
fi

echo ""
echo "Run '$BINARY_NAME setup' to get started"
```

**Step 2: Make executable and test locally**

```bash
chmod +x install.sh
```

**Step 3: Commit**

```bash
git add install.sh
git commit -m "feat: add one-click install script"
```

---

## Task 8: Update Internal Package Paths

**Files:**
- All files in `internal/` that import other internal packages

**Step 1: Find all files with old import path**

```bash
grep -r "github.com/malston/claude-pm" --include="*.go"
```

**Step 2: Update all imports**

Replace `github.com/malston/claude-pm` with `github.com/claudeup/claudeup` in all Go files.

**Step 3: Verify build**

Run: `go build ./...`
Expected: Build succeeds

**Step 4: Run tests**

Run: `go test ./...`
Expected: All tests pass

**Step 5: Commit**

```bash
git add -A
git commit -m "refactor: update internal package imports"
```

---

## Task 9: Final Verification

**Step 1: Full test suite**

Run: `go test -v ./...`
Expected: All tests pass

**Step 2: Build all platforms**

```bash
GOOS=linux GOARCH=amd64 go build -o /dev/null ./cmd/claudeup
GOOS=linux GOARCH=arm64 go build -o /dev/null ./cmd/claudeup
GOOS=darwin GOARCH=amd64 go build -o /dev/null ./cmd/claudeup
GOOS=darwin GOARCH=arm64 go build -o /dev/null ./cmd/claudeup
GOOS=windows GOARCH=amd64 go build -o /dev/null ./cmd/claudeup
```

Expected: All build without errors

**Step 3: Commit any remaining changes**

```bash
git status
# If any uncommitted changes:
git add -A
git commit -m "chore: final cleanup"
```

---

## Post-Implementation Steps (Manual)

After merging this branch:

1. **Push install.sh to claudeup.github.io repo**
   - Clone `claudeup/claudeup.github.io`
   - Copy `install.sh` to root
   - Commit and push

2. **Create a release**
   - Tag with `v0.2.0` or appropriate version
   - Verify CI builds and publishes artifacts

3. **Test the install script**
   - From a clean machine or container:
     ```bash
     curl -fsSL https://claudeup.github.io/install.sh | bash
     claudeup --version
     ```

4. **Update claudeup.github.io landing page** (optional)
   - Add install instructions to index.html
