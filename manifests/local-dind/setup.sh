#!/usr/bin/env bash
set -euo pipefail

./manifests/local-dind/pv-hosts.sh
kubectl apply -f manifests/local-dind/local-volume-provisioner.yaml
