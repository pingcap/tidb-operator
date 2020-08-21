#!/usr/bin/env bash
# Set the ReclaimPolicy of persistent volumes bound to PVCs for a TiDB cluster in a given namespace
# Inputs: Path to a valid kubeconfig file and the namespace in which the PVCs live.
# Run before terraform destroy

set -euo pipefail

usage="usage: $0 <kubeconfigfile> <namespace>"

if (( $# != 2 )); then
    printf "%s\n" "$usage" >&2
    exit 1
fi

set -x

kubeconfigfile=$1
namespace=$2

if ! [[ -f $kubeconfigfile ]]; then
    printf "The given kubeconfig file (%s) does not exist\n" "$kubeconfigfile" >&2
    exit 1
fi

if ! kubectl --kubeconfig "$kubeconfigfile" get ns "$namespace"; then
    printf "The given namespace (%s) was not found in the kubernetes cluster for the given kubeconfig file (%s)" "$namespace" "$kubeconfigfile" >&2
    exit 1
fi

kubectl --kubeconfig "$kubeconfigfile" get pvc -n "$namespace" -o jsonpath='{.items[*].spec.volumeName}'|fmt -1 | xargs -I {} kubectl --kubeconfig "$kubeconfigfile" patch pv {} -p '{"spec":{"persistentVolumeReclaimPolicy":"Delete"}}'
