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

TARGET_DIR=${ROOT}/manifests/crd
VERIFY_TMP_DIR=$(mktemp -d)
trap "rm -rf $VERIFY_TMP_DIR" EXIT

cp -R "$TARGET_DIR/." "$VERIFY_TMP_DIR"

$ROOT/hack/update-crd.sh

for version in v1beta1 v1; do
    for file in `ls $TARGET_DIR/$version`; do
        targetFile=$TARGET_DIR/$version/$file
        verifyFile=$VERIFY_TMP_DIR/$version/$file
        echo "diffing $targetFile with $verifyFile" >&2
        diff=$(diff "$targetFile" "$verifyFile") || true
        if [[ -n "${diff}" ]]; then
            echo "${diff}" >&2
            echo >&2
            echo "Run ./hack/update-crd.sh" >&2
            exit 1
        fi
    done
done
