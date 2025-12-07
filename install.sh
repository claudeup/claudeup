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
trap 'rm -rf $TMP_DIR' EXIT

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
        echo "$HOME/.local/bin is not in your PATH. Add this to your shell config:"
        echo "  export PATH=\"\$HOME/.local/bin:\$PATH\""
    fi
fi

echo ""
echo "Run '$BINARY_NAME setup' to get started"
