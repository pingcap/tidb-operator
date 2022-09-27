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

source hack/lib.sh

hack::ensure_golangci_lint

# main module
${OUTPUT_BIN}/golangci-lint run --timeout 10m $(go list ./... | sed 's|github.com/pingcap/tidb-operator/||')

# sub modules
for dir in ${GO_SUBMODULE_DIRS[@]}; do
    pushd "${ROOT}/${dir}" >/dev/null
        ${OUTPUT_BIN}/golangci-lint run --timeout 10m
    popd >/dev/null
done
