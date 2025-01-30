#!/usr/bin/env bash
# Copyright 2024 PingCAP, Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.


set -o errexit
set -o nounset
set -o pipefail

ROOT=$(cd $(dirname "${BASH_SOURCE[0]}")/../..; pwd -P)

# source only once
[[ $(type -t kind::loaded) == function ]] && return 0

source $ROOT/hack/lib/vars.sh

OUTPUT_DIR=$ROOT/_output
KIND=$OUTPUT_DIR/bin/kind
KUBECTL=$OUTPUT_DIR/bin/kubectl
KIND_CFG_DIR=$OUTPUT_DIR/kind

METAL_LB_NS=metallb-system
METAL_LB_DEPLOYMENT=controller
METAL_LB_MANIFEST=https://raw.githubusercontent.com/metallb/metallb/v0.14.7/config/manifests/metallb-native.yaml
KIND_NETWORK_NAME=kind

declare -A KIND_IMAGE=(
    ["v1.31.0"]="kindest/node:v1.31.0@sha256:53df588e04085fd41ae12de0c3fe4c72f7013bba32a20e7325357a1ac94ba865"
)

function kind::ensure_cluster() {
    if ! command -v $KIND &>/dev/null; then
        echo "kind not found, please run 'make bin/kind'"
    fi

    mkdir -p $KIND_CFG_DIR
    cat << EOF > $KIND_CFG_DIR/config.yaml
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
featureGates:
  VolumeAttributesClass: true
runtimeConfig:
  "storage.k8s.io/v1beta1": "true"
networking:
  ipFamily: dual
nodes:
- role: control-plane
- role: worker
  labels:
    zone: zone-a
- role: worker
  labels:
    zone: zone-b
- role: worker
  labels:
    zone: zone-c
EOF


    if ! $KIND get clusters | grep -q ${V_KIND_CLUSTER}; then
        local image=${KIND_IMAGE[${V_KUBE_VERSION}]}
        echo "kind cluster ${V_KIND_CLUSTER} not found, creating"

        local opt="--image $image"
        if [[ -z $image ]]; then
            echo "$V_KUBE_VERSION is not supported, use default one supported by kind"
            opt=""
        else
            echo "create cluster with image $image"
        fi
        $KIND create cluster --name ${V_KIND_CLUSTER} --config $KIND_CFG_DIR/config.yaml ${opt}
    fi
}

# marker function
function kind::loaded() {
  return 0
}

function kind::ensure_metal_lb() {
    echo "installing MetalLB..."
    $KUBECTL apply -f $METAL_LB_MANIFEST

    echo "waiting for MetalLB to be ready..."
    $KUBECTL -n $METAL_LB_NS wait --for=condition=Available --timeout=5m deployment/$METAL_LB_DEPLOYMENT

    echo "getting IPv4 subnet from Kind..."
    subnet=$(docker network inspect -f '{{index .IPAM.Config 0 "Subnet"}}' $KIND_NETWORK_NAME | cut -d. -f1-2)
    if [[ $subnet == *"/"* ]]; then
        subnet=$(docker network inspect -f '{{index .IPAM.Config 1 "Subnet"}}' $KIND_NETWORK_NAME | cut -d. -f1-2)
    fi

    echo "configuring address pool for MetalLB..."
    cat <<EOF | $KUBECTL apply -f -
apiVersion: metallb.io/v1beta1
kind: IPAddressPool
metadata:
    name: addr-pool
    namespace: metallb-system
spec:
    addresses:
    - $subnet.255.100-$subnet.255.250
---
apiVersion: metallb.io/v1beta1
kind: L2Advertisement
metadata:
    name: advertisement
    namespace: metallb-system
EOF
}
