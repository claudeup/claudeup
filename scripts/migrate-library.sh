#!/usr/bin/env bash
# ABOUTME: One-time migration script for library relocation (issue #138)
# ABOUTME: Moves .library/ and enabled.json from CLAUDE_CONFIG_DIR to CLAUDEUP_HOME
set -euo pipefail

# Resolve directories
CLAUDE_DIR="${CLAUDE_CONFIG_DIR:-$HOME/.claude}"
CLAUDEUP_HOME="${CLAUDEUP_HOME:-$HOME/.claudeup}"

OLD_LIBRARY="$CLAUDE_DIR/.library"
OLD_CONFIG="$CLAUDE_DIR/enabled.json"
NEW_LIBRARY="$CLAUDEUP_HOME/local"
NEW_CONFIG="$CLAUDEUP_HOME/enabled.json"

CATEGORIES=(agents commands hooks output-styles rules skills)

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BOLD='\033[1m'
NC='\033[0m'

info()  { echo -e "${GREEN}[ok]${NC} $1"; }
warn()  { echo -e "${YELLOW}[!!]${NC} $1"; }
error() { echo -e "${RED}[err]${NC} $1"; }
bold()  { echo -e "${BOLD}$1${NC}"; }

echo
bold "claudeup library migration"
echo "Migrating from $CLAUDE_DIR to $CLAUDEUP_HOME"
echo

# Pre-flight checks
has_old_library=false
has_old_config=false

if [[ -d "$OLD_LIBRARY" ]]; then
    has_old_library=true
fi

if [[ -f "$OLD_CONFIG" ]]; then
    has_old_config=true
fi

if [[ "$has_old_library" == false && "$has_old_config" == false ]]; then
    info "Nothing to migrate. No .library/ or enabled.json found in $CLAUDE_DIR"
    exit 0
fi

if [[ -d "$NEW_LIBRARY" ]]; then
    error "$NEW_LIBRARY already exists. Migration may have already run."
    error "Remove it first if you want to re-migrate: rm -rf $NEW_LIBRARY"
    exit 1
fi

if [[ -f "$NEW_CONFIG" ]]; then
    error "$NEW_CONFIG already exists. Migration may have already run."
    error "Remove it first if you want to re-migrate: rm $NEW_CONFIG"
    exit 1
fi

# Create destination
mkdir -p "$CLAUDEUP_HOME"

# Step 1: Move library directory
if [[ "$has_old_library" == true ]]; then
    echo "Moving library..."
    mv "$OLD_LIBRARY" "$NEW_LIBRARY"
    info "Moved .library/ -> $NEW_LIBRARY"
else
    warn "No .library/ directory found, skipping"
fi

# Step 2: Move enabled.json
if [[ "$has_old_config" == true ]]; then
    mv "$OLD_CONFIG" "$NEW_CONFIG"
    info "Moved enabled.json -> $NEW_CONFIG"
else
    warn "No enabled.json found, skipping"
fi

# Step 3: Recreate symlinks as absolute paths
echo
echo "Updating symlinks to absolute paths..."
fixed=0
removed=0

for category in "${CATEGORIES[@]}"; do
    cat_dir="$CLAUDE_DIR/$category"
    [[ -d "$cat_dir" ]] || continue

    # Process symlinks at all depths
    while IFS= read -r -d '' symlink; do
        if [[ -L "$symlink" ]]; then
            old_target=$(readlink "$symlink")

            # Check if this symlink pointed into the old .library
            if [[ "$old_target" == *".library"* ]]; then
                # Compute the expected absolute target in the new location
                # Old relative: ../.library/hooks/my-hook.sh -> new absolute: $NEW_LIBRARY/hooks/my-hook.sh
                # Extract the category/item portion after .library/
                relative_part="${old_target#*".library/"}"
                new_target="$NEW_LIBRARY/$relative_part"

                if [[ -e "$new_target" ]]; then
                    rm "$symlink"
                    ln -s "$new_target" "$symlink"
                    fixed=$((fixed + 1))
                else
                    warn "Target missing for $symlink -> $new_target (removing broken symlink)"
                    rm "$symlink"
                    removed=$((removed + 1))
                fi
            fi
        fi
    done < <(find "$cat_dir" -type l -print0 2>/dev/null)

    # Clean up empty group directories left behind
    find "$cat_dir" -mindepth 1 -type d -empty -delete 2>/dev/null || true
done

if [[ "$fixed" -gt 0 ]]; then
    info "Updated $fixed symlink(s) to absolute paths"
fi
if [[ "$removed" -gt 0 ]]; then
    warn "Removed $removed broken symlink(s)"
fi
if [[ "$fixed" -eq 0 && "$removed" -eq 0 ]]; then
    info "No symlinks needed updating"
fi

# Summary
echo
bold "Migration complete!"
echo "  Library:  $NEW_LIBRARY"
echo "  Config:   $NEW_CONFIG"
echo
echo "Run 'claudeup doctor' to verify everything looks good."
