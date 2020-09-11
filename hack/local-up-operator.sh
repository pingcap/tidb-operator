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

#
# This command runs tidb-operator in Kubernetes.
#

set -o errexit
set -o nounset
set -o pipefail

ROOT=$(unset CDPATH && cd $(dirname "${BASH_SOURCE[0]}")/.. && pwd)
cd $ROOT

source "${ROOT}/hack/lib.sh"

function usage() {
    cat <<'EOF'
This commands run tidb-operator in Kubernetes.

Usage: hack/local-up-operator.sh [-hd]

    -h      show this message and exit
    -i      install dependencies only

Environments:

    PROVIDER              Kubernetes provider. Defaults: kind.
    CLUSTER               the name of e2e cluster. Defaults to kind for kind provider.
    KUBECONFIG            path to the kubeconfig file, defaults: ~/.kube/config
    KUBECONTEXT           context in kubeconfig file, defaults to current context
    NAMESPACE             Kubernetes namespace in which we run our tidb-operator.
    DOCKER_REGISTRY       image docker registry
    IMAGE_TAG             image tag
    SKIP_IMAGE_BUILD      skip build and push images

EOF
}

installOnly=false
while getopts "h?i" opt; do
    case "$opt" in
    h|\?)
        usage
        exit 0
        ;;
    i)
      installOnly=true
        ;;
    esac
done

PROVIDER=${PROVIDER:-kind}
CLUSTER=${CLUSTER:-}
KUBECONFIG=${KUBECONFIG:-~/.kube/config}
KUBECONTEXT=${KUBECONTEXT:-}
NAMESPACE=${NAMESPACE:-pingcap}
DOCKER_REGISTRY=${DOCKER_REGISTRY:-localhost:5000}
IMAGE_TAG=${IMAGE_TAG:-latest}
SKIP_IMAGE_BUILD=${SKIP_IMAGE_BUILD:-}

hack::ensure_kubectl
hack::ensure_kind
hack::ensure_helm

if [[ "$installOnly" == "true" ]]; then
    exit 0
fi

function hack::create_namespace() {
    local ns="$1"
    $KUBECTL_BIN create namespace $ns
    for ((i=0; i < 30; i++)); do
        local phase=$(kubectl get ns $ns -ojsonpath='{.status.phase}')
        if [ "$phase" == "Active" ]; then
            return 0
        fi
        sleep 1
    done
    return 1
}

function hack::wait_for_deploy() {
    local ns="$1"
    local name="$2"
    local retries="${3:-300}"
    echo "info: waiting for pods of deployment $ns/$name are ready (retries: $retries, interval: 1s)"
    for ((i = 0; i < retries; i++)) {
        read a b <<<$($KUBECTL_BIN --context $KUBECONTEXT -n $ns get deploy/$name -ojsonpath='{.spec.replicas} {.status.readyReplicas}{"\n"}')
        if [[ "$a" -gt 0 && "$a" -eq "$b" ]]; then
            echo "info: all pods of deployment $ns/$name are ready (desired: $a, ready: $b)"
            return 0
        fi
        echo "info: pods of deployment $ns/$name (desired: $a, ready: $b)"
        sleep 1
    }
    echo "info: timed out waiting for pods of deployment $ns/$name are ready"
    return 1
}

function hack::cluster_exists() {
    local c="$1"
    for n in $($KIND_BIN get clusters); do
        if [ "$n" == "$c" ]; then
            return 0
        fi
    done
    return 1
}

echo "info: checking clusters"

if [ "$PROVIDER" == "kind" ]; then
    if [ -z "$CLUSTER" ]; then
        CLUSTER=kind
    fi
    if ! hack::cluster_exists "$CLUSTER"; then
        echo "error: kind cluster '$CLUSTER' not found, please create it or specify the right cluster name with CLUSTER environment"
        exit 1
    fi
else
    echo "erorr: only kind PROVIDER is supported"
    exit 1
fi

if [ -z "$KUBECONTEXT" ]; then
    KUBECONTEXT=$(kubectl config current-context)
    echo "info: KUBECONTEXT is not set, current context $KUBECONTEXT is used"
fi

if [ -z "$SKIP_IMAGE_BUILD" ]; then
    echo "info: building docker images"
    E2E=y DOCKER_REGISTRY=$DOCKER_REGISTRY IMAGE_TAG=$IMAGE_TAG make docker
else
    echo "info: skip building docker images"
fi

echo "info: loading images into cluster"
images=(
    $DOCKER_REGISTRY/pingcap/tidb-operator:${IMAGE_TAG}
    $DOCKER_REGISTRY/pingcap/tidb-backup-manager:${IMAGE_TAG}
)
for n in ${images[@]}; do
    echo "info: loading image $n"
    $KIND_BIN load docker-image --name $CLUSTER $n
done

echo "info: uninstall tidb-operator"
$KUBECTL_BIN -n "$NAMESPACE" delete deploy -l app.kubernetes.io/name=tidb-operator
$KUBECTL_BIN -n "$NAMESPACE" delete pods -l app.kubernetes.io/name=tidb-operator

echo "info: create namespace '$NAMESPACE' if absent"
if ! $KUBECTL_BIN get ns "$NAMESPACE" &>/dev/null; then
    hack::create_namespace "$NAMESPACE"
fi

echo "info: installing crds"
$KUBECTL_BIN apply -f manifests/crd.yaml

echo "info: deploying tidb-operator"
KUBE_VERSION=$($KUBECTL_BIN version --short | awk '/Server Version:/ {print $3}')
helm_args=(
    template
    --kube-version "$KUBE_VERSION"
    --name tidb-operator-dev
    --namespace "$NAMESPACE"
    --set-string operatorImage=$DOCKER_REGISTRY/pingcap/tidb-operator:${IMAGE_TAG}
    --set-string tidbBackupManagerImage=$DOCKER_REGISTRY/pingcap/tidb-backup-manager:${IMAGE_TAG}
    --set-string controllerManager.logLevel=4
    --set-string scheduler.logLevel=4
)
 
$HELM_BIN ${helm_args[@]} ./charts/tidb-operator/ | kubectl -n "$NAMESPACE" apply -f  -

deploys=(
    tidb-controller-manager 
    tidb-scheduler
)
for deploy in ${deploys[@]}; do
    echo "info: waiting for $NAMESPACE/$deploy to be ready"
	hack::wait_for_deploy "$NAMESPACE" "$deploy"
done
