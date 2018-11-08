#!/usr/bin/env bash
set -euo pipefail

PV_NUMS=${PV_NUMS:-$(seq 0 3)}
for node in $(kubectl get nodes --no-headers | grep -v master | awk '{print $1}') ; do
  echo "$(dirname "$0")/pv-create.sh" "$node" ${PV_NUMS}
  "$(dirname "$0")/pv-create.sh" "$node" ${PV_NUMS}
done
