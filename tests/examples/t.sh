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

function t::tc_is_ready() {
    local ns="$1"
    local name="$2"
    kubectl -n $ns wait --for=condition=Ready --timeout 10s tc/$name
    if [ $? -ne 0 ]; then
        echo "info: cluster $ns/$name is not ready, status: "
        kubectl -n $ns get tc/$name -o wide
        return 1
    fi
    return 0
}

function t::crds_are_ready() {
    for name in $@; do
        local established=$(kubectl get crd $name -o json | jq '.status["conditions"][] | select(.type == "Established") | .status')
        if [ $? -ne 0 ]; then
            echo "error: crd $name is not found"
            return 1
        fi
        if [[ "$established" != "True" ]]; then
            echo "error: crd $name is not ready"
            return 1
        fi
    done
    return 0
}

function t::ns_is_active() {
    local ns="$1"
    local phase=$(kubectl get ns $ns -ojsonpath='{.status.phase}')
    [[ "$phase" == "Active" ]]
}

function t::deploy_is_ready() {
    local ns="$1"
    local name="$2"
    read a b <<<$(kubectl -n $ns get deploy/$name -ojsonpath='{.spec.replicas} {.status.readyReplicas}{"\n"}')
    if [[ "$a" -gt 0 && "$a" -eq "$b" ]]; then
        echo "info: all pods of deployment $ns/$name are ready (desired: $a, ready: $b)"
        return 0
    fi
    echo "info: pods of deployment $ns/$name (desired: $a, ready: $b)"
    return 1
}

