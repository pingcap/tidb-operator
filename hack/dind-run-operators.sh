#!/usr/bin/env bash
set -euo pipefail

helm init --upgrade
./manifests/local-dind/setup.sh
kubectl apply -f ./manifests/crd.yaml
helm install charts/tidb-operator --name tidb-operator --namespace=tidb-admin --set "imagePullPolicy=Always"
helm install charts/tidb-cluster --name tidb-cluster --namespace=tidb
