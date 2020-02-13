#!/bin/bash

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

ROOT=$(unset CDPATH && cd $(dirname "${BASH_SOURCE[0]}")/../.. && pwd)
cd $ROOT

source "${ROOT}/hack/lib.sh"
source "${ROOT}/tests/examples/t.sh"

NS=$(basename ${0%.*})

function cleanup() {
    kubectl -n $NS delete -f examples/basic/tidb-cluster.yaml
    kubectl delete ns $NS
}

trap cleanup EXIT

kubectl create ns $NS
hack::wait_for_success 10 3 "t::ns_is_active $NS"

kubectl -n $NS apply -f examples/basic/tidb-cluster.yaml

hack::wait_for_success 600 3 "t::tc_is_ready $NS basic"
