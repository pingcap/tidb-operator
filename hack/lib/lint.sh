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
[[ $(type -t lint::loaded) == function ]] && return 0

source "$ROOT/hack/lib/vars.sh"
source "$ROOT/hack/lib/repo.sh"

function lint::usage() {
    echo "Usage: hack/lint.sh [golangci|feature-log ...]"
    echo "With no lint names, runs all checks."
}

function lint::run() {
    local golangci=0
    local feature_log=0
    local selected=0
    while [[ $# -gt 0 ]]; do
        case "$1" in
            golangci)
                golangci=1
                selected=1
                ;;
            feature-log)
                feature_log=1
                selected=1
                ;;
            -h|--help)
                lint::usage
                return 0
                ;;
            *)
                echo "Unknown lint argument: $1" >&2
                lint::usage >&2
                return 1
                ;;
        esac
        shift
    done
    if [[ $selected -eq 0 ]]; then
        golangci=1
        feature_log=1
    fi
    if [[ $golangci -eq 1 ]]; then
        (cd "$ROOT" && "${V_BIN}/golangci-lint" run -v ./...) || return 1
    fi
    if [[ $feature_log -eq 1 ]]; then
        repo::fetch || return 1
        "${V_BIN}/feature-log-lint" --root="$ROOT" --base-root="$V_REPO_DIR" || return 1
    fi
}

# marker function
function lint::loaded() {
    return 0
}
