#!/usr/bin/env bash
set -euo pipefail

./manifests/local-dind/setup.sh
kubectl apply -f ./manifests/crd.yaml
helm install charts/tidb-operator --name tidb-operator --namespace=tidb-operator
helm install charts/tidb-cluster --name tidb-cluster --namespace=tidb
