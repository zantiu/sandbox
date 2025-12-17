#!/bin/bash
#
# sync-to-wfm.sh - Synchronize desired deployment state to WFM
#
# Usage:
#   ./scripts/sync-to-wfm.sh [OPTIONS]
#
# Options:
#   -D, --dir DIR        Path to deployments directory (default: deployments/)
#   -d, --dry-run        Show what would be done without making changes
#   -v, --verbose        Enable verbose output
#   -h, --help           Show this help message
#
# Environment Variables (required):
#   EXPOSED_SYMPHONY_IP    WFM Symphony IP address
#   EXPOSED_SYMPHONY_PORT  WFM Symphony port (default: 8082)
#   EXPOSED_HARBOR_IP      Harbor registry IP (for env substitution)
#   EXPOSED_HARBOR_PORT    Harbor registry port (default: 8081)
#
# Each .yaml file in the deployments directory represents one deployment.
# This script is additive-only: it creates/updates deployments but never deletes them.
# Safe to run multiple times (idempotent).
#
set -euo pipefail

# Script directory and repo root
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

# Default configuration
DEPLOYMENTS_DIR="${REPO_ROOT}/deployments"
DRY_RUN=false
VERBOSE=false

# WFM Configuration from environment
EXPOSED_SYMPHONY_IP="${EXPOSED_SYMPHONY_IP:-}"
EXPOSED_SYMPHONY_PORT="${EXPOSED_SYMPHONY_PORT:-8082}"
EXPOSED_HARBOR_IP="${EXPOSED_HARBOR_IP:-}"
EXPOSED_HARBOR_PORT="${EXPOSED_HARBOR_PORT:-8081}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
log_info()  { echo -e "${BLUE}[INFO]${NC} $*"; }
log_ok()    { echo -e "${GREEN}[OK]${NC} $*"; }
log_warn()  { echo -e "${YELLOW}[WARN]${NC} $*" >&2; }
log_error() { echo -e "${RED}[ERROR]${NC} $*" >&2; }
log_debug() { [[ "$VERBOSE" == "true" ]] && echo -e "[DEBUG] $*" || true; }

# Show usage
usage() {
    head -22 "$0" | grep "^#" | sed 's/^# *//'
}

# Parse command line arguments
parse_args() {
    while [[ $# -gt 0 ]]; do
        case "$1" in
            -D|--dir)
                DEPLOYMENTS_DIR="$2"
                shift 2
                ;;
            -d|--dry-run)
                DRY_RUN=true
                shift
                ;;
            -v|--verbose)
                VERBOSE=true
                shift
                ;;
            -h|--help)
                usage
                exit 0
                ;;
            *)
                log_error "Unknown option: $1"
                usage
                exit 1
                ;;
        esac
    done
}

# Check required tools and environment
validate_environment() {
    local errors=0

    # Check required tools
    for tool in yq jq curl; do
        if ! command -v "$tool" &> /dev/null; then
            log_error "Required tool not found: $tool"
            ((errors++))
        fi
    done

    # Check required environment variables
    if [[ -z "$EXPOSED_SYMPHONY_IP" ]]; then
        log_error "EXPOSED_SYMPHONY_IP environment variable is required"
        ((errors++))
    fi

    # Check deployments directory exists
    if [[ ! -d "$DEPLOYMENTS_DIR" ]]; then
        log_error "Deployments directory not found: $DEPLOYMENTS_DIR"
        ((errors++))
    fi

    if [[ $errors -gt 0 ]]; then
        log_error "Environment validation failed with $errors error(s)"
        exit 1
    fi

    log_debug "Environment validated successfully"
}

# Substitute environment variables in a string
substitute_env_vars() {
    local content="$1"
    # Use envsubst if available, otherwise use sed
    if command -v envsubst &> /dev/null; then
        echo "$content" | envsubst
    else
        # Manual substitution for common variables
        echo "$content" | \
            sed "s|\${EXPOSED_HARBOR_IP}|${EXPOSED_HARBOR_IP}|g" | \
            sed "s|\${EXPOSED_HARBOR_PORT}|${EXPOSED_HARBOR_PORT}|g" | \
            sed "s|\${EXPOSED_SYMPHONY_IP}|${EXPOSED_SYMPHONY_IP}|g" | \
            sed "s|\${EXPOSED_SYMPHONY_PORT}|${EXPOSED_SYMPHONY_PORT}|g"
    fi
}

# WFM API base URL
get_api_base() {
    echo "https://${EXPOSED_SYMPHONY_IP}:${EXPOSED_SYMPHONY_PORT}/non-margo/wfm-nbi/api/v0"
}

# API GET request
api_get() {
    local endpoint="$1"
    curl -sk -X GET "$(get_api_base)${endpoint}" \
        -H "Content-Type: application/json" \
        2>/dev/null
}

