#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

ROOT=$(unset CDPATH && cd $(dirname "${BASH_SOURCE[0]}")/.. && pwd)
cd $ROOT

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
kubectl delete validatingwebhookconfiguration --all
kubectl delete -f tests/manifests/e2e/e2e.yaml --ignore-not-found
kubectl wait --for=delete -n ${NS} pod/tidb-operator-e2e || true
# Create sa first and wait for the controller to create the secret for this sa
# This avoids `No API token found for service account " tidb-operator-e2e"`
# TODO better way to work around this issue
kubectl -n ${NS} create sa tidb-operator-e2e
while true; do
    secret=$(kubectl -n ${NS} get sa tidb-operator-e2e -ojsonpath='{.secrets[0].name}' || true)
    if [[ -n "$secret"  ]]; then
        echo "info: secret '$secret' created for service account ${NS}/tidb-operator-e2e"
        break
    fi
    sleep 1
done
# copy and modify to avoid local changes
sed "s#image: localhost:5000/pingcap/tidb-operator-e2e:latest#image: $E2E_IMAGE#g
s#--operator-image=.*#--operator-image=${TIDB_OPERATOR_IMAGE}#g
s#--test-apiserver-image=.*#--test-apiserver-image=${TEST_APISERVER_IMAGE}#g
" tests/manifests/e2e/e2e.yaml | kubectl -n ${NS} apply -f -
kubectl -n ${NS} wait --for=condition=Ready pod/tidb-operator-e2e
# print e2e logs
execType=0
while true; do
    if [[ $execType -eq 0 ]]; then
        kubectl -n ${NS} logs -f tidb-operator-e2e || true
    else
        kubectl -n ${NS} logs --tail 1 -f tidb-operator-e2e || true
    fi
    phase=$(kubectl -n ${NS} get pods tidb-operator-e2e -ojsonpath='{.status.phase}')
    if [[ "$phase" == "Succeeded" ]]; then
        echo "info: e2e succeeded"
        exit 0
    elif [[ "$phase" == "Failed" ]]; then
        echo "error: e2e failed, phase: $phase"
        exit 1
    fi
    # if we failed on "kubectl logs -f", try to print one line each time
    execType=1
done
