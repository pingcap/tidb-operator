#!/bin/bash

if ! kubectl -n openebs get lvmsnapshot >/dev/null; then
  echo "lvmsnapshot is unsupported in this cluster, skipping"
  exit 0
fi

BACKUPMETA=backupmeta.json

if [ ! -f $BACKUPMETA ]; then
  echo "ERROR: $BACKUPMETA not found"
  exit 1
fi

for volume_id in $(cat backupmeta.json | jq -r '.tikv.stores[].volumes[].volume_id'); do
  echo "creating snapshot for volume $volume_id"
  pvc=$(kubectl -n openebs get pv "$volume_id" -o json | jq -r '.spec.claimRef.name')

  cat >snap.yaml <<EOF
apiVersion: snapshot.storage.k8s.io/v1
kind: VolumeSnapshot
metadata:
  name: ${pvc}-snap
  namespace: ebs
spec:
  volumeSnapshotClassName: lvmpv-snapclass
  source:
    persistentVolumeClaimName: ${pvc}
EOF
  kubectl apply -f snap.yaml
  rm -f snap.yaml
done

echo -n >snapshot-ids.txt

for volume_id in $(cat backupmeta.json | jq -r '.tikv.stores[].volumes[].volume_id'); do
  echo "wait for snapshot to be ready for volume $volume_id"
  pvc=$(kubectl -n openebs get pv "$volume_id" -o json | jq -r '.spec.claimRef.name')
  snap=${pvc}-snap
  while true; do
    snap_content=$(kubectl -n ebs get volumesnapshot "$snap" -o json | jq -r .status.boundVolumeSnapshotContentName)
    if [ -z "$snap_content" ]; then
      echo "snapshot not ready for volume $volume_id"
      sleep 5
      continue
    fi
    snap_handle=$(kubectl get volumesnapshotcontent "$snap_content" -o json | jq -r '.status.snapshotHandle' | awk -F@ '{print $2}')
    if [ -z "$snap_handle" ]; then
      echo "snap_handle not ready for volume $volume_id"
      sleep 5
      continue
    fi
    break
  done
  echo "${volume_id} ${snap_handle}" >>snapshot-ids.txt
  kubectl patch volumesnapshotcontent "$snap_content" --patch '{"spec":{"deletionPolicy":"Retain"}}' --type=merge
  kubectl -n ebs delete volumesnapshot "$snap"
  kubectl delete volumesnapshotcontent "$snap_content"
done