# API POST request
api_post() {
    local endpoint="$1"
    local data="$2"
    curl -sk -X POST "$(get_api_base)${endpoint}" \
        -H "Content-Type: application/json" \
        -d "$data" \
        2>/dev/null
}

# Get list of existing packages from WFM
get_existing_packages() {
    api_get "/app-packages" | jq -r '.items // []' 2>/dev/null || echo "[]"
}

# Get list of existing deployments from WFM
get_existing_deployments() {
    api_get "/app-deployments" | jq -r '.items // []' 2>/dev/null || echo "[]"
}

# Get list of devices from WFM
get_devices() {
    api_get "/devices" | jq -r '.items // []' 2>/dev/null || echo "[]"
}

# Check if a package exists by name
package_exists() {
    local package_name="$1"
    local packages
    packages=$(get_existing_packages)
    echo "$packages" | jq -e --arg name "$package_name" \
        'map(select(.metadata.name == $name)) | length > 0' > /dev/null 2>&1
}

# Get package ID by name
get_package_id() {
    local package_name="$1"
    get_existing_packages | jq -r --arg name "$package_name" \
        'map(select(.metadata.name == $name)) | .[0].metadata.id // empty'
}

# Check if deployment exists by name
deployment_exists() {
    local deployment_name="$1"
    local deployments
    deployments=$(get_existing_deployments)
    echo "$deployments" | jq -e --arg name "$deployment_name" \
        'map(select(.metadata.name == $name)) | length > 0' > /dev/null 2>&1
}

# Find devices matching labels
find_devices_by_labels() {
    local labels_json="$1"
    get_devices | jq -r --argjson labels "$labels_json" '
        map(select(
            .metadata.labels as $dev_labels |
            ($labels | to_entries | all(
                .key as $k | .value as $v |
                $dev_labels[$k] == $v
            ))
        )) | .[].metadata.id
    ' 2>/dev/null
}

# Generate ApplicationDeployment manifest JSON
generate_deployment_manifest() {
    local name="$1"
    local package_id="$2"
    local device_id="$3"
    local profile_json="$4"
    local parameters_json="$5"

    local profile_type
    profile_type=$(echo "$profile_json" | jq -r '.type')

    local components
    components=$(echo "$profile_json" | jq -c '.components // []')

    # Build the manifest
    local manifest
    manifest=$(cat <<EOF
{
  "apiVersion": "non-margo.org",
  "kind": "ApplicationDeployment",
  "metadata": {
    "name": "${name}"
  },
  "spec": {
    "appPackageRef": {
      "id": "${package_id}"
    },
    "deviceRef": {
      "id": "${device_id}"
    },
    "deploymentProfile": {
      "type": "${profile_type}",
      "components": ${components}
    }
  }
}
EOF
)

    # Add parameters if present
    if [[ "$parameters_json" != "null" && "$parameters_json" != "{}" ]]; then
        manifest=$(echo "$manifest" | jq --argjson params "$parameters_json" '.spec.parameters = $params')
    fi

    echo "$manifest"
}

# Process a single deployment file
process_deployment_file() {
    local file_path="$1"
    local file_name
    file_name=$(basename "$file_path")

    log_info "Processing file: $file_name"

    # Read the deployment YAML
    local deployment_yaml
    deployment_yaml=$(cat "$file_path")

    # Extract deployment name
    local deployment_name
    deployment_name=$(echo "$deployment_yaml" | yq -r '.name')

    if [[ -z "$deployment_name" || "$deployment_name" == "null" ]]; then
        log_warn "  Skipping: no 'name' field in $file_name"
        return 1
    fi

    log_debug "  Deployment name: $deployment_name"

    # Extract and validate package
    local package_name
    package_name=$(echo "$deployment_yaml" | yq -r '.package.name')
    if [[ -z "$package_name" || "$package_name" == "null" ]]; then
        log_warn "  Skipping: no package name specified"
        return 1
    fi

    # Check if package exists in WFM
    if ! package_exists "$package_name"; then
        log_warn "  Skipping: package not found in WFM: $package_name"
        log_warn "  Ensure package is uploaded to Harbor and onboarded to WFM"
        return 1
    fi

    local package_id
    package_id=$(get_package_id "$package_name")
    log_debug "  Package ID: $package_id"

    # Resolve device reference
    local device_id
    device_id=$(echo "$deployment_yaml" | yq -r '.device.id // empty')
    device_id=$(substitute_env_vars "$device_id")

    if [[ -z "$device_id" ]]; then
        # Try label-based selection
        local device_labels_json
        device_labels_json=$(echo "$deployment_yaml" | yq -o=json '.device.labels // {}')
        if [[ "$device_labels_json" != "{}" && "$device_labels_json" != "null" ]]; then
            local matching_devices
            matching_devices=$(find_devices_by_labels "$device_labels_json")

            if [[ -z "$matching_devices" ]]; then
                log_warn "  Skipping: no devices match labels"
                return 1
            fi

            # For label-based selection, deploy to each matching device
            local device_count=0
            while IFS= read -r dev_id; do
                if [[ -n "$dev_id" ]]; then
                    local unique_name="${deployment_name}-${dev_id:0:8}"
                    process_single_deployment "$deployment_yaml" "$package_id" "$dev_id" "$unique_name"
                    ((device_count++))
                fi
            done <<< "$matching_devices"
            log_info "  Processed for $device_count device(s) matching labels"
            return 0
        else
            log_warn "  Skipping: no device ID or labels specified"
            return 1
        fi
    fi

    # Single device deployment
    process_single_deployment "$deployment_yaml" "$package_id" "$device_id" "$deployment_name"
}

