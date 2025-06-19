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

ROOT=$(cd $(dirname "${BASH_SOURCE[0]}")/..; pwd -P)

# download
#   go_install
#   github.com/golangci/golangci-lint/cmd/golangci-lint
#   v1.59.1
#   "version --format=short"
function download() {
    local type=$1
    shift

    case $type in
    go_install)
        go_install "$@"
        ;;
    *)
        echo "unknown download type: $type"
        exit 1
        ;;
    esac
}

function go_install() {
    local output=$1
    local path=$2
    local version=${3:-""}
    local cond=${4:-""}

    if [[ -n $cond && -f $output ]]; then
        local curVersion=$(eval "$output $cond")
        if [[ $curVersion == $version ]]; then
            echo "$output@$version has been installed"
            return
        else
            echo "$output@$curVersion is out dated, try to re-install"
        fi
    fi

    local pkgPath=$path
    if [[ -n $version ]]; then
        echo "Install $output with version $version"
        pkgPath=$path@$version
    else
        echo "Install $output"
    fi

    local output_dir=$(dirname $output)
    mkdir -p $output_dir

    local tmp_dir=$(mktemp -d)

    # TODO: define a var presents absolute path of go command
    GOBIN=${tmp_dir} go install -v $pkgPath
    mv ${tmp_dir}/* $output
    rm -rf $tmp_dir
}

download "$@"
