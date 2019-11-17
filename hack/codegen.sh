#!/usr/bin/env bash

# Copyright 2019 PingCAP, Inc.
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

CODEGEN_PKG=${CODEGEN_PKG:-$(ls -d -1 ./vendor/k8s.io/code-generator 2>/dev/null || echo ../code-generator)}

# Generate deepcopy of v1alpha1
GO111MODULE=off bash "${CODEGEN_PKG}"/generate-groups.sh "deepcopy" \
    github.com/pingcap/tidb-operator/pkg/client \
    github.com/pingcap/tidb-operator/pkg/apis \
    pingcap:v1alpha1 \
    --go-header-file ./hack/boilerplate.go.txt

# Generate api registries of v1alpha2
bash ./hack/generate-apiregistry.sh github.com/pingcap/tidb-operator/pkg/apis \
    --go-header-file ./hack/boilerplate.go.txt

# Generate deepcopy, defaulter, conversion and openapi of v1alpha2
GO111MODULE=off bash "${CODEGEN_PKG}"/generate-internal-groups.sh "deepcopy,defaulter,conversion,openapi" \
    github.com/pingcap/tidb-operator/pkg/client \
    github.com/pingcap/tidb-operator/pkg/apis \
    github.com/pingcap/tidb-operator/pkg/apis \
    "pingcap:v1alpha2" \
    --go-header-file ./hack/boilerplate.go.txt

# Generate the clients of v1alpha1 and v1alpha2 together
GO111MODULE=off bash "${CODEGEN_PKG}"/generate-groups.sh "client,informer,lister" \
    github.com/pingcap/tidb-operator/pkg/client \
    github.com/pingcap/tidb-operator/pkg/apis \
    "pingcap:v1alpha1,v1alpha2" \
    --go-header-file ./hack/boilerplate.go.txt
