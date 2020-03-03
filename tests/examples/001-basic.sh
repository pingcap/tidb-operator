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

function cleanup() {
    kubectl delete -f examples/basic/tidb-cluster.yaml 
}

trap cleanup EXIT

function checkReplicas() {
    local pdDesiredReplicas="$1"
    local tikvDesiredReplicas="$2"
    local tidbDesiredReplicas="$3"
    local pdReplicas=$(kubectl get tc basic -ojsonpath='{.status.pd.statefulSet.readyReplicas}')
    if [[ "$pdReplicas" != "$pdDesiredReplicas" ]]; then
        echo "info: got pd replicas $pdReplicas, expects $pdDesiredReplicas"
        return 1
    fi
    local tikvReplicas=$(kubectl get tc basic -ojsonpath='{.status.tikv.statefulSet.readyReplicas}')
    if [[ "$tikvReplicas" != "$tikvDesiredReplicas" ]]; then
        echo "info: got tikv replicas $tikvReplicas, expects $tikvDesiredReplicas"
        return 1
    fi
    local tidbReplicas=$(kubectl get tc basic -ojsonpath='{.status.tidb.statefulSet.readyReplicas}')
    if [[ "$tidbReplicas" != "$tidbDesiredReplicas" ]]; then
        echo "info: got tidb replicas $tidbReplicas, expects $tidbDesiredReplicas"
        return 1
    fi
    echo "info: pd replicas $pdReplicas, tikv replicas $tikvReplicas, tidb replicas $tidbReplicas"
    return 0
}

kubectl apply -f examples/basic/tidb-cluster.yaml 

hack::wait_for_success 600 3 "checkReplicas 3 3 2"
