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

source "${ROOT}/hack/lib.sh"

crd_target="$ROOT/manifests/crd.yaml"

# Ensure that we find the binaries we build before anything else.
export GOBIN="${OUTPUT_BIN}"
PATH="${GOBIN}:${PATH}"

# Enable go modules explicitly.
export GO111MODULE=on
go install github.com/pingcap/tidb-operator/cmd/to-crdgen

to-crdgen generate tidbcluster > $crd_target
to-crdgen generate dmcluster >> $crd_target
to-crdgen generate backup >> $crd_target
to-crdgen generate restore >> $crd_target
to-crdgen generate backupschedule >> $crd_target
to-crdgen generate tidbmonitor >> $crd_target
to-crdgen generate tidbinitializer >> $crd_target
to-crdgen generate tidbclusterautoscaler >> $crd_target

hack::ensure_gen_crd_api_references_docs

DOCS_PATH="$ROOT/docs/api-references"

${DOCS_BIN} \
-config "$DOCS_PATH/config.json" \
-template-dir "$DOCS_PATH/template" \
-api-dir "github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1" \
-out-file "$DOCS_PATH/docs.md"
