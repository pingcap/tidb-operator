#!/usr/bin/env bash
set -euo pipefail

PV_NUMS=${PV_NUMS:-$(seq 0 3)}
for node in $(kubectl get nodes --no-headers | awk '{print $1}' | grep -v master) ; do
  echo "$(dirname "$0")/pv-create.sh" "$node" ${PV_NUMS}
  "$(dirname "$0")/pv-create.sh" "$node" ${PV_NUMS}
done
