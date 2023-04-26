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

for info in $(cat backupmeta.json | jq -r '.tikv.stores[].volumes[] | "\(.volume_id)@\(.snapshot_id)"'); do
  volume_id=$(echo "$info" | awk -F@ '{print $1}')
  snapshot_id=$(echo "$info" | awk -F@ '{print $2}')
  echo "installing snapshot $snapshot_id for volume $volume_id"
  node_name=$(kubectl -n openebs get lvmsnapshot "$snapshot_id" -o json | jq -r '.spec.ownerNodeID')
  vg_name=$(kubectl -n openebs get lvmsnapshot "$snapshot_id" -o json | jq -r '.spec.volGroup')
  lv_name=${snapshot_id//snapshot-/}
  echo "node_name: $node_name, vg_name: $vg_name, lv_name: $lv_name"
  pod_name=install-${snapshot_id}
  cat >install-snap.yaml <<EOF
apiVersion: v1
kind: Pod
metadata:
  name: ${pod_name}
  namespace: ebs
spec:
  containers:
  - command:
    - "/bin/bash"
    args:
    - "-c"
    - |
      #!/bin/bash
      set -e
      lvconvert --mergesnapshot ${vg_name}/${lv_name}
    image: tispace/lvm-driver:latest
    imagePullPolicy: IfNotPresent
    name: ${pod_name}
    securityContext:
      allowPrivilegeEscalation: true
      privileged: true
    volumeMounts:
    - mountPath: /dev
      name: device-dir
  nodeName: ${node_name}
  restartPolicy: Never
  volumes:
  - hostPath:
      path: /dev
      type: Directory
    name: device-dir
EOF
  kubectl apply -f install-snap.yaml
  rm -f install-snap.yaml
done

echo -n >volume-ids.txt

for info in $(cat backupmeta.json | jq -r '.tikv.stores[].volumes[] | "\(.volume_id)@\(.snapshot_id)"'); do
  volume_id=$(echo "$info" | awk -F@ '{print $1}')
  snapshot_id=$(echo "$info" | awk -F@ '{print $2}')
  echo "waiting for snapshot is installed for volume $volume_id"
  pod_name=install-${snapshot_id}
  while true; do
    status="$(kubectl -n ebs get pod "$pod_name" -o json | jq -r '.status.phase')"
    if [ "$status" == "Succeeded" ]; then
      break
    fi
    if [ "$status" == "Failed" ]; then
      echo "ERROR: failed to install snapshot $snapshot_id for volume $volume_id"
      exit 1
    fi
    echo "status: $status"
    sleep 1
  done
  echo "snapshot is installed for volume $volume_id"
  kubectl -n ebs delete pod "$pod_name"
  kubectl -n openebs delete lvmsnapshot "$snapshot_id"
  echo "$volume_id $volume_id" >>volume-ids.txt
done