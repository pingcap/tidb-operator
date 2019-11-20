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

NS=tidb-operator-e2e

# clear all validatingwebhookconfigurations first otherwise it may deny pods deletion requests
$KUBECTL_BIN delete validatingwebhookconfiguration --all
$KUBECTL_BIN delete -f tests/manifests/e2e/e2e.yaml --ignore-not-found
$KUBECTL_BIN wait --for=delete -n ${NS} pod/tidb-operator-e2e || true
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
# copy and modify to avoid local changes
sed "s#image: .*#image: $E2E_IMAGE#g
s#--operator-image=.*#--operator-image=${TIDB_OPERATOR_IMAGE}#g
s#--test-apiserver-image=.*#--test-apiserver-image=${TEST_APISERVER_IMAGE}#g
" tests/manifests/e2e/e2e.yaml | $KUBECTL_BIN -n ${NS} apply -f -
$KUBECTL_BIN -n ${NS} wait --for=condition=Ready pod/tidb-operator-e2e
# print e2e logs and check the result
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
