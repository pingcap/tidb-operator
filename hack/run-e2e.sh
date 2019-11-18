#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

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

kubectl delete -f tests/manifests/e2e/e2e.yaml --ignore-not-found
kubectl wait --for=delete -n ${NS} pod/tidb-operator-e2e || true
# Create sa first and wait for the controller to create the secret for this sa
# This avoids `No API token found for service account " tidb-operator-e2e"`
# TODO better way to work around this issue
kubectl -n ${NS} create sa tidb-operator-e2e
# copy and modify to avoid local changes
sed "s#image: localhost:5000/pingcap/tidb-operator-e2e:latest#image: $E2E_IMAGE#g
s#=pingcap/tidb-operator:latest#=${TIDB_OPERATOR_IMAGE}#g
s#=pingcap/test-apiserver:latest#=${TEST_APISERVER_IMAGE}#g
" tests/manifests/e2e/e2e.yaml | kubectl -n ${NS} apply -f -
kubectl -n ${NS} wait --for=condition=Ready pod/tidb-operator-e2e
kubectl -n ${NS} logs -f tidb-operator-e2e
echo "info: checking e2e result"
phase=$(kubectl -n ${NS} get pods tidb-operator-e2e -ojsonpath='{.status.phase}')
if [[ "$phrase" == "Succeeded" ]]; then
    echo "info: e2e succeeded"
else
    echo "error: e2e failed, phrase: $phrase"
    exit 1
fi
