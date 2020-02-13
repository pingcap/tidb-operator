#!/bin/bash

# Copyright 2020 PingCAP, Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# See the License for the specific language governing permissions and
# limitations under the License.

#
# This is a helper script to start a VM and run command in it.
#
# TODO create an isolated network

set -o errexit
set -o nounset
set -o pipefail

ROOT=$(unset CDPATH && cd $(dirname "${BASH_SOURCE[0]}")/.. && pwd)
cd $ROOT

source "${ROOT}/hack/lib.sh"

GCP_CREDENTIALS=${GCP_CREDENTIALS:-}
GCP_PROJECT=${GCP_PROJECT:-}
GCP_ZONE=${GCP_ZONE:-}
GCP_SSH_PRIVATE_KEY=${GCP_SSH_PRIVATE_KEY:-}
GCP_SSH_PUBLIC_KEY=${GCP_SSH_PUBLIC_KEY:-}
NAME=${NAME:-tidb-operator-e2e}
SSH_USER=${SSH_USER:-vagrant}
GIT_URL=${GIT_URL:-https://github.com/pingcap/tidb-operator}
GIT_REF=${GIT_REF:-origin/master}
SYNC_FILES=${SYNC_FILES:-}

echo "GCP_CREDENTIALS: $GCP_CREDENTIALS"
echo "GCP_PROJECT: $GCP_PROJECT"
echo "GCP_ZONE: $GCP_ZONE"
echo "GCP_SSH_PRIVATE_KEY: $GCP_SSH_PRIVATE_KEY"
echo "GCP_SSH_PUBLIC_KEY: $GCP_SSH_PUBLIC_KEY"
echo "NAME: $NAME"
echo "GIT_URL: $GIT_URL"
echo "GIT_REF: $GIT_REF"
echo "SYNC_FILES: $SYNC_FILES"

# Pre-created nested virtualization enabled image with following commands:
#
#   gcloud compute disks create disk1 --image-project centos-cloud --image-family centos-8 --zone us-central1-b
#   gcloud compute images create centos-8-nested-vm \
#     --source-disk disk1 --source-disk-zone us-central1-b \
#     --licenses "https://compute.googleapis.com/compute/v1/projects/vm-options/global/licenses/enable-vmx"
#   gcloud compute disks delete disk1
#
# Refer to
# https://cloud.google.com/compute/docs/instances/enable-nested-virtualization-vm-instances
# for more details.
IMAGE=centos-8-nested-vm

echo "info: configure gcloud"
if [ -z "$GCP_PROJECT" ]; then
    echo "error: GCP_PROJECT is required"
    exit 1
fi
if [ -z "$GCP_CREDENTIALS" ]; then
    echo "error: GCP_CREDENTIALS is required"
    exit 1
fi
if [ -z "$GCP_ZONE" ]; then
    echo "error: GCP_ZONE is required"
    exit 1
fi
gcloud auth activate-service-account --key-file "$GCP_CREDENTIALS"
gcloud config set core/project $GCP_PROJECT
gcloud config set compute/zone $GCP_ZONE

echo "info: preparing ssh keypairs for GCP"
if [ ! -d ~/.ssh ]; then
    mkdir ~/.ssh
fi
if [ ! -e ~/.ssh/google_compute_engine -a -n "$GCP_SSH_PRIVATE_KEY" ]; then
    echo "Copying $GCP_SSH_PRIVATE_KEY to ~/.ssh/google_compute_engine" >&2
    cp $GCP_SSH_PRIVATE_KEY ~/.ssh/google_compute_engine
    chmod 0600 ~/.ssh/google_compute_engine
fi
if [ ! -e ~/.ssh/google_compute_engine.pub -a -n "$GCP_SSH_PUBLIC_KEY" ]; then
    echo "Copying $GCP_SSH_PUBLIC_KEY to ~/.ssh/google_compute_engine.pub" >&2
    cp $GCP_SSH_PUBLIC_KEY ~/.ssh/google_compute_engine.pub
    chmod 0600 ~/.ssh/google_compute_engine.pub
fi

function gcloud_resource_exists() {
    local args=($(tr -s '_' ' ' <<<"$1"))
    unset args[$[${#args[@]}-1]]
    local name="$2"
    x=$(${args[@]} list --filter="name='$name'" --format='table[no-heading](name)' | wc -l)
    [ "$x" -ge 1 ]
}

function gcloud_compute_instances_exists() {
    gcloud_resource_exists ${FUNCNAME[0]} $@
}

function e2e::down() {
    echo "info: tearing down"
    if ! gcloud_compute_instances_exists $NAME; then
        echo "info: instance '$NAME' does not exist, skipped"
        return 0
    fi
    echo "info: deleting instance '$NAME'"
    gcloud compute instances delete $NAME -q
}

function e2e::up() {
    echo "info: setting up"
    echo "info: creating instance '$NAME'"
    gcloud compute instances create $NAME \
        --machine-type n1-standard-8 \
        --min-cpu-platform "Intel Haswell" \
        --image $IMAGE \
        --boot-disk-size 30GB \
        --local-ssd interface=scsi
}

function e2e::test() {
    echo "info: testing"
    echo "info: waiting for the VM is ready"
    hack::wait_for_success 60 3 "gcloud compute ssh $SSH_USER@$NAME --command 'uname -a'"
    echo "info: syncing files $SYNC_FILES"
    while IFS=$',' read -r line; do
        IFS=':' read -r src dst <<< "$line"
        if [ -z "$dst" ]; then
            dst="$src"
        fi
        gcloud compute scp $src $SSH_USER@$NAME:$dst
    done <<< "$SYNC_FILES"
    local tmpfile=$(mktemp)
    trap "rm -f $tmpfile" RETURN
    cat <<EOF > $tmpfile
sudo yum install -y git
cd \$HOME
sudo rm -rf tidb-operator
git init tidb-operator
cd tidb-operator
git fetch --depth 1 --tags --progress ${GIT_URL} +refs/heads/*:refs/remotes/origin/* +refs/pull/*:refs/remotes/origin/pr/* +refs/heads/*:refs/*
GIT_COMMIT=\$(git rev-parse ${GIT_REF}^{commit})
git checkout -f \${GIT_COMMIT}
$@
EOF
    cat $tmpfile
    gcloud compute scp $tmpfile $SSH_USER@$NAME:/tmp/e2e.sh
    gcloud compute ssh $SSH_USER@$NAME --command "bash /tmp/e2e.sh"
}

e2e::down
trap 'e2e::down' EXIT
e2e::up
e2e::test "$@"
