#!/usr/bin/env bash
# ABOUTME: Shared library for claudeup example scripts
# ABOUTME: Provides colors, prompts, safety checks, and temp environment setup

set -euo pipefail

# =============================================================================
# Colors (disabled when not a TTY)
# =============================================================================

if [[ -t 1 ]]; then
    RED='\033[0;31m'
    GREEN='\033[0;32m'
    YELLOW='\033[1;33m'
    BLUE='\033[0;34m'
    MAGENTA='\033[0;35m'
    CYAN='\033[0;36m'
    BOLD='\033[1m'
    NC='\033[0m'
else
    RED='' GREEN='' YELLOW='' BLUE='' MAGENTA='' CYAN='' BOLD='' NC=''
fi

# =============================================================================
# Global State Variables
# =============================================================================

EXAMPLE_TEMP_DIR=""
EXAMPLE_REAL_MODE=false
EXAMPLE_INTERACTIVE=true
EXAMPLE_CLAUDEUP_BIN="${CLAUDEUP_BIN:-claudeup}"

# =============================================================================
# Output Helpers
# =============================================================================

section() {
    local title="$1"
    echo ""
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${BLUE}┃${NC} ${BOLD}${title}${NC}"
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo ""
}

step() {
    local msg="$1"
    echo -e "${MAGENTA}→${NC} ${msg}"
}

info() {
    local msg="$1"
    echo -e "${CYAN}ℹ${NC} ${msg}"
}

warn() {
    local msg="$1"
    echo -e "${YELLOW}⚠${NC} ${msg}"
}

error() {
    local msg="$1"
    echo -e "${RED}✖${NC} ${msg}" >&2
}

success() {
    local msg="$1"
    echo -e "${GREEN}✔${NC} ${msg}"
}

# =============================================================================
# Interactive Helpers
# =============================================================================

pause() {
    if [[ "$EXAMPLE_INTERACTIVE" == "true" ]]; then
        echo ""
        read -r -p "Press ENTER to continue..."
        echo ""
    fi
}

run_cmd() {
    local cmd="$*"
    echo -e "${YELLOW}\$ ${cmd}${NC}"
    eval "$cmd"
}

# =============================================================================
# Environment Detection
# =============================================================================

check_claudeup_installed() {
    if ! command -v "$EXAMPLE_CLAUDEUP_BIN" &>/dev/null; then
        error "claudeup not found in PATH"
        error "Please install claudeup first: go install github.com/claudeup/claudeup/cmd/claudeup@latest"
        exit 1
    fi
    success "Found claudeup: $(command -v "$EXAMPLE_CLAUDEUP_BIN")"
}

check_claude_config_dir_override() {
    if [[ -n "${CLAUDE_CONFIG_DIR:-}" ]]; then
        warn "CLAUDE_CONFIG_DIR is set to: $CLAUDE_CONFIG_DIR"
        warn "This will affect where claudeup looks for Claude configuration"
    fi
}

# =============================================================================
# Safety Checks for --real Mode
# =============================================================================

check_git_clean() {
    local claude_dir="${HOME}/.claude"

    if [[ ! -d "$claude_dir/.git" ]]; then
        warn "~/.claude is not a git repository"
        warn "Consider initializing git for safety: cd ~/.claude && git init && git add -A && git commit -m 'Initial'"
        return 0
    fi

    if ! git -C "$claude_dir" diff --quiet 2>/dev/null || \
       ! git -C "$claude_dir" diff --cached --quiet 2>/dev/null; then
        error "~/.claude has uncommitted changes"
        error "Please commit or stash changes before running in --real mode"
        echo ""
        git -C "$claude_dir" status --short
        exit 1
    fi

    success "~/.claude git status is clean"
}

warn_real_mode() {
    echo ""
    echo -e "${RED}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${RED}┃${NC} ${BOLD}WARNING: REAL MODE${NC}"
    echo -e "${RED}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo ""
    warn "This will modify your REAL Claude configuration at ~/.claude"
    warn "Changes will affect your actual Claude Code installation"
    echo ""

    if [[ "$EXAMPLE_INTERACTIVE" != "true" ]]; then
        error "Cannot run --real mode in non-interactive mode without explicit confirmation"
        error "This is a safety measure to prevent accidental modification of real config"
        exit 1
    fi

    read -r -p "Are you sure you want to continue? [y/N] " response
    case "$response" in
        [yY][eE][sS]|[yY])
            success "Proceeding with real mode..."
            ;;
        *)
            info "Aborted by user"
            exit 0
            ;;
    esac
}

