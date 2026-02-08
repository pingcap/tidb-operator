#!/usr/bin/env bash
# Copyright 2024 PingCAP, Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# This script updates go modules dependencies.
#
# Usage:
#   ./hack/update-modules.sh [options]
#
# Options:
#   -p, --patch-only PREFIX    Only update patch version for modules matching PREFIX (can be specified multiple times)
#   -a, --allowed PREFIX       Only update modules matching PREFIX (can be specified multiple times, includes both direct and indirect)
#   -x, --denied PREFIX        Exclude modules matching PREFIX (can be specified multiple times, takes precedence over allowed)
#   -d, --dir DIR              Directory containing go.mod to update (can be specified multiple times, default: all go.mod files)
#   -n, --dry-run              Show what would be done without making changes
#   -h, --help                 Show this help message
#
# Examples:
#   # Update all modules in all go.mod files
#   ./hack/update-modules.sh
#
#   # Update all modules, but only patch versions for k8s.io and sigs.k8s.io
#   ./hack/update-modules.sh -p k8s.io -p sigs.k8s.io
#
#   # Update only specific directories
#   ./hack/update-modules.sh -d . -d ./api
#
#   # Only update modules matching specific prefixes (both direct and indirect)
#   ./hack/update-modules.sh -a golang.org -a github.com/aws
#
#   # Exclude specific modules from updates
#   ./hack/update-modules.sh -a golang.org -x golang.org/x/net
#
#   # Combine options
#   ./hack/update-modules.sh -p k8s.io -p sigs.k8s.io -a golang.org -x golang.org/x/net -d .

set -o errexit
set -o nounset
set -o pipefail

ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")"/..; pwd -P)

source "$ROOT/hack/lib/modules.sh"

# Default values
# PATCH_ONLY_PREFIXES: Modules matching these prefixes will only get patch version updates.
# These prefixes should also be included in ALLOWED_PREFIXES to be updated.
PATCH_ONLY_PREFIXES=(
    "k8s.io/"
    "sigs.k8s.io/"
    "helm.sh/helm/v3"
)
# ALLOWED_PREFIXES: Only modules matching these prefixes will be updated.
# Modules also matching PATCH_ONLY_PREFIXES will get patch-only updates.
ALLOWED_PREFIXES=(
    "k8s.io/"
    "sigs.k8s.io/"
    "helm.sh/helm/v3"
    "golang.org/"
    "github.com/aws/aws-sdk-go-v2/"
    "github.com/aws/smithy-go"
    "github.com/Azure/azure-sdk-for-go/"
    "github.com/onsi/ginkgo/v2"
    "github.com/onsi/gomega"
    "go.uber.org/mock"
    "github.com/golangci/golangci-lint/v2"
    "github.com/apache/skywalking-eyes"
)
# DENIED_PREFIXES: Modules matching these prefixes will NOT be updated, even if they match ALLOWED_PREFIXES.
DENIED_PREFIXES=(
    "k8s.io/gengo/v2"
    "k8s.io/kube-openapi"
    "k8s.io/utils"
)
DIRS=()
DRY_RUN=false

# Print usage
usage() {
    sed -n '/^# Usage:/,/^[^#]/p' "$0" | grep '^#' | sed 's/^# \?//'
    exit 0
}

# Parse command line arguments
parse_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            -p|--patch-only)
                if [[ -z "${2:-}" ]]; then
                    modules::error "Option -p/--patch-only requires a PREFIX argument"
                    usage
                fi
                PATCH_ONLY_PREFIXES+=("$2")
                shift 2
                ;;
            -a|--allowed)
                if [[ -z "${2:-}" ]]; then
                    modules::error "Option -a/--allowed requires a PREFIX argument"
                    usage
                fi
                ALLOWED_PREFIXES+=("$2")
                shift 2
                ;;
            -x|--denied)
                if [[ -z "${2:-}" ]]; then
                    modules::error "Option -x/--denied requires a PREFIX argument"
                    usage
                fi
                DENIED_PREFIXES+=("$2")
                shift 2
                ;;
            -d|--dir)
                if [[ -z "${2:-}" ]]; then
                    modules::error "Option -d/--dir requires a directory argument"
                    usage
                fi
                DIRS+=("$2")
                shift 2
                ;;
            -n|--dry-run)
                DRY_RUN=true
                shift
                ;;
            -h|--help)
                usage
                ;;
            *)
                modules::error "Unknown option: $1"
                usage
                ;;
        esac
    done
}

# Find all go.mod directories
find_go_mod_dirs() {
    find "$ROOT" -name "go.mod" -type f | while read -r mod_file; do
        dirname "$mod_file"
    done
}

# Main function
main() {
    parse_args "$@"

    modules::info "Starting go modules update"
    if [[ "$DRY_RUN" == "true" ]]; then
        modules::warn "Running in dry-run mode, no changes will be made"
    fi

    # Print configuration
    if [[ ${#PATCH_ONLY_PREFIXES[@]} -gt 0 ]]; then
        modules::info "Patch-only prefixes:"
        for prefix in "${PATCH_ONLY_PREFIXES[@]}"; do
            echo "    - $prefix"
        done
    fi

    if [[ ${#ALLOWED_PREFIXES[@]} -gt 0 ]]; then
        modules::info "Allowed module prefixes (filter):"
        for prefix in "${ALLOWED_PREFIXES[@]}"; do
            echo "    - $prefix"
        done
    else
        modules::info "No allowed prefixes specified, updating all direct dependencies"
    fi

    if [[ ${#DENIED_PREFIXES[@]} -gt 0 ]]; then
        modules::info "Denied module prefixes (excluded):"
        for prefix in "${DENIED_PREFIXES[@]}"; do
            echo "    - $prefix"
        done
    fi

    # Determine directories to update
    local dirs_to_update=()
    if [[ ${#DIRS[@]} -eq 0 ]]; then
        modules::info "Finding all go.mod files..."
        while IFS= read -r dir; do
            dirs_to_update+=("$dir")
        done < <(find_go_mod_dirs)
    else
        for dir in "${DIRS[@]}"; do
            if [[ "$dir" = /* ]]; then
                dirs_to_update+=("$dir")
            else
                dirs_to_update+=("$ROOT/$dir")
            fi
        done
    fi

    modules::info "Directories to update:"
    for dir in "${dirs_to_update[@]}"; do
        echo "    - $dir"
    done
    echo

    # Convert arrays to space-separated strings for passing to function
    local patch_prefixes_str="${PATCH_ONLY_PREFIXES[*]}"
    local allowed_prefixes_str="${ALLOWED_PREFIXES[*]}"
    local denied_prefixes_str="${DENIED_PREFIXES[*]:-}"

    # Update each directory
    local failed_dirs=()
    for dir in "${dirs_to_update[@]}"; do
        if ! modules::update_dir "$dir" "$DRY_RUN" "$patch_prefixes_str" "$allowed_prefixes_str" "$denied_prefixes_str"; then
            failed_dirs+=("$dir")
        fi
        echo
    done

    # Summary
    if [[ ${#failed_dirs[@]} -gt 0 ]]; then
        modules::warn "Some directories failed to update:"
        for dir in "${failed_dirs[@]}"; do
            echo "    - $dir"
        done
        exit 1
    fi

    modules::success "All modules updated successfully!"
}

main "$@"
