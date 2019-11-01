#!/bin/bash -e

# Copyright 2017 PingCAP, Inc.
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

SCRIPT_ROOT=$(dirname "${BASH_SOURCE[0]}")/..
cd $SCRIPT_ROOT

export GO111MODULE=on

go mod vendor
# workaround for https://github.com/golang/go/issues/31248
git checkout go.mod || true

CODEGEN_PKG=${CODEGEN_PKG:-$(cd "${SCRIPT_ROOT}"; ls -d -1 ./vendor/k8s.io/code-generator 2>/dev/null || echo ../code-generator)}

GO111MODULE=off bash "${CODEGEN_PKG}"/generate-groups.sh "deepcopy,client,informer,lister" \
    github.com/pingcap/tidb-operator/pkg/client \
    github.com/pingcap/tidb-operator/pkg/apis \
    pingcap:v1alpha1 \
    --go-header-file "${SCRIPT_ROOT}"/hack/boilerplate.go.txt
