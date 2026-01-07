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

# This script downloads Grafana dashboard JSON files from TiDB ecosystem repositories.
# It clones each repository at the specified tag/branch and copies the dashboard files
# to the destination directory.
#
# Usage: ./get-grafana-dashboards.sh [tag] [destination]
#   tag:         Git tag or branch name (default: master)
#   destination: Output directory for JSON files (default: current directory)
#
# Example:
#   ./get-grafana-dashboards.sh v8.0.0 ./dashboards

set -euo pipefail

# Parse arguments with defaults
tag=${1:-master}
dst=${2:-.}
tmp="./tmp"

# cleanup removes the temporary directory on script exit.
cleanup() {
    if [ -d "$tmp" ]; then
        rm -rf "$tmp"
    fi
}
trap cleanup EXIT INT TERM

mkdir -p "$dst"
mkdir -p "$tmp"

# clone_repo clones a git repository with shallow depth.
# Arguments:
#   $1 - Repository URL
#   $2 - Target directory
#   $3 - Branch or tag name
clone_repo() {
    local url="$1"
    local dir="$2"
    local branch="$3"

    echo "Cloning $(basename "$url") (branch: $branch)..."
    if ! git clone --depth 1 -b "$branch" "$url" "$dir"; then
        echo "Error: Failed to clone $url (branch: $branch)" >&2
        exit 1
    fi
}

# copy_if_exists copies JSON files from source directory to destination.
# Safely handles non-existent directories and empty directories.
# Arguments:
#   $1 - Source directory path
#   $2 - Destination directory path
copy_if_exists() {
    local src="$1"
    local dest="$2"
    if [ ! -d "$src" ]; then
        echo "Warning: $src does not exist, skipping" >&2
        return 0
    fi

    # Use find to avoid glob expansion failure when no files match
    local file_count
    file_count=$(find "$src" -maxdepth 1 -name "*.json" -type f | wc -l | tr -d ' ')

    if [ "$file_count" -eq 0 ]; then
        echo "Warning: No JSON files found in $src, skipping" >&2
        return 0
    fi

    find "$src" -maxdepth 1 -name "*.json" -type f -exec cp {} "$dest/" \;
    echo "Copied $file_count JSON file(s) from $src"
}

# Download dashboards from each TiDB ecosystem repository

clone_repo "https://github.com/pingcap/tidb.git" "$tmp/tidb" "$tag"
# The path changed in different versions, try both paths
copy_if_exists "$tmp/tidb/pkg/metrics/grafana" "$dst"
copy_if_exists "$tmp/tidb/metrics/grafana" "$dst"
copy_if_exists "$tmp/tidb/br/metrics/grafana" "$dst"

clone_repo "https://github.com/tikv/pd.git" "$tmp/pd" "$tag"
copy_if_exists "$tmp/pd/metrics/grafana" "$dst"

clone_repo "https://github.com/tikv/tikv.git" "$tmp/tikv" "$tag"
copy_if_exists "$tmp/tikv/metrics/grafana" "$dst"

clone_repo "https://github.com/pingcap/tiflash.git" "$tmp/tiflash" "$tag"
copy_if_exists "$tmp/tiflash/metrics/grafana" "$dst"

clone_repo "https://github.com/pingcap/tiflow.git" "$tmp/tiflow" "$tag"
copy_if_exists "$tmp/tiflow/metrics/grafana" "$dst"

tiproxy_branch="$tag"
if [[ "$tag" == "master" ]]; then
    tiproxy_branch="main"
fi
clone_repo "https://github.com/pingcap/TiProxy.git" "$tmp/TiProxy" "$tiproxy_branch"
copy_if_exists "$tmp/TiProxy/pkg/metrics/grafana" "$dst"

clone_repo "https://github.com/pingcap/ticdc.git" "$tmp/ticdc" "$tag"
copy_if_exists "$tmp/ticdc/metrics/grafana" "$dst"

echo "All Grafana dashboards have been downloaded to $dst"
