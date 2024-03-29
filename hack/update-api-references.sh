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

hack::ensure_gen_crd_api_references_docs

DOCS_PATH="$ROOT/docs/api-references"

echo "Generating API references docs ..."
API_DIR="${ROOT}/pkg/apis/pingcap/v1alpha1"

pushd ${API_DIR} >/dev/null
    GOROOT=$(go env GOROOT) ${DOCS_BIN} \
        -config "$DOCS_PATH/config.json" \
        -template-dir "$DOCS_PATH/template" \
        -api-dir "github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1" \
        -out-file "$DOCS_PATH/docs.md"
popd >/dev/null

echo "Generating API references docs for federation ..."
API_DIR="${ROOT}/pkg/apis/federation/pingcap/v1alpha1"

pushd ${API_DIR} >/dev/null
    GOROOT=$(go env GOROOT) ${DOCS_BIN} \
        -config "$DOCS_PATH/config.json" \
        -template-dir "$DOCS_PATH/template" \
        -api-dir "github.com/pingcap/tidb-operator/pkg/apis/federation/pingcap/v1alpha1" \
        -out-file "$DOCS_PATH/federation-docs.md"
popd >/dev/null
