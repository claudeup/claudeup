#!/bin/bash
# ABOUTME: Interactive setup wizard for the hobson profile
# ABOUTME: Guides users through category-based plugin selection for wshobson/agents

set -e

# Verify claudeup is available
if ! command -v claudeup &> /dev/null; then
    echo "Error: claudeup command not found"
    echo "Please ensure claudeup is installed and in your PATH"
    exit 1
fi

# Check if gum is available for interactive UI
HAS_GUM=false
if command -v gum &> /dev/null; then
    HAS_GUM=true
fi

# Marketplace suffix for plugins
MARKETPLACE="wshobson-agents"

# All available categories
ALL_CATEGORIES=(
    "Core Development"
    "Quality & Testing"
    "AI & Machine Learning"
    "Infrastructure & DevOps"
    "Security & Compliance"
    "Data & Databases"
    "Languages"
    "Business & Specialty"
)

# Category definitions - maps theme names to plugin lists
declare -A CATEGORIES
CATEGORIES["Core Development"]="code-documentation debugging-toolkit git-pr-workflows backend-development frontend-mobile-development full-stack-orchestration code-refactoring dependency-management error-debugging team-collaboration documentation-generation c4-architecture multi-platform-apps developer-essentials"
CATEGORIES["Quality & Testing"]="unit-testing tdd-workflows code-review-ai comprehensive-review performance-testing-review framework-migration codebase-cleanup"
CATEGORIES["AI & Machine Learning"]="llm-application-dev agent-orchestration context-management machine-learning-ops"
CATEGORIES["Infrastructure & DevOps"]="deployment-strategies deployment-validation kubernetes-operations cloud-infrastructure cicd-automation incident-response error-diagnostics distributed-debugging observability-monitoring"
CATEGORIES["Security & Compliance"]="security-scanning security-compliance backend-api-security frontend-mobile-security"
CATEGORIES["Data & Databases"]="data-engineering data-validation-suite database-design database-migrations application-performance database-cloud-optimization"
CATEGORIES["Languages"]="python-development javascript-typescript systems-programming jvm-languages web-scripting functional-programming julia-development arm-cortex-microcontrollers shell-scripting"
CATEGORIES["Business & Specialty"]="api-scaffolding api-testing-observability seo-content-creation seo-technical-optimization seo-analysis-monitoring business-analytics hr-legal-compliance customer-sales-automation content-marketing blockchain-web3 quantitative-trading payment-processing game-development accessibility-compliance"

# Category descriptions for display
declare -A CATEGORY_DESCRIPTIONS
CATEGORY_DESCRIPTIONS["Core Development"]="workflows, debugging, docs, refactoring"
CATEGORY_DESCRIPTIONS["Quality & Testing"]="code review, testing, cleanup"
CATEGORY_DESCRIPTIONS["AI & Machine Learning"]="LLM dev, agents, MLOps"
CATEGORY_DESCRIPTIONS["Infrastructure & DevOps"]="K8s, cloud, CI/CD, monitoring"
CATEGORY_DESCRIPTIONS["Security & Compliance"]="scanning, compliance, API security"
CATEGORY_DESCRIPTIONS["Data & Databases"]="ETL, schema design, migrations"
CATEGORY_DESCRIPTIONS["Languages"]="Python, JS/TS, Go, Rust, etc."
CATEGORY_DESCRIPTIONS["Business & Specialty"]="SEO, analytics, blockchain, gaming"

# Selected categories (stored as array)
declare -a SELECTED_CATEGORIES_ARRAY=()

# Print header
print_header() {
    echo ""
    echo "╭─────────────────────────────────────────────────────╮"
    echo "│       Welcome to the Hobson Profile Setup           │"
    echo "╰─────────────────────────────────────────────────────╯"
    echo ""
}

# Check if category is selected
is_selected() {
    local category="$1"
    for selected in "${SELECTED_CATEGORIES_ARRAY[@]}"; do
        if [[ "$selected" == "$category" ]]; then
            return 0
        fi
    done
    return 1
}

# Print category list with checkboxes
print_categories() {
    local idx=1
    for category in "${ALL_CATEGORIES[@]}"; do
        local marker="[ ]"
        if is_selected "$category"; then
            marker="[x]"
        fi
        local desc="${CATEGORY_DESCRIPTIONS[$category]}"
        printf "  %d. %s %-25s (%s)\n" "$idx" "$marker" "$category" "$desc"
        ((idx++))
    done
}

# Toggle category selection
toggle_category() {
    local category="$1"
    if is_selected "$category"; then
        # Remove category
        local new_array=()
        for selected in "${SELECTED_CATEGORIES_ARRAY[@]}"; do
            if [[ "$selected" != "$category" ]]; then
                new_array+=("$selected")
            fi
        done
        SELECTED_CATEGORIES_ARRAY=("${new_array[@]}")
    else
        # Add category
        SELECTED_CATEGORIES_ARRAY+=("$category")
    fi
}