# Process deployment for a single device
process_single_deployment() {
    local deployment_yaml="$1"
    local package_id="$2"
    local device_id="$3"
    local deployment_name="$4"

    log_debug "  Device ID: $device_id"

    # Check if deployment already exists
    if deployment_exists "$deployment_name"; then
        log_ok "  Already exists: $deployment_name (skipping)"
        return 0
    fi

    # Extract and substitute profile
    local profile_json
    profile_json=$(echo "$deployment_yaml" | yq -o=json '.profile')
    profile_json=$(substitute_env_vars "$profile_json")

    # Extract and substitute parameters
    local parameters_json
    parameters_json=$(echo "$deployment_yaml" | yq -o=json '.parameters // {}')
    parameters_json=$(substitute_env_vars "$parameters_json")

    # Generate manifest
    local manifest
    manifest=$(generate_deployment_manifest "$deployment_name" "$package_id" "$device_id" "$profile_json" "$parameters_json")

    if [[ "$DRY_RUN" == "true" ]]; then
        log_info "  [DRY-RUN] Would create deployment: $deployment_name"
        if [[ "$VERBOSE" == "true" ]]; then
            echo "$manifest" | jq .
        fi
        return 0
    fi

    # Create deployment via API
    local response
    response=$(api_post "/app-deployments" "$manifest")

    # Check response
    local result_id
    result_id=$(echo "$response" | jq -r '.metadata.id // empty' 2>/dev/null)

    if [[ -n "$result_id" ]]; then
        log_ok "  Created deployment: $deployment_name (ID: $result_id)"
        return 0
    else
        local error_msg
        error_msg=$(echo "$response" | jq -r '.message // .error // "Unknown error"' 2>/dev/null)
        log_error "  Failed to create deployment: $deployment_name"
        log_error "  Response: $error_msg"
        return 1
    fi
}

# Main sync function
sync() {
    log_info "=========================================="
    log_info "GitOps Sync to WFM"
    log_info "=========================================="
    log_info "Deployments directory: $DEPLOYMENTS_DIR"
    log_info "WFM endpoint: $(get_api_base)"
    log_info "Dry run: $DRY_RUN"
    log_info ""

    # Find all deployment YAML files (exclude README.md and other non-yaml)
    local deployment_files=()
    while IFS= read -r -d '' file; do
        deployment_files+=("$file")
    done < <(find "$DEPLOYMENTS_DIR" -maxdepth 1 -name "*.yaml" -type f -print0 2>/dev/null | sort -z)

    local deployment_count=${#deployment_files[@]}

    if [[ $deployment_count -eq 0 ]]; then
        log_warn "No deployment files (*.yaml) found in $DEPLOYMENTS_DIR"
        log_info "Create deployment files and commit to trigger sync"
        return 0
    fi

    log_info "Found $deployment_count deployment file(s) to process"
    log_info ""

    # Process each deployment file
    local total=0
    local success=0
    local skipped=0
    local failed=0

    for file in "${deployment_files[@]}"; do
        ((total++))

        if process_deployment_file "$file"; then
            ((success++))
        else
            ((skipped++))
        fi
        echo ""
    done

    # Summary
    log_info "=========================================="
    log_info "Sync Summary"
    log_info "=========================================="
    log_info "Total processed: $total"
    log_ok   "Successful: $success"
    if [[ $skipped -gt 0 ]]; then
        log_warn "Skipped: $skipped"
    fi
    if [[ $failed -gt 0 ]]; then
        log_error "Failed: $failed"
    fi

    if [[ $failed -gt 0 ]]; then
        return 1
    fi
    return 0
}

# Main entry point
main() {
    parse_args "$@"
    validate_environment
    sync
}

main "$@"
