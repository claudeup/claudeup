#!/usr/bin/env bash
# ABOUTME: Build the sandbox Docker image locally for testing.
# ABOUTME: Creates a multi-arch image tagged as claudeup-sandbox:local.

set -euo pipefail

# Determine the script's directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

cd "$PROJECT_ROOT"

# Detect current architecture
ARCH=$(uname -m)
case "$ARCH" in
  x86_64)
    DOCKER_ARCH="amd64"
    GO_ARCH="amd64"
    ;;
  arm64|aarch64)
    DOCKER_ARCH="arm64"
    GO_ARCH="arm64"
    ;;
  *)
    echo "Unsupported architecture: $ARCH"
    exit 1
    ;;
esac

echo "Building claudeup binary for Linux/$GO_ARCH..."
GOOS=linux GOARCH=$GO_ARCH go build -o docker/claudeup-$GO_ARCH ./cmd/claudeup

echo "Building Docker image for linux/$DOCKER_ARCH..."
docker buildx build \
  --platform linux/$DOCKER_ARCH \
  --tag ghcr.io/claudeup/claudeup-sandbox:local \
  --load \
  docker/

echo ""
echo "✓ Image built successfully: ghcr.io/claudeup/claudeup-sandbox:local"
echo ""
echo "To test the sandbox with your local image:"
echo "  claudeup sandbox --image ghcr.io/claudeup/claudeup-sandbox:local"
