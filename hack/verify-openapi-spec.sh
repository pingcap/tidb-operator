#!/usr/bin/env bash

# Copyright 2020 PingCAP, Inc.
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
cd $ROOT

target="pkg/apis/pingcap/v1alpha1/openapi_generated.go"
verify_tmp=$(mktemp)
trap "rm -f $verify_tmp" EXIT
cp "$target" "${verify_tmp}"

targetFed="pkg/apis/federation/pingcap/v1alpha1/openapi_generated.go"
verifyFed_tmp=$(mktemp)
trap "rm -f $verifyFed_tmp" EXIT
cp "$targetFed" "${verifyFed_tmp}"

hack/update-openapi-spec.sh

echo "diffing $target with $verify_tmp" >&2
diff=$(diff "$target" "$verify_tmp") || true
if [[ -n "${diff}" ]]; then
    echo "${diff}" >&2
    echo >&2
    echo "Run ./hack/update-openapi-spec.sh" >&2
    exit 1
fi

echo "diffing $targetFed with $verifyFed_tmp" >&2
diff=$(diff "$targetFed" "$verifyFed_tmp") || true
if [[ -n "${diff}" ]]; then
    echo "${diff}" >&2
    echo >&2
    echo "Run ./hack/update-openapi-spec.sh" >&2
    exit 1
fi
