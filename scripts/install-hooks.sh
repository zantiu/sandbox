#!/bin/bash
#
# install-hooks.sh - Install git hooks for GitOps sync
#
# This script installs a post-commit hook that automatically syncs
# deployments to WFM when changes are committed to deployments/
#
# Usage:
#   ./scripts/install-hooks.sh          # Install hooks
#   ./scripts/install-hooks.sh --remove # Remove hooks
#
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
HOOKS_DIR="${REPO_ROOT}/.git/hooks"
HOOK_FILE="${HOOKS_DIR}/post-commit"
LOG_DIR="${REPO_ROOT}/logs"

# Colors
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
RED='\033[0;31m'
NC='\033[0m'

log_info()  { echo -e "${GREEN}[INFO]${NC} $*"; }
log_warn()  { echo -e "${YELLOW}[WARN]${NC} $*"; }
log_error() { echo -e "${RED}[ERROR]${NC} $*" >&2; }

# Check if we're in a git repository
check_git_repo() {
    if [[ ! -d "${REPO_ROOT}/.git" ]]; then
        log_error "Not a git repository: ${REPO_ROOT}"
        exit 1
    fi
}

# Install the post-commit hook
install_hook() {
    log_info "Installing GitOps post-commit hook..."

    # Create hooks directory if it doesn't exist
    mkdir -p "$HOOKS_DIR"
    mkdir -p "$LOG_DIR"

    # Check if hook already exists
    if [[ -f "$HOOK_FILE" ]]; then
        if grep -q "sync-to-wfm" "$HOOK_FILE"; then
            log_warn "GitOps hook already installed"
            log_info "To reinstall, run: $0 --remove && $0"
            return 0
        else
            log_warn "Existing post-commit hook found"
            log_warn "Backing up to ${HOOK_FILE}.backup"
            cp "$HOOK_FILE" "${HOOK_FILE}.backup"
        fi
    fi

    # Create the hook
    cat > "$HOOK_FILE" << 'HOOK_CONTENT'
#!/bin/bash
#
# GitOps post-commit hook
# Syncs deployments to WFM when deployments/ files are changed
#

REPO_ROOT="$(git rev-parse --show-toplevel)"
LOG_FILE="${REPO_ROOT}/logs/gitops-sync.log"
SYNC_SCRIPT="${REPO_ROOT}/scripts/sync-to-wfm.sh"

# Check if any files in deployments/ were changed in this commit
CHANGED_FILES=$(git diff-tree --no-commit-id --name-only -r HEAD 2>/dev/null || true)

if echo "$CHANGED_FILES" | grep -q "^deployments/"; then
    echo ""
    echo "=========================================="
    echo "GitOps: Detected changes in deployments/"
    echo "=========================================="

    # Check if sync script exists
    if [[ ! -x "$SYNC_SCRIPT" ]]; then
        echo "[WARN] Sync script not found or not executable: $SYNC_SCRIPT"
        exit 0
    fi

    # Check if required environment variables are set
    if [[ -z "${EXPOSED_SYMPHONY_IP:-}" ]]; then
        echo "[WARN] EXPOSED_SYMPHONY_IP not set - skipping sync"
        echo "[INFO] Run: source pipeline/wfm.env"
        exit 0
    fi

    # Create log directory if needed
    mkdir -p "$(dirname "$LOG_FILE")"

    # Run sync and log output
    echo "[INFO] Running sync to WFM..."
    echo ""

    if "$SYNC_SCRIPT" 2>&1 | tee -a "$LOG_FILE"; then
        echo ""
        echo "[OK] Sync completed successfully"
    else
        echo ""
        echo "[WARN] Sync completed with warnings or errors"
        echo "[INFO] Check log: $LOG_FILE"
    fi

    echo "=========================================="
else
    # No deployments/ changes, skip silently
    :
fi
HOOK_CONTENT

    chmod +x "$HOOK_FILE"

    log_info "Hook installed successfully: $HOOK_FILE"
    log_info ""
    log_info "Usage:"
    log_info "  1. Set environment: source pipeline/wfm.env"
    log_info "  2. Edit deployments/desired-state.yaml"
    log_info "  3. Commit changes: git commit -m 'Deploy app'"
    log_info "  4. Hook will auto-sync to WFM"
    log_info ""
    log_info "To remove: $0 --remove"
}

# Remove the hook
remove_hook() {
    log_info "Removing GitOps post-commit hook..."

    if [[ -f "$HOOK_FILE" ]]; then
        if grep -q "sync-to-wfm" "$HOOK_FILE"; then
            rm "$HOOK_FILE"
            log_info "Hook removed: $HOOK_FILE"

            # Restore backup if it exists
            if [[ -f "${HOOK_FILE}.backup" ]]; then
                mv "${HOOK_FILE}.backup" "$HOOK_FILE"
                log_info "Restored previous hook from backup"
            fi
        else
            log_warn "Hook file exists but is not the GitOps hook"
            log_warn "Not removing: $HOOK_FILE"
        fi
    else
        log_info "No hook to remove"
    fi
}

# Main
main() {
    check_git_repo

    case "${1:-}" in
        --remove|-r)
            remove_hook
            ;;
        --help|-h)
            echo "Usage: $0 [--remove]"
            echo ""
            echo "Options:"
            echo "  --remove, -r   Remove the installed hook"
            echo "  --help, -h     Show this help"
            ;;
        *)
            install_hook
            ;;
    esac
}

main "$@"