# =============================================================================
# Temp Environment Setup
# =============================================================================

setup_temp_claude_dir() {
    EXAMPLE_TEMP_DIR=$(mktemp -d "/tmp/claudeup-example-XXXXX")
    export CLAUDE_CONFIG_DIR="$EXAMPLE_TEMP_DIR"
    export CLAUDEUP_HOME="$EXAMPLE_TEMP_DIR/.claudeup"

    # Create basic directory structure
    mkdir -p "$EXAMPLE_TEMP_DIR/plugins"
    mkdir -p "$CLAUDEUP_HOME"

    success "Created isolated environment: $EXAMPLE_TEMP_DIR"
    info "CLAUDE_CONFIG_DIR=$CLAUDE_CONFIG_DIR"
    info "CLAUDEUP_HOME=$CLAUDEUP_HOME"
}

cleanup_temp_dir() {
    if [[ -n "$EXAMPLE_TEMP_DIR" && -d "$EXAMPLE_TEMP_DIR" ]]; then
        rm -rf "$EXAMPLE_TEMP_DIR"
        success "Cleaned up temp directory"
    fi
}

on_error() {
    local exit_code=$?
    echo ""
    error "Script failed with exit code $exit_code"

    if [[ -n "$EXAMPLE_TEMP_DIR" && -d "$EXAMPLE_TEMP_DIR" ]]; then
        warn "Preserving temp directory for debugging: $EXAMPLE_TEMP_DIR"
        warn "Contents:"
        ls -la "$EXAMPLE_TEMP_DIR" 2>/dev/null || true
    fi

    exit $exit_code
}

trap_preserve_on_error() {
    trap on_error ERR
}

prompt_cleanup() {
    if [[ -z "$EXAMPLE_TEMP_DIR" || ! -d "$EXAMPLE_TEMP_DIR" ]]; then
        return 0
    fi

    if [[ "$EXAMPLE_INTERACTIVE" != "true" ]]; then
        cleanup_temp_dir
        return 0
    fi

    echo ""
    read -r -p "Remove temp directory $EXAMPLE_TEMP_DIR? [Y/n] " response
    case "$response" in
        [nN][oO]|[nN])
            info "Keeping temp directory: $EXAMPLE_TEMP_DIR"
            ;;
        *)
            cleanup_temp_dir
            ;;
    esac
}

# =============================================================================
# Argument Parsing
# =============================================================================

show_help() {
    local script_name="${EXAMPLE_SCRIPT_NAME:-$(basename "$0")}"
    local description="${EXAMPLE_DESCRIPTION:-Example script}"

    echo "Usage: $script_name [OPTIONS]"
    echo ""
    echo "$description"
    echo ""
    echo "Options:"
    echo "  --real            Use real ~/.claude config (default: isolated temp dir)"
    echo "  --non-interactive Skip prompts and confirmations"
    echo "  --help, -h        Show this help message"
    echo ""
    echo "Environment Variables:"
    echo "  CLAUDEUP_BIN      Path to claudeup binary (default: claudeup)"
    echo "  CLAUDE_CONFIG_DIR Override Claude config directory"
}

parse_common_args() {
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --real)
                EXAMPLE_REAL_MODE=true
                shift
                ;;
            --non-interactive)
                EXAMPLE_INTERACTIVE=false
                shift
                ;;
            --help|-h)
                show_help
                exit 0
                ;;
            *)
                # Unknown arg - let caller handle it
                break
                ;;
        esac
    done
}

# =============================================================================
# Main Setup
# =============================================================================

setup_environment() {
    check_claudeup_installed
    check_claude_config_dir_override

    if [[ "$EXAMPLE_REAL_MODE" == "true" ]]; then
        check_git_clean
        warn_real_mode
    else
        setup_temp_claude_dir
        trap_preserve_on_error
    fi
}
