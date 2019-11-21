#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

ROOT=$(unset CDPATH && cd $(dirname "${BASH_SOURCE[0]}")/.. && pwd)
cd $ROOT

source $ROOT/hack/lib.sh

hack::ensure_kubectl

TIDB_OPERATOR_IMAGE=${TIDB_OPERATOR_IMAGE:-localhost:5000/pingcap/tidb-operator:latest}
E2E_IMAGE=${E2E_IMAGE:-localhost:5000/pingcap/tidb-operator-e2e:latest}
TEST_APISERVER_IMAGE=${TEST_APISERVER_IMAGE:-localhost:5000/pingcap/test-apiserver:latest}
KUBECONFIG=${KUBECONFIG:-}

if [ -z "$KUBECONFIG" ]; then
    echo "error: KUBECONFIG is required"
    exit 1
fi

echo "TIDB_OPERATOR_IMAGE: $TIDB_OPERATOR_IMAGE"
echo "E2E_IMAGE: $E2E_IMAGE"
echo "TEST_APISERVER_IMAGE: $TEST_APISERVER_IMAGE"
echo "KUBECONFIG: $KUBECONFIG"

GINKGO_PARALLEL=${GINKGO_PARALLEL:-n} # set to 'y' to run tests in parallel
# If 'y', Ginkgo's reporter will not print out in color when tests are run
# in parallel
GINKGO_NO_COLOR=${GINKGO_NO_COLOR:-n}

ginkgo_args=()

if [[ -n "${GINKGO_NODES:-}" ]]; then
    ginkgo_args+=("--nodes=${GINKGO_NODES}")
elif [[ ${GINKGO_PARALLEL} =~ ^[yY]$ ]]; then
    ginkgo_args+=("-p")
fi

if [[ "${GINKGO_NO_COLOR}" == "y" ]]; then
    ginkgo_args+=("--noColor")
fi

NS=tidb-operator-e2e

# TODO move these clean logic into e2e code
echo "info: clear helm releases"
$KUBECTL_BIN -n kube-system delete cm -l OWNER=TILLER

# clear all validatingwebhookconfigurations first otherwise it may deny pods deletion requests
echo "info: clear validatingwebhookconfiguration"
$KUBECTL_BIN delete validatingwebhookconfiguration --all

echo "info: clear namespace ${NS}"
$KUBECTL_BIN delete ns ${NS} --ignore-not-found
$KUBECTL_BIN wait --for=delete -n ${NS} pod/tidb-operator-e2e || true
$KUBECTL_BIN delete clusterrolebinding tidb-operator-e2e --ignore-not-found

echo "info: creating namespace $NS"
$KUBECTL_BIN create ns ${NS}

echo "info: creating service account $NS/tidb-operator-e2e"
# Create sa first and wait for the controller to create the secret for this sa
# This avoids `No API token found for service account " tidb-operator-e2e"`
# TODO better way to work around this issue
$KUBECTL_BIN -n ${NS} create sa tidb-operator-e2e
while true; do
    secret=$($KUBECTL_BIN -n ${NS} get sa tidb-operator-e2e -ojsonpath='{.secrets[0].name}' || true)
    if [[ -n "$secret"  ]]; then
        echo "info: secret '$secret' created for service account ${NS}/tidb-operator-e2e"
        break
    fi
    sleep 1
done

echo "info: creating webhook service and clusterrolebinding etc"
$KUBECTL_BIN -n ${NS} apply -f tests/manifests/e2e/e2e.yaml

echo "info: start to run e2e pod"
e2e_args=(
    /usr/local/bin/ginkgo
    ${ginkgo_args[@]:-}
    /usr/local/bin/e2e.test
    --
    --provider=skeleton 
    --delete-namespace-on-failure=false
    # tidb-operator e2e flags
    # TODO make these configurable via environments
    --operator-tag=e2e
    --operator-image=${TIDB_OPERATOR_IMAGE}
    --test-apiserver-image=${TEST_APISERVER_IMAGE}
    --tidb-versions=v3.0.2,v3.0.3,v3.0.4,v3.0.5
    --chart-dir=/charts
    -v=4
    ${@:-}
)
# We don't attach into the container because the connection may lost.
# Instead we print logs and check the result repeatedly after the pod is Ready.
$KUBECTL_BIN -n ${NS} run tidb-operator-e2e --generator=run-pod/v1 --image $E2E_IMAGE \
    --env NAMESPACE=$NS \
    --labels app=webhook \
    --serviceaccount=tidb-operator-e2e \
    --image-pull-policy=Always \
    --restart=Never \
    --command -- ${e2e_args[@]}
echo "info: wait for e2e pod to be ready"
ret=0
$KUBECTL_BIN -n ${NS} wait --for=condition=Ready pod/tidb-operator-e2e || ret=$?
if [ $ret -ne 0 ]; then
    echo "error: failed to wait for the e2e pod to be ready, printing its logs"
    $KUBECTL_BIN -n ${NS} logs tidb-operator-e2e
    exit 1
fi

echo "info: start to print e2e logs and check the result"
execType=0
while true; do
    if [[ $execType -eq 0 ]]; then
        $KUBECTL_BIN -n ${NS} logs -f tidb-operator-e2e || true
    else
        $KUBECTL_BIN -n ${NS} logs --tail 1 -f tidb-operator-e2e || true
    fi
    phase=$($KUBECTL_BIN -n ${NS} get pods tidb-operator-e2e -ojsonpath='{.status.phase}')
    if [[ "$phase" == "Succeeded" ]]; then
        echo "info: e2e succeeded"
        exit 0
    elif [[ "$phase" == "Failed" ]]; then
        echo "error: e2e failed, phase: $phase"
        exit 1
    fi
    # if we failed on "$KUBECTL_BIN logs -f", try to print one line each time
    # TODO find a better way to print logs starting from the last place
    execType=1
done