# Interactive selection using gum
select_with_gum() {
    local options=()

    for category in "${ALL_CATEGORIES[@]}"; do
        local desc="${CATEGORY_DESCRIPTIONS[$category]}"
        options+=("$category ($desc)")
    done

    echo "Select development categories (space to toggle, enter to confirm):"
    echo ""

    local selected
    selected=$(gum choose --no-limit "${options[@]}" 2>/dev/null || true)

    if [[ -z "$selected" ]]; then
        return 1
    fi

    # Parse selected options back to category names
    SELECTED_CATEGORIES_ARRAY=()
    while IFS= read -r line; do
        # Extract category name before the parenthesis
        local cat_name="${line%% (*}"
        if [[ -n "$cat_name" ]]; then
            SELECTED_CATEGORIES_ARRAY+=("$cat_name")
        fi
    done <<< "$selected"

    return 0
}

# Interactive selection using basic prompts (fallback)
select_with_prompts() {
    while true; do
        clear
        print_header
        echo "Select development categories (enter numbers to toggle):"
        echo ""
        print_categories
        echo ""
        local max_idx=${#ALL_CATEGORIES[@]}
        echo "Commands: [1-$max_idx] Toggle category  [a] Select all  [n] Select none"
        echo "          [c] Customize plugins  [Enter] Continue  [q] Quit"
        echo ""
        read -r -p "> " input

        case "$input" in
            [1-9]|[1-9][0-9])
                # Validate bounds before accessing array
                if [[ "$input" -ge 1 && "$input" -le "$max_idx" ]]; then
                    local idx=$((input - 1))
                    # Double-check index is valid (defensive)
                    if [[ $idx -ge 0 && $idx -lt ${#ALL_CATEGORIES[@]} ]]; then
                        toggle_category "${ALL_CATEGORIES[$idx]}"
                    fi
                fi
                ;;
            a|A)
                SELECTED_CATEGORIES_ARRAY=("${ALL_CATEGORIES[@]}")
                ;;
            n|N)
                SELECTED_CATEGORIES_ARRAY=()
                ;;
            c|C)
                if [[ ${#SELECTED_CATEGORIES_ARRAY[@]} -eq 0 ]]; then
                    echo "Please select at least one category first."
                    sleep 1
                else
                    return 2  # Signal to customize
                fi
                ;;
            q|Q)
                echo "Setup cancelled."
                exit 0
                ;;
            "")
                if [[ ${#SELECTED_CATEGORIES_ARRAY[@]} -eq 0 ]]; then
                    echo "Please select at least one category."
                    sleep 1
                else
                    return 0  # Continue without customizing
                fi
                ;;
        esac
    done
}

# Enable plugins for selected categories
# Returns: 0 if all plugins enabled successfully, 1 if any failed
enable_plugins() {
    local plugins_to_enable=()

    # Collect all plugins from selected categories
    for category in "${SELECTED_CATEGORIES_ARRAY[@]}"; do
        local category_plugins="${CATEGORIES[$category]}"
        for plugin in $category_plugins; do
            plugins_to_enable+=("$plugin")
        done
    done

    if [[ ${#plugins_to_enable[@]} -eq 0 ]]; then
        echo "No plugins to enable."
        return 0
    fi

    echo ""
    echo "Enabling ${#plugins_to_enable[@]} plugins..."
    echo ""

    local success=0
    local failed=0

    for plugin in "${plugins_to_enable[@]}"; do
        local full_name="${plugin}@${MARKETPLACE}"
        local error_output
        if error_output=$(claudeup enable "$full_name" 2>&1); then
            echo "  ✓ $full_name"
            ((success++))
        else
            echo "  ✗ $full_name"
            # Show first line of error for context
            local first_error
            first_error=$(echo "$error_output" | head -1)
            if [[ -n "$first_error" ]]; then
                echo "    → $first_error"
            fi
            ((failed++))
        fi
    done

    echo ""
    echo "Setup complete!"
    echo "  Enabled: $success plugins"
    if [[ $failed -gt 0 ]]; then
        echo "  Failed:  $failed plugins"
        echo ""
        echo "Run 'claudeup status' to see your configuration."
        return 1
    fi
    echo ""
    echo "Run 'claudeup status' to see your configuration."
    return 0
}

# Main execution
main() {
    print_header

    local selection_result=0

    if [[ "$HAS_GUM" == "true" ]]; then
        if ! select_with_gum; then
            echo "No categories selected. Setup cancelled."
            exit 0
        fi
    else
        select_with_prompts
        selection_result=$?

        if [[ $selection_result -eq 2 ]]; then
            echo "Per-plugin customization coming soon..."
        fi
    fi

    # Enable plugins and exit with appropriate code
    if ! enable_plugins; then
        exit 1
    fi
}

main "$@"
