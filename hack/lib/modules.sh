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

set -o errexit
set -o nounset
set -o pipefail

# source only once
[[ $(type -t modules::loaded) == function ]] && return 0

# Check for required dependencies
if ! command -v jq &> /dev/null; then
    echo "Error: jq is required but not installed. Please install jq first." >&2
    exit 1
fi

# Colors for output
readonly MODULES_RED='\033[0;31m'
readonly MODULES_GREEN='\033[0;32m'
readonly MODULES_YELLOW='\033[1;33m'
readonly MODULES_BLUE='\033[0;34m'
readonly MODULES_NC='\033[0m' # No Color

# Print colored output
modules::info() {
    echo -e "${MODULES_BLUE}[INFO]${MODULES_NC} $*"
}

modules::success() {
    echo -e "${MODULES_GREEN}[SUCCESS]${MODULES_NC} $*"
}

modules::warn() {
    echo -e "${MODULES_YELLOW}[WARN]${MODULES_NC} $*"
}

modules::error() {
    echo -e "${MODULES_RED}[ERROR]${MODULES_NC} $*" >&2
}

# Check if a module matches any prefix in the given array
# Args: module prefix1 [prefix2 ...]
modules::match_prefix() {
    local module=$1
    shift
    for prefix in "$@"; do
        if [[ "$module" == "$prefix"* ]]; then
            return 0
        fi
    done
    return 1
}

# Get go version from go.mod (e.g., "1.23")
modules::get_go_version() {
    go mod edit -json | jq -r '.Go // empty' 2>/dev/null || true
}

# Get toolchain version from go.mod (e.g., "go1.23.6"), empty if not set
modules::get_toolchain_version() {
    go mod edit -json | jq -r '.Toolchain // empty' 2>/dev/null || true
}

# Get all direct dependencies from go.mod
modules::get_direct_deps() {
    go mod edit -json | jq -r '.Require[] | select(.Indirect!=true) | "\(.Path)"' 2>/dev/null || true
}

# Get all indirect dependencies from go.mod
modules::get_indirect_deps() {
    go mod edit -json | jq -r '.Require[] | select(.Indirect==true) | "\(.Path)"' 2>/dev/null || true
}

