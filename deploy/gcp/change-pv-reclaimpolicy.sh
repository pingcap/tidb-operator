#!/usr/bin/env bash
# Set the ReclaimPolicy of persistent volumes bound to PVCs for a TiDB cluster in a given namespace
# Inputs: Path to a valid kubeconfig file and the namespace in which the PVCs live.
# Run before terraform destroy

set -euo pipefail
set -x

KUBECONFIGFILE=$1
NAMESPACE=$2

if [[ ! -f ${KUBECONFIGFILE} ]]; then
    echo "The given kubeconfig file does not exist"
    exit 1
fi

if ! kubectl --kubeconfig ${KUBECONFIGFILE} get ns ${NAMESPACE}; then
    echo "The given namespace was not found in the kubernetes cluster for the given kubeconfig file"
    exit 1
fi

kubectl --kubeconfig ${KUBECONFIGFILE} get pvc -n ${NAMESPACE} -o jsonpath='{.items[*].spec.volumeName}'|fmt -1 | xargs -I {} kubectl --kubeconfig ${KUBECONFIGFILE} patch pv {} -p '{"spec":{"persistentVolumeReclaimPolicy":"Delete"}}'
