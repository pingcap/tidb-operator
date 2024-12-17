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

ROOT=$(cd $(dirname "${BASH_SOURCE[0]}")/../..; pwd -P)

source $ROOT/hack/lib/vars.sh
source $ROOT/hack/lib/version.sh

OUTPUT_DIR=$ROOT/_output

function build::go() {
    local target=$1
    local os=${2:-""}
    local arch=${3:-""}

    CGO_ENABLED=0 \
    GOOS=${os} \
    GOARCH=${arch} \
    go build -v \
        -ldflags "$(version::ldflags)" \
        -o ${OUTPUT_DIR}/${os}/${arch}/bin/${target} \
        ${ROOT}/cmd/${target}/main.go
}

function build::all() {
    local targets=()
    while [[ $# -gt 0 ]]; do
        targets+=("$1")
        shift
    done
    if [[ ${#targets[@]} -eq 0 ]]; then
        targets=("operator" "prestop-checker")
    fi

    local platforms
    IFS=, read -ra platforms <<< "${V_PLATFORMS}"

    for target in ${targets[@]}; do
        if [[ ${#platforms[@]} -eq 0 ]]; then
            build::go $target
        else
            for platform in ${platforms[@]}; do
                case $platform in
                linux/arm64)
                    build::go $target linux arm64 ;;
                linux/amd64)
                    build::go $target linux amd64 ;;
                *)
                    echo "unsupported platform ${platform}"
                    exit 1
                    ;;
                esac
            done
        fi
    done
}
