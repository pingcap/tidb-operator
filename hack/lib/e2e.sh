#!/usr/bin/env bash
# Copyright 2024 PingCAP, Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -o errexit
set -o nounset
set -o pipefail

ROOT=$(
    cd $(dirname "${BASH_SOURCE[0]}")/../..
    pwd -P
)

source $ROOT/hack/lib/vars.sh
source $ROOT/hack/lib/kind.sh
source $ROOT/hack/lib/image.sh

OUTPUT_DIR=$ROOT/_output
KIND=$OUTPUT_DIR/bin/kind
KUBECTL=$OUTPUT_DIR/bin/kubectl
GINKGO=$OUTPUT_DIR/bin/ginkgo
GENERATEJWT=$OUTPUT_DIR/bin/generate_jwt

CI=${CI:-""}

function e2e::ensure_kubectl() {
    if ! command -v $KUBECTL &>/dev/null; then
        echo "kubectl not found, installing..."
        curl -L -o $KUBECTL https://dl.k8s.io/release/v1.30.2/bin/${V_OS}/${V_ARCH}/kubectl
        chmod +x $KUBECTL
    fi
}

function e2e::switch_kube_context() {
    echo "switching to kind context ${V_KIND_CLUSTER}"
    $KUBECTL config use-context kind-${V_KIND_CLUSTER}
}

function e2e::ensure_cert_manager() {
    echo "checking if cert-manager is installed..."
    if $KUBECTL -n cert-manager get deployment cert-manager &>/dev/null; then
        echo "cert-manager already installed, skipping..."
        return
    fi

    echo "installing cert-manager..."
    $KUBECTL apply --server-side=true -f https://github.com/cert-manager/cert-manager/releases/download/v1.15.2/cert-manager.yaml

    echo "waiting for cert-manager to be ready..."
    $KUBECTL -n cert-manager wait --for=condition=Available --timeout=5m deployment/cert-manager
}

function e2e::install_crds() {
    echo "installing CRDs..."
    $KUBECTL apply --server-side=true -f $ROOT/manifests/crd
}

function e2e::install_operator() {
    echo "installing operator..."
    $KUBECTL -n $V_DEPLOY_NAMESPACE apply --server-side=true -f $OUTPUT_DIR/manifests/tidb-operator.yaml

    echo "waiting for operator to be ready..."
    $KUBECTL -n $V_DEPLOY_NAMESPACE wait --for=condition=Available --timeout=5m deployment/tidb-operator
}

function e2e::uninstall_operator() {
    echo "checking if operator is installed..."
    if ! $KUBECTL -n $V_DEPLOY_NAMESPACE get deployment tidb-operator &>/dev/null; then
        echo "operator not found, skipping uninstall..."
        return
    fi

    echo "uninstalling operator..."
    $KUBECTL -n $V_DEPLOY_NAMESPACE delete -f $OUTPUT_DIR/manifests/tidb-operator.yaml

    echo "waiting for operator to be deleted..."
    $KUBECTL -n $V_DEPLOY_NAMESPACE wait --for=delete --timeout=5m deployment/tidb-operator
}

function e2e::reload_testing_workload() {
    image::build testing-workload --push
}

function e2e::install_ginkgo() {
    if ! command -v $GINKGO &>/dev/null; then
        echo "ginkgo not found, installing..."
        $ROOT/hack/download.sh go_install $GINKGO github.com/onsi/ginkgo/v2/ginkgo
    fi
}

# generate_jwt is a tool to generate JWT token for `tidb_auth_token` test
# ref: https://docs.pingcap.com/tidb/stable/security-compatibility-with-mysql#tidb_auth_token
function e2e::install_generate_jwt() {
    if ! command -v $GENERATEJWT &>/dev/null; then
        echo "generate_jwt not found, installing..."
        $ROOT/hack/download.sh go_install $GENERATEJWT github.com/cbcwestwolf/generate_jwt@latest
    fi
}

function e2e::run() {
    if [[ "$CI" == "true" ]]; then
        echo "running e2e tests in CI mode with options: $*"
        $GINKGO -v -r --timeout=2h --randomize-all --randomize-suites --fail-on-empty --keep-going --race --trace --flake-attempts=3 "$*" "$ROOT/tests/e2e/..."
    else
        echo "running e2e tests locally..."
        $GINKGO -r -v "$@" "$ROOT/tests/e2e/..."
    fi
}

function e2e::prepare() {
    e2e::install_ginkgo
    e2e::install_generate_jwt
    e2e::ensure_kubectl
    kind::ensure_cluster
    e2e::switch_kube_context
    e2e::ensure_cert_manager

    e2e::install_crds

    # build the operator image and load it into the kind cluster
    image::build prestop-checker tidb-operator testing-workload tidb-backup-manager --push
    e2e::uninstall_operator
    e2e::install_operator

    image:prepare
}

function e2e::reinstall_operator() {
    image::build tidb-operator --push
    e2e::uninstall_operator
    e2e::install_operator
}

function e2e::reinstall_backup_manager() {
    image::build tidb-backup-manager --push
}

function e2e::e2e() {
    local ginkgo_opts=()
    local prepare=0
    local run=0
    local reinstall_operator=0
    local reinstall_backup_manager=0
    while [[ $# -gt 0 ]]; do
        case $1 in
        --prepare)
            prepare=1
            shift
            ;;
        --reinstall-operator)
            reinstall_operator=1
            shift
            ;;
        --reinstall-backup-manager)
            reinstall_backup_manager=1
            shift
            ;;
        --reload-testing-workload)
            reload_testing_workload=1
            shift
            ;;
        run)
            run=1
            shift
            ;;
        *)
            if [[ $run -eq 1 ]]; then
                ginkgo_opts+=("${1}")
            else
                echo "Unknown option $1"
                exit 1
            fi
            shift
            ;;
        esac
    done

    if [[ $prepare -eq 1 ]]; then
        e2e::prepare
    elif [[ $reinstall_operator -eq 1 ]]; then
        e2e::reinstall_operator
    elif [[ $reinstall_backup_manager -eq 1 ]]; then
        e2e::reinstall_backup_manager
    elif [[ $reload_testing_workload -eq 1 ]]; then
        e2e::reload_testing_workload
    fi
    if [[ $run -eq 1 ]]; then
        e2e::run "${ginkgo_opts[@]}"
    fi
}
