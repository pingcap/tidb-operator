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

ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd -P)

# source only once
[[ $(type -t repo::loaded) == function ]] && return 0

source "$ROOT/hack/lib/vars.sh"

# Shared by upgrade e2e and feature-log lint. stdout contains only the checkout
# path so callers can capture it; progress and Git diagnostics go to stderr.
function repo::fetch() {
    local ref="${V_REPO_REF}"
    local sanitized_ref="${ref//\//-}"
    local repo_dir="${V_REPO_DIR}"
    if [[ -z "$ref" || "$ref" == -* || "$sanitized_ref" == "." || "$sanitized_ref" == ".." ]]; then
        echo "Invalid repo ref: $ref" >&2
        return 1
    fi

    if [[ ! -d "${repo_dir}/.git" ]]; then
        echo "Initializing repo in ${repo_dir}" >&2
        git init --quiet -- "$repo_dir" >&2 || return 1
        git -C "$repo_dir" remote add origin "$V_REPO_URL" >&2 || return 1
    else
        git -C "$repo_dir" remote set-url origin "$V_REPO_URL" >&2 || return 1
    fi

    # Fetch the requested branch, tag or SHA on every call. Never reuse a stale
    # checkout after a fetch failure, and support exact PR base commits.
    echo "Fetching revision ${ref} into ${repo_dir}" >&2
    git -C "$repo_dir" fetch --depth=1 origin "$ref" >&2 || return 1
    git -C "$repo_dir" checkout --detach --force FETCH_HEAD >&2 || return 1
    # Drop untracked source left by generators, preserving ignored build caches.
    git -C "$repo_dir" clean -fd >&2 || return 1
    git -C "$repo_dir" rev-parse HEAD >&2 || return 1
    (cd "$repo_dir" && pwd -P)
}

# marker function
function repo::loaded() {
    return 0
}
