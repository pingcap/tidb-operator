#!/usr/bin/env bash

# Copyright 2021 PingCAP, Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# See the License for the specific language governing permissions and
# limitations under the License.

set -o errexit
set -o nounset
set -o pipefail

ROOT=$(unset CDPATH && cd $(dirname "${BASH_SOURCE[0]}")/.. && pwd)
TMP="$ROOT/_tmp"

cd $ROOT
source hack/lib.sh

cleanup() {
    rm -f $TMP
}
trap "cleanup" EXIT SIGINT
cleanup

ignored_modules=(
    "github.com/pingcap/tidb-operator/pkg/apis"
)

# check if all packages in submodule is same as main module
for sub_mod in ${GO_SUBMODULE_DIRS[@]}; do
    echo "verifying submodule ${sub_mod} ..."
    dir="$ROOT/$sub_mod"

    pushd $dir >/dev/null
        go mod edit -json | jq -r ".Require[] | .Path" | grep -v -F ${ignored_modules[@]} | while read path; do
            ver_in_sub=$(cd $dir && go list -m -f '{{.Version}}' $path)
            ver_in_main=$(cd $ROOT && go list -mod=readonly -m -f '{{with .Replace}} {{- .Version}} {{else}} {{- .Version}} {{end}}' ${path})
            if [ $ver_in_sub != $ver_in_main ]; then
                touch $TMP
                echo "\"$path\" version in \"$sub_mod\" isn't same as main: $ver_in_sub != $ver_in_main"
            fi
        done
    popd >/dev/null
done

if [ -f $TMP ]; then
    exit 1
fi
