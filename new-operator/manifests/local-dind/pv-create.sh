#!/usr/bin/env bash
set -euo pipefail

if [[ -z ${1+undefined-guard} ]] ; then
    echo "expected a kubernetes node name as the first argument"
    exit 1
fi

node="$1"
shift

for i in "$@" ; do
    src="/data/local-pv${i}"
    mount_point="/mnt/disks/vol${i}"

    container_id=$(docker ps | grep "$node" | awk '{print $1}' | tail -1)
    # local-volume-provisioner requires the directory be a mount point, so we bind mount other directories under discovery directory
    docker exec "${container_id}" /bin/bash -c "mkdir -p ${src} ${mount_point} && mount --bind ${src} ${mount_point}"
done
