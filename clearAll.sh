#!/bin/bash

# MC-IAM-Manager Cleanup Script

# =============================================================================
# Usage Function
# =============================================================================
usage() {
    echo "Usage: $0 [OPTIONS]"
    echo ""
    echo "Options:"
    echo "  -f, --full    Full cleanup: stop containers, remove volumes and all generated data"
    echo "  -h, --help    Display this help message"
    echo ""
    echo "Without options (interactive):"
    echo "  1. Stop only   — docker compose down (containers/networks removed, data preserved)"
    echo "  2. Full reset  — docker compose down -v + remove container-volume/ and .env"
    echo ""
    echo "Examples:"
    echo "  $0          # Interactive mode"
    echo "  $0 --full   # Non-interactive full reset"
    exit 1
}

# =============================================================================
# Parameter Parsing
# =============================================================================
CLEAR_MODE=""

while [[ $# -gt 0 ]]; do
    case $1 in
        -f|--full)
            CLEAR_MODE="full"
            shift
            ;;
        -h|--help)
            usage
            ;;
        *)
            echo "Unknown option: $1"
            usage
            ;;
    esac
done

# =============================================================================
# Locate project root (script must be in project root)
# =============================================================================
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT_ABS="$SCRIPT_DIR"

cd "$PROJECT_ROOT_ABS" || {
    echo "Error: Cannot change to project root: $PROJECT_ROOT_ABS"
    exit 1
}

# =============================================================================
# Mode Selection (interactive if not specified)
# =============================================================================
if [ -z "$CLEAR_MODE" ]; then
    echo "=========================================="
    echo "MC-IAM-Manager Cleanup"
    echo "=========================================="
    echo ""
    echo "Select cleanup level:"
    echo ""
    echo "  1. Stop only"
    echo "     - Stops and removes containers and networks"
    echo "     - Database data and certificates are preserved"
    echo "     - Re-run 'docker compose up -d' to restart"
    echo ""
    echo "  2. Full reset"
    echo "     - Stops and removes containers, networks, and named volumes"
    echo "     - Deletes container-volume/ (DB data, certs, nginx config)"
    echo "     - Deletes .env"
    echo "     - Re-run installAll.sh to set up again from scratch"
    echo ""
    echo "=========================================="

    while true; do
        echo -n "Select cleanup level (1: Stop only, 2: Full reset): "
        read -r choice
        case $choice in
            1) CLEAR_MODE="stop"; break ;;
            2) CLEAR_MODE="full"; break ;;
            *) echo "Invalid selection. Please enter 1 or 2." ;;
        esac
    done
fi

# =============================================================================
# Execute Cleanup
# =============================================================================
case $CLEAR_MODE in
    stop)
        echo ""
        echo "Stopping containers..."
        docker compose down
        echo ""
        echo "✓ Containers stopped. Data in container-volume/ is preserved."
        echo "  To restart: docker compose up -d"
        ;;

    full)
        echo ""
        echo "⚠️  Full reset will permanently delete:"
        echo "   - All named Docker volumes (database data)"
        echo "   - container-volume/ (nginx config, certificates, DB files)"
        echo "   - .env (restored to repo defaults if tracked by git)"
        echo ""
        echo -n "Are you sure? (yes/N): "
        read -r confirm
        if [ "$confirm" != "yes" ]; then
            echo "Aborted."
            exit 0
        fi

        echo ""
        echo "Stopping containers and removing volumes..."
        docker compose down -v

        echo "Removing generated data..."
        if [ -d "$PROJECT_ROOT_ABS/container-volume" ]; then
            if command -v sudo >/dev/null 2>&1; then
                sudo rm -rf "$PROJECT_ROOT_ABS/container-volume"
            else
                rm -rf "$PROJECT_ROOT_ABS/container-volume"
            fi
            echo "✓ container-volume/ removed."
        fi

        # Restore .env: if tracked by git, restore defaults; otherwise delete
        if git -C "$PROJECT_ROOT_ABS" ls-files --error-unmatch .env >/dev/null 2>&1; then
            git -C "$PROJECT_ROOT_ABS" checkout -- .env
            echo "✓ .env restored to repo defaults."
        elif [ -f "$PROJECT_ROOT_ABS/.env" ]; then
            rm -f "$PROJECT_ROOT_ABS/.env"
            echo "✓ .env removed."
        fi

        echo ""
        echo "✓ Full reset complete. Re-run './installAll.sh' to set up again."
        ;;
esac
