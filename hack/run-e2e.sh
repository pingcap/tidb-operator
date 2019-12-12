#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

ROOT=$(unset CDPATH && cd $(dirname "${BASH_SOURCE[0]}")/.. && pwd)
cd $ROOT

source $ROOT/hack/lib.sh

hack::ensure_kubectl
hack::ensure_helm

TIDB_OPERATOR_IMAGE=${TIDB_OPERATOR_IMAGE:-localhost:5000/pingcap/tidb-operator:latest}
E2E_IMAGE=${E2E_IMAGE:-localhost:5000/pingcap/tidb-operator-e2e:latest}
TEST_APISERVER_IMAGE=${TEST_APISERVER_IMAGE:-localhost:5000/pingcap/test-apiserver:latest}
KUBECONFIG=${KUBECONFIG:-$HOME/.kube/config}
KUBECONTEXT=${KUBECONTEXT:-}

if [ -z "$KUBECONFIG" ]; then
    echo "error: KUBECONFIG is required"
    exit 1
fi

echo "TIDB_OPERATOR_IMAGE: $TIDB_OPERATOR_IMAGE"
echo "E2E_IMAGE: $E2E_IMAGE"
echo "TEST_APISERVER_IMAGE: $TEST_APISERVER_IMAGE"
echo "KUBECONFIG: $KUBECONFIG"
echo "KUBECONTEXT: $KUBECONTEXT"

GINKGO_PARALLEL=${GINKGO_PARALLEL:-n} # set to 'y' to run tests in parallel
# If 'y', Ginkgo's reporter will not print out in color when tests are run
# in parallel
GINKGO_NO_COLOR=${GINKGO_NO_COLOR:-n}
GINKGO_STREAM=${GINKGO_STREAM:-y}

ginkgo_args=()

if [[ -n "${GINKGO_NODES:-}" ]]; then
    ginkgo_args+=("--nodes=${GINKGO_NODES}")
elif [[ ${GINKGO_PARALLEL} =~ ^[yY]$ ]]; then
    ginkgo_args+=("-p")
fi

if [[ "${GINKGO_NO_COLOR}" == "y" ]]; then
    ginkgo_args+=("--noColor")
fi

if [[ "${GINKGO_STREAM}" == "y" ]]; then
    ginkgo_args+=("--stream")
fi

kubectl_args=()
if [[ -n "$KUBECONTEXT" ]]; then
    kubectl_args+=(--context "$KUBECONTEXT")
fi

# TODO move these clean logic into e2e code
echo "info: clear helm releases"
$HELM_BIN ls --all --short | xargs -n 1 -r $HELM_BIN delete --purge

echo "info: clear non-kubernetes apiservices"
$KUBECTL_BIN ${kubectl_args[@]:-} delete apiservices -l kube-aggregator.kubernetes.io/automanaged!=onstart

# clear all validatingwebhookconfigurations first otherwise it may deny pods deletion requests
echo "info: clear validatingwebhookconfiguration"
$KUBECTL_BIN ${kubectl_args[@]:-} delete validatingwebhookconfiguration --all

echo "info: start to run e2e process"
e2e_args=(
    /usr/local/bin/ginkgo
    ${ginkgo_args[@]:-}
    /usr/local/bin/e2e.test
    --
    --provider=skeleton 
    --clean-start=true
    --delete-namespace-on-failure=false
    # tidb-operator e2e flags
    --operator-tag=e2e
    --operator-image=${TIDB_OPERATOR_IMAGE}
    --e2e-image=${E2E_IMAGE}
    --test-apiserver-image=${TEST_APISERVER_IMAGE}
    --tidb-versions=v3.0.2,v3.0.3,v3.0.4,v3.0.5
    --chart-dir=/charts
    -v=4
    ${@:-}
)

docker run --rm \
    --net=host \
    -v $KUBECONFIG:/etc/kubernetes/admin.conf \
    --env KUBECONFIG=/etc/kubernetes/admin.conf \
    --env KUBECONTEXT=$KUBECONTEXT \
    $E2E_IMAGE \
    ${e2e_args[@]}