# Update modules in batch
# Set GOPROXY=direct to avoid https://github.com/golang/go/issues/49111
# Args: dry_run version_query modules...
#   - version_query: "latest" or "patch"
modules::update_modules_batch() {
    local dry_run=$1
    local version_query=$2
    shift 2
    local modules=("$@")

    if [[ ${#modules[@]} -eq 0 ]]; then
        return 0
    fi

    # Build module specs with version query (e.g., module@latest or module@patch)
    local module_specs=()
    for module in "${modules[@]}"; do
        module_specs+=("${module}@${version_query}")
    done

    modules::info "  Updating ${#modules[@]} modules (@${version_query}): ${modules[*]}"

    if [[ "$dry_run" == "true" ]]; then
        echo "    [DRY-RUN] GOPROXY=direct go get ${module_specs[*]}"
    else
        if ! GOPROXY=direct go get "${module_specs[@]}" 2>&1; then
            modules::error "    Some modules failed to update"
            return 1
        fi
    fi
}

# Update all modules in a directory
# Args: dir dry_run patch_prefixes_str allowed_prefixes_str denied_prefixes_str
#   - dir: directory containing go.mod
#   - dry_run: "true" or "false"
#   - patch_prefixes_str: space-separated prefixes for patch-only updates
#   - allowed_prefixes_str: space-separated prefixes to filter modules (only update matching modules, empty means all direct deps)
#   - denied_prefixes_str: space-separated prefixes to exclude modules (if a module matches both allowed and denied, it will NOT be updated)
modules::update_dir() {
    local dir=$1
    local dry_run=$2
    local patch_prefixes_str=$3
    local allowed_prefixes_str=$4
    local denied_prefixes_str=${5:-}

    # Convert prefix strings to arrays
    local patch_prefixes=()
    local allowed_prefixes=()
    local denied_prefixes=()
    if [[ -n "$patch_prefixes_str" ]]; then
        read -ra patch_prefixes <<< "$patch_prefixes_str"
    fi
    if [[ -n "$allowed_prefixes_str" ]]; then
        read -ra allowed_prefixes <<< "$allowed_prefixes_str"
    fi
    if [[ -n "$denied_prefixes_str" ]]; then
        read -ra denied_prefixes <<< "$denied_prefixes_str"
    fi

    if [[ ! -f "$dir/go.mod" ]]; then
        modules::error "No go.mod found in $dir"
        return 1
    fi

    modules::info "Updating modules in $dir"

    pushd "$dir" > /dev/null

    # Get go and toolchain version from go.mod
    local go_ver
    local toolchain_ver
    go_ver=$(modules::get_go_version)
    toolchain_ver=$(modules::get_toolchain_version)

    # Get all direct dependencies
    local deps
    deps=$(modules::get_direct_deps)

    # Collect modules into patch and latest groups
    local patch_modules=()
    local latest_modules=()

    # Helper function to categorize a module
    categorize_module() {
        local dep=$1
        # Skip if module matches any denied prefix
        if [[ ${#denied_prefixes[@]} -gt 0 ]] && modules::match_prefix "$dep" "${denied_prefixes[@]}"; then
            modules::info "    Skipping denied module: $dep"
            return 0
        fi
        if [[ ${#patch_prefixes[@]} -gt 0 ]] && modules::match_prefix "$dep" "${patch_prefixes[@]}"; then
            patch_modules+=("$dep")
        else
            latest_modules+=("$dep")
        fi
    }

    if [[ ${#denied_prefixes[@]} -gt 0 ]]; then
        modules::info "  Excluding modules by denied prefixes: ${denied_prefixes[*]}"
    fi

    if [[ ${#allowed_prefixes[@]} -gt 0 ]]; then
        # Filter mode: only update modules matching allowed prefixes (both direct and indirect)
        modules::info "  Filtering modules by allowed prefixes: ${allowed_prefixes[*]}"

        if [[ -n "$deps" ]]; then
            while IFS= read -r dep; do
                [[ -z "$dep" ]] && continue
                if modules::match_prefix "$dep" "${allowed_prefixes[@]}"; then
                    categorize_module "$dep"
                fi
            done <<< "$deps"
        fi

        # Also check indirect dependencies
        local indirect_deps
        indirect_deps=$(modules::get_indirect_deps)
        if [[ -n "$indirect_deps" ]]; then
            while IFS= read -r dep; do
                [[ -z "$dep" ]] && continue
                if modules::match_prefix "$dep" "${allowed_prefixes[@]}"; then
                    categorize_module "$dep"
                fi
            done <<< "$indirect_deps"
        fi
    else
        # No filter: update all direct dependencies
        if [[ -z "$deps" ]]; then
            modules::warn "  No direct dependencies found, skipping direct module updates"
        else
            while IFS= read -r dep; do
                [[ -z "$dep" ]] && continue
                categorize_module "$dep"
            done <<< "$deps"
        fi
    fi

    # Update modules
    if [[ ${#latest_modules[@]} -gt 0 ]]; then
        if ! modules::update_modules_batch "$dry_run" "latest" "${latest_modules[@]}"; then
            popd > /dev/null
            return 1
        fi
    fi
    if [[ ${#patch_modules[@]} -gt 0 ]]; then
        if ! modules::update_modules_batch "$dry_run" "patch" "${patch_modules[@]}"; then
            popd > /dev/null
            return 1
        fi
    fi

    if [[ -n "$go_ver" ]]; then
        modules::info "  Locking go version to $go_ver..."
        if [[ "$dry_run" == "true" ]]; then
            echo "    [DRY-RUN] go get go@$go_ver"
        else
            if ! go get "go@$go_ver"; then
                modules::error "  Failed to lock go version to $go_ver"
                popd > /dev/null
                return 1
            fi
        fi
    else
        modules::error "  Could not read go version from go.mod"
        popd > /dev/null
        return 1
    fi

    if [[ -n "$toolchain_ver" ]]; then
        modules::info "  Locking toolchain version to $toolchain_ver..."
        if [[ "$dry_run" == "true" ]]; then
            echo "    [DRY-RUN] go get toolchain@$toolchain_ver"
        else
            if ! go get "toolchain@$toolchain_ver"; then
                modules::error "  Failed to lock toolchain version to $toolchain_ver"
                popd > /dev/null
                return 1
            fi
        fi
    fi

    # Run go mod tidy after locking go version
    modules::info "  Running go mod tidy..."
    if [[ "$dry_run" == "true" ]]; then
        echo "    [DRY-RUN] go mod tidy"
    else
        if ! go mod tidy; then
            modules::error "  Failed to run go mod tidy"
            popd > /dev/null
            return 1
        fi
    fi

    popd > /dev/null
    modules::success "  Done updating $dir"
}

# marker function
function modules::loaded() {
    return 0
}
