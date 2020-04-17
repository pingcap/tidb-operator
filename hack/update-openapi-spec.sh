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

# Ensure that we find the binaries we build before anything else.
export GOBIN="${OUTPUT_BIN}"
PATH="${GOBIN}:${PATH}"

# Enable go modules explicitly.
export GO111MODULE=on
go install k8s.io/code-generator/cmd/openapi-gen

openapi-gen --go-header-file=./hack/boilerplate/boilerplate.generatego.txt \
    -i github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1,k8s.io/apimachinery/pkg/apis/meta/v1,k8s.io/api/core/v1 \
    -p apis/pingcap/v1alpha1 -O openapi_generated -o ./pkg
