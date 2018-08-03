#!/usr/bin/env bash
set -euo pipefail

if [[ -z ${1+undefined-guard} ]] ; then
  echo "expected a kubernetes node name as the first argument"
  exit 1
fi

node="$1"
shift

for i in "$@" ; do
  # Make the directory to mount into
  host_dir="/mnt/disk$i"
  container_id=$(docker ps | grep "$node" | awk '{print $1}' | tail -1)
  docker exec "$container_id" mkdir -p "$host_dir"

  kubectl apply -f - <<TEMPLATE
apiVersion: v1
kind: PersistentVolume
metadata:
 name: ${node}-mnt-local-${i}
spec:
 capacity:
   storage: 100Gi
 # volumeMode field requires BlockVolume Alpha feature gate to be enabled.
 # volumeMode: Filesystem
 accessModes:
 - ReadWriteOnce
 persistentVolumeReclaimPolicy: Retain
 storageClassName: local-storage
 local:
   path: $host_dir
 # requires k8s >= v1.10
 nodeAffinity:
    required:
      nodeSelectorTerms:
      - matchExpressions:
        - key: kubernetes.io/hostname
          operator: In
          values:
          - ${node}
TEMPLATE
done
