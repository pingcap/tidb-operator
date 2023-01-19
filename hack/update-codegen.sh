#!/usr/bin/env bash

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

ROOT=$(unset CDPATH && cd $(dirname "${BASH_SOURCE[0]}")/.. && pwd)
cd $ROOT

source hack/lib.sh
hack::ensure_codegen

# `--output-base $ROOT` will output generated code to current dir
GOBIN=$OUTPUT_BIN bash $ROOT/hack/generate-groups.sh "deepcopy,client,informer,lister" \
    github.com/pingcap/tidb-operator/pkg/client \
    github.com/pingcap/tidb-operator/pkg/apis \
    pingcap:v1alpha1 \
    --output-base $ROOT \
    --go-header-file ./hack/boilerplate/boilerplate.generatego.txt

# then we merge generated code with our code base and clean up
cp -r github.com/pingcap/tidb-operator/pkg $ROOT && rm -rf github.com
hack::ensure_go117
