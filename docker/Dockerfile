# ABOUTME: Dockerfile for claude-pm sandbox environment.
# ABOUTME: Runs Claude Code CLI in an isolated container with claude-pm for plugin management.

FROM ubuntu:24.04

# Avoid interactive prompts during package installation
ENV DEBIAN_FRONTEND=noninteractive

# Install minimal runtime dependencies
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    curl \
    git \
    && rm -rf /var/lib/apt/lists/*

# Install Claude CLI using official installer
# The installer places the binary at ~/.local/bin/claude
RUN curl -fsSL https://claude.ai/install.sh | bash \
    && ln -s /root/.local/bin/claude /usr/local/bin/claude

# Copy claude-pm binary (built for linux-amd64)
# This is provided during the docker build via --build-arg or copied in
COPY claude-pm /usr/local/bin/claude-pm
RUN chmod +x /usr/local/bin/claude-pm

# Set up working directory
WORKDIR /workspace

# Claude config directory will be mounted at runtime for persistence
# or left unmounted for ephemeral sessions
ENV CLAUDE_CONFIG_DIR=/root/.claude

ENTRYPOINT ["claude"]
