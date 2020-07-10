#!/usr/bin/env bash

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
# E2E entrypoint script for OpenShift.
#

set -o errexit
set -o nounset
set -o pipefail

ROOT=$(unset CDPATH && cd $(dirname "${BASH_SOURCE[0]}")/.. && pwd)
cd $ROOT

PULL_SECRET_FILE=${PULL_SECRET_FILE:-}
DOCKER_VERSION=${DOCKER_VERSION:-19.03.9}

if [ ! -e "$PULL_SECRET_FILE" ]; then
    echo "error: pull secret file '$PULL_SECRET_FILE' does not exist"
    exit 1
fi

vmx_cnt=$(grep -cw vmx /proc/cpuinfo)
if [ "$vmx_cnt" -gt 0 ]; then
    echo "info: nested virtualization enabled (vmx cnt: $vmx_cnt)"
else
    echo "error: nested virtualization not enabled, please refer to https://cloud.google.com/compute/docs/instances/enable-nested-virtualization-vm-instances"
    exit 1
fi

echo "info: install required software packages"
sudo yum install -y NetworkManager
sudo yum install -y jq git make golang
sudo yum install -y yum-utils
sudo yum-config-manager \
    --add-repo https://download.docker.com/linux/centos/docker-ce.repo
# pin the same version for daemon and cli: https://github.com/docker/cli/issues/2533
sudo yum install -y docker-ce-${DOCKER_VERSION} docker-ce-cli-${DOCKER_VERSION}
if ! systemctl is-active --quiet docker; then
    sudo systemctl start docker
fi
echo "info: printing docker information"
sudo docker info
sudo chmod o+rw /var/run/docker.sock

CRC_HOME=$HOME/.crc
echo "info: mouting disk onto $CRC_HOME"
if ! mountpoint $CRC_HOME &>/dev/null; then
    sudo mkfs.ext4 -F /dev/disk/by-id/google-local-ssd-0
    sudo rm -rf $CRC_HOME
    mkdir $CRC_HOME
    sudo mount /dev/disk/by-id/google-local-ssd-0 $CRC_HOME
    sudo chown -R $(id -u):$(id -g) $CRC_HOME
fi

echo "info: downloading latest crc"
cd $HOME
CRC_VERSION=$(curl --retry 10 -L -s 'https://mirror.openshift.com/pub/openshift-v4/clients/crc/latest/release-info.json' | jq -r '.version.crcVersion')
if ! test -e crc-linux-amd64.tar.xz; then
    curl --retry 10 -LO https://mirror.openshift.com/pub/openshift-v4/clients/crc/$CRC_VERSION/crc-linux-amd64.tar.xz
    tar -xvf crc-linux-amd64.tar.xz
fi
export PATH=$HOME/crc-linux-$CRC_VERSION-amd64:$PATH

crc version

crcStatus=$(crc status 2>/dev/null | awk '/CRC VM:/ {print $3}') || true
if [[ "$crcStatus" == "Running" ]]; then
    echo "info: OpenShift cluster is running"
    crc status
else
    echo "info: starting OpenShift clsuter"
    crc setup
    crc config set cpus 6
    crc config set memory 24576
    crc start --pull-secret-file $PULL_SECRET_FILE
fi

echo "info: login"
eval $(crc oc-env)
KUBEADMIN_PASSWORD=$(cat $HOME/.crc/cache/crc_libvirt_*/kubeadmin-password)
oc login -u kubeadmin -p "$KUBEADMIN_PASSWORD" https://api.crc.testing:6443 --insecure-skip-tls-verify

echo "info: building images"
cd $HOME/tidb-operator
./hack/run-in-container.sh bash -c 'make docker e2e-docker
images=(
    tidb-operator:latest
    tidb-backup-manager:latest
    tidb-operator-e2e:latest
)
for image in ${images[@]}; do
    docker save localhost:5000/pingcap/$image -o $image.tar.gz
done
'

echo "info: pusing images"
OC_PROJECT=openshift
oc extract secret/router-ca --keys=tls.crt -n openshift-ingress-operator
sudo mkdir /etc/docker/certs.d/default-route-openshift-image-registry.apps-crc.testing/ -p
sudo mv tls.crt /etc/docker/certs.d/default-route-openshift-image-registry.apps-crc.testing/
docker login -u kubeadmin --password-stdin default-route-openshift-image-registry.apps-crc.testing <<< "$(oc whoami -t)"

images=(
    tidb-operator:latest
    tidb-backup-manager:latest
    tidb-operator-e2e:latest
)
for image in ${images[@]}; do
    sudo chown -R $(id -u):$(id -g) $image.tar.gz
    docker load -i $image.tar.gz
    docker tag localhost:5000/pingcap/$image image-registry.openshift-image-registry.svc:5000/$OC_PROJECT/$image
    docker tag localhost:5000/pingcap/$image default-route-openshift-image-registry.apps-crc.testing/$OC_PROJECT/$image
    docker push default-route-openshift-image-registry.apps-crc.testing/$OC_PROJECT/$image
done

export PROVIDER=openshift
export TIDB_OPERATOR_IMAGE=image-registry.openshift-image-registry.svc:5000/$OC_PROJECT/tidb-operator:latest
export TIDB_BACKUP_MANAGER_IMAGE=image-registry.openshift-image-registry.svc:5000/$OC_PROJECT/tidb-backup-manager:latest
export E2E_IMAGE=image-registry.openshift-image-registry.svc:5000/$OC_PROJECT/tidb-operator-e2e:latest
# 'Restarter' test starts 1 replica of pd and tikv and can pass in single-node OpenShift cluster.
./hack/run-e2e.sh --ginkgo.focus 'Restarter'
