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
[[ $(type -t verify::loaded) == function ]] && return 0

ROOT=$(cd $(dirname "${BASH_SOURCE[0]}")/../..; pwd -P)

source $ROOT/hack/lib/util.sh

# NOTE: this function is forked from kubernetes/hack/lib/verify-generated.sh#kube::verify:generated
#
# This function verifies whether generated files are up-to-date. The first two
# parameters are messages that get printed to stderr when changes are found,
# the rest are the function or command and its parameters for generating files
# in the work tree.
#
# Example: kube::verify::generated "Mock files are out of date" "Please run 'hack/update-mocks.sh'" hack/update-mocks.sh
function verify::generated() {
  ( # a subshell prevents environment changes from leaking out of this function
    local failure_header=$1
    shift
    local failure_tail=$1
    shift

    util::ensure_clean_working_dir

    local tmpdir="$(mktemp -d -t "tidb-operator-verify.XXXXXX")"
    git worktree add -f -q "${tmpdir}" HEAD
    util::trap_add "git worktree remove -f ${tmpdir}" EXIT
    cd "${tmpdir}"

    echo "cmd: $@"
    # Update generated files.
    "$@"

    # Test for diffs
    diffs=$(git status --porcelain | wc -l)
    if [[ ${diffs} -gt 0 ]]; then
      if [[ -n "${failure_header}" ]]; then
        echo "${failure_header}" >&2
      fi
      git status >&2
      git diff >&2
      if [[ -n "${failure_tail}" ]]; then
        echo "" >&2
        echo "${failure_tail}" >&2
      fi
      return 1
    fi
  )
}


# marker function
function verify::loaded() {
  return 0
}
