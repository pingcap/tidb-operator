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
# E2E entrypoint script.
#

set -o errexit
set -o nounset
set -o pipefail

ROOT=$(unset CDPATH && cd $(dirname "${BASH_SOURCE[0]}")/.. && pwd)
cd $ROOT

source "${ROOT}/hack/lib.sh"

# check bash version
BASH_MAJOR_VERSION=$(echo "$BASH_VERSION" | cut -d '.' -f 1)
# we need bash version >= 4
if [ $BASH_MAJOR_VERSION -lt 4 ]
then
  echo "error: e2e.sh could not work with bash version earlier than 4 for now, please upgrade your bash"
  exit 1
fi


function usage() {
    cat <<'EOF'
This script is entrypoint to run e2e tests.

Usage: hack/e2e.sh [-h] -- [extra test args]

    -h      show this message and exit

Environments:

    PROVIDER              Kubernetes provider, e.g. kind, gke, eks, defaults: kind
    DOCKER_REPO           docker image repo
    IMAGE_TAG             image tag
    CLUSTER               the name of e2e cluster, defaults: tidb-operator
    KUBECONFIG            path to the kubeconfig file, defaults: ~/.kube/config
    SKIP_BUILD            skip building binaries
    SKIP_IMAGE_BUILD      skip build and push images
    SKIP_IMAGE_LOAD       skip load images
    SKIP_UP               skip starting the cluster
    SKIP_DOWN             skip shutting down the cluster
    SKIP_DUMP             skip dump cluster logs
    SKIP_TEST             skip running the test
    KUBE_VERSION          the version of Kubernetes to test against
    KUBE_WORKERS          the number of worker nodes (excludes master nodes), defaults: 3
    DOCKER_IO_MIRROR      configure mirror for docker.io
    GCR_IO_MIRROR         configure mirror for gcr.io
    QUAY_IO_MIRROR        configure mirror for quay.io
    KIND_DATA_HOSTPATH    (kind only) the host path of data directory for kind cluster, defaults: none
    KIND_ETCD_DATADIR     (kind only) the host path of etcd data directory for kind cluster, defaults: none
    GCP_PROJECT           (gke only) the GCP project to run in
    GCP_CREDENTIALS       (gke only) the GCP service account to use
    GCP_REGION            (gke only) the GCP region, if specified a regional cluster is creaetd
    GCP_ZONE              (gke only) the GCP zone, if specified a zonal cluster is created
    GCP_SSH_PRIVATE_KEY   (gke only) the path to the private ssh key
    GCP_SSH_PUBLIC_KEY    (gke only) the path to the public ssh key
    GCP_MACHINE_TYPE      (gke only) the machine type of instance, defaults: n1-standard-4
    AWS_ACCESS_KEY_ID     (eks only) the aws access key id
    AWS_SECRET_ACCESS_KEY (eks only) the aws secret access key
    AWS_REGION            (eks only) the aws region
    AWS_ZONE              (eks only) the aws zone
    GINKGO_NODES          ginkgo nodes to run specs, defaults: 1
    GINKGO_PARALLEL       if set to `y`, will run specs in parallel, the number of nodes will be the number of cpus
    GINKGO_NO_COLOR       if set to `y`, suppress color output in default reporter
    RUNNER_SUITE_NAME     the suite name of runner
    SKIP_GINKGO           if set to `y`, skip ginkgo
    ARTIFACTS             directory to put artifacts

Examples:

0) view help

    ./hack/e2e.sh -h

1) run all specs

    ./hack/e2e.sh
    GINKGO_NODES=8 ./hack/e2e.sh # in parallel

2) limit specs to run

    ./hack/e2e.sh -- --ginkgo.focus='Basic'
    ./hack/e2e.sh -- --ginkgo.focus='Backup\sand\srestore'

    See https://onsi.github.io/ginkgo/ for more ginkgo options.

3) reuse the cluster and don't tear down it after the testing

    # for the first time, skip the down phase
    SKIP_DOWN=y ./hack/e2e.sh -- <e2e args>
    # then skip both the up/down phase in subsequent tests
    SKIP_UP=y SKIP_DOWN=y ./hack/e2e.sh -- <e2e args>

4) use registry mirrors

    DOCKER_IO_MIRROR=https://dockerhub.azk8s.cn QUAY_IO_MIRROR=https://quay.azk8s.cn GCR_IO_MIRROR=https://gcr.azk8s.cn ./hack/e2e.sh -- <e2e args>

5) run e2e with gke provider locally

    You need prepare GCP service account with the following permissions:

        - Compute Network Admin
        - Kubernetes Engine Admin
        - Service Account User
        - Storage Admin
        - Compute Instance Admin (v1)

    You can create ssh keypair with ssh-keygen at  ~/.ssh/google_compute_engine
    or specifc existing ssh keypair with following environments:

        export GCP_SSH_PRIVATE_KEY=<path-to-your-ssh-private-key>
        export GCP_SSH_PUBLIC_KEY=<path-to-your-ssh-public-key>

    Then run with following additional GCP-specific environments:

        export GCP_PROJECT=<project>
        export GCP_CREDENTIALS=<path-to-gcp-service-account>
        export GCP_ZONE=us-central1-b

        PROVIDER=gke ./hack/e2e.sh -- <e2e args>

    If you run the outside of the dev containter started by
    ./hack/run-in-container.sh, Google Cloud SDK must be installed on you
    machine.

6) run e2e with eks provider locally

    You need configure your aws credential and region or set it via following
    environments:

        export AWS_ACCESS_KEY_ID=<your-aws-access-key-id>
        export AWS_SECRET_ACCESS_KEY=<your-aws-secret-key-id>
        export AWS_REGION=<your-aws-region>

    then run e2e with eks provider:

        PROVIDER=eks ./hack/e2e.sh -- <e2e args>

    If you run the outside of the dev containter started by
    ./hack/run-in-container.sh, AWS CLI must be installed on you
    machine.

EOF

}

while getopts "h?" opt; do
    case "$opt" in
    h|\?)
        usage
        exit 0
        ;;
    esac
done

if [ "${1:-}" == "--" ]; then
    shift
fi

PROVIDER=${PROVIDER:-kind}
DOCKER_REPO=${DOCKER_REPO:-localhost:5000/pingcap}
IMAGE_TAG=${IMAGE_TAG:-latest}
CLUSTER=${CLUSTER:-tidb-operator}
KUBECONFIG=${KUBECONFIG:-~/.kube/config}
SKIP_BUILD=${SKIP_BUILD:-}
SKIP_IMAGE_BUILD=${SKIP_IMAGE_BUILD:-}
SKIP_IMAGE_LOAD=${SKIP_IMAGE_LOAD:-}
SKIP_UP=${SKIP_UP:-}
SKIP_DOWN=${SKIP_DOWN:-}
SKIP_TEST=${SKIP_TEST:-}
SKIP_DUMP=${SKIP_DUMP:-}
REUSE_CLUSTER=${REUSE_CLUSTER:-}
KIND_DATA_HOSTPATH=${KIND_DATA_HOSTPATH:-none}
KIND_ETCD_DATADIR=${KIND_ETCD_DATADIR:-none}
GCP_PROJECT=${GCP_PROJECT:-}
GCP_CREDENTIALS=${GCP_CREDENTIALS:-}
GCP_REGION=${GCP_REGION:-}
GCP_ZONE=${GCP_ZONE:-}
GCP_SSH_PRIVATE_KEY=${GCP_SSH_PRIVATE_KEY:-}
GCP_SSH_PUBLIC_KEY=${GCP_SSH_PUBLIC_KEY:-}
GCP_MACHINE_TYPE=${GCP_MACHINE_TYPE:-n1-standard-4}
AWS_ACCESS_KEY_ID=${AWS_ACCESS_KEY_ID:-}
AWS_SECRET_ACCESS_KEY=${AWS_SECRET_ACCESS_KEY:-}
AWS_REGION=${AWS_REGION:-}
AWS_ZONE=${AWS_ZONE:-}
KUBE_VERSION=${KUBE_VERSION:-v1.18.2}
KUBE_WORKERS=${KUBE_WORKERS:-3}
DOCKER_IO_MIRROR=${DOCKER_IO_MIRROR:-}
GCR_IO_MIRROR=${GCR_IO_MIRROR:-}
QUAY_IO_MIRROR=${QUAY_IO_MIRROR:-}
SKIP_GINKGO=${SKIP_GINKGO:-}
RUNNER_SUITE_NAME=${RUNNER_SUITE_NAME:-}
ARTIFACTS=${ARTIFACTS:-}
GIT_BRANCH=${SRC_BRANCH:-}
GIT_COMMIT=${GIT_COMMIT:-}
PR_ID=${PR_ID:-}
CODECOV_TOKEN=${CODECOV_TOKEN:-}

echo "PROVIDER: $PROVIDER"
echo "DOCKER_REPO: $DOCKER_REPO"
echo "IMAGE_TAG: $IMAGE_TAG"
echo "CLUSTER: $CLUSTER"
echo "KUBECONFIG: $KUBECONFIG"
echo "SKIP_BUILD: $SKIP_BUILD"
echo "SKIP_IMAGE_BUILD: $SKIP_IMAGE_BUILD"
echo "SKIP_UP: $SKIP_UP"
echo "SKIP_DOWN: $SKIP_DOWN"
echo "SKIP_TEST: $SKIP_TEST"
echo "SKIP_DUMP: $SKIP_DUMP"
echo "KIND_DATA_HOSTPATH: $KIND_DATA_HOSTPATH"
echo "KIND_ETCD_DATADIR: $KIND_ETCD_DATADIR"
echo "GCP_PROJECT: $GCP_PROJECT"
echo "GCP_CREDENTIALS: $GCP_CREDENTIALS"
echo "GCP_REGION: $GCP_REGION"
echo "GCP_ZONE: $GCP_ZONE"
# We shouldn't print aws credential environments.
# echo "AWS_ACCESS_KEY_ID: $AWS_ACCESS_KEY_ID"
# echo "AWS_SECRET_ACCESS_KEY: $AWS_SECRET_ACCESS_KEY"
echo "AWS_REGION: $AWS_REGION"
echo "AWS_ZONE: $AWS_ZONE"
echo "KUBE_VERSION: $KUBE_VERSION"
echo "KUBE_WORKERS: $KUBE_WORKERS"
echo "DOCKER_IO_MIRROR: $DOCKER_IO_MIRROR"
echo "GCR_IO_MIRROR: $GCR_IO_MIRROR"
echo "QUAY_IO_MIRROR: $QUAY_IO_MIRROR"
echo "ARTIFACTS: $ARTIFACTS"
echo "GIT_BRANCH: $GIT_BRANCH"
echo "GIT_COMMIT: $GIT_COMMIT"
echo "PR_ID: $PR_ID"

# https://github.com/kubernetes-sigs/kind/releases/tag/v0.8.1
declare -A kind_node_images
kind_node_images["v1.11.10"]="kindest/node:v1.11.10@sha256:74c8740710649a3abb169e7f348312deff88fc97d74cfb874c5095ab3866bb42"
kind_node_images["v1.12.10"]="kindest/node:v1.12.10@sha256:faeb82453af2f9373447bb63f50bae02b8020968e0889c7fa308e19b348916cb"
kind_node_images["v1.13.12"]="kindest/node:v1.13.12@sha256:214476f1514e47fe3f6f54d0f9e24cfb1e4cda449529791286c7161b7f9c08e7"
kind_node_images["v1.14.10"]="kindest/node:v1.14.10@sha256:6cd43ff41ae9f02bb46c8f455d5323819aec858b99534a290517ebc181b443c6"
kind_node_images["v1.15.11"]="kindest/node:v1.15.11@sha256:6cc31f3533deb138792db2c7d1ffc36f7456a06f1db5556ad3b6927641016f50"
kind_node_images["v1.16.9"]="kindest/node:v1.16.9@sha256:7175872357bc85847ec4b1aba46ed1d12fa054c83ac7a8a11f5c268957fd5765"
kind_node_images["v1.17.5"]="kindest/node:v1.17.5@sha256:ab3f9e6ec5ad8840eeb1f76c89bb7948c77bbf76bcebe1a8b59790b8ae9a283a"
kind_node_images["v1.18.2"]="kindest/node:v1.18.2@sha256:7b27a6d0f2517ff88ba444025beae41491b016bc6af573ba467b70c5e8e0d85f"

function e2e::image_build() {
    if [ -n "$SKIP_BUILD" ]; then
        echo "info: skip building binaries"
        export NO_BUILD=y
    fi
    if [ -n "$SKIP_IMAGE_BUILD" ]; then
        echo "info: skip building and pushing images"
        return
    fi
    DOCKER_REPO=$DOCKER_REPO IMAGE_TAG=$IMAGE_TAG make docker
    DOCKER_REPO=$DOCKER_REPO IMAGE_TAG=$IMAGE_TAG make e2e-docker
}

function e2e::__restart_docker() {
    echo "info: restarting docker"
    service docker restart
    # the service can be started but the docker socket not ready, wait for ready
    local WAIT_N=0
    local MAX_WAIT=5
    while true; do
        # docker ps -q should only work if the daemon is ready
        docker ps -q > /dev/null 2>&1 && break
        if [[ ${WAIT_N} -lt ${MAX_WAIT} ]]; then
            WAIT_N=$((WAIT_N+1))
            echo "info; Waiting for docker to be ready, sleeping for ${WAIT_N} seconds."
            sleep ${WAIT_N}
        else
            echo "info: Reached maximum attempts, not waiting any longer..."
            break
        fi
    done
    echo "info: done restarting docker"
}

function e2e::__configure_docker_mirror_for_dind() {
    echo "info: configure docker.io mirror '$DOCKER_IO_MIRROR' for DinD"
cat <<EOF > /etc/docker/daemon.json.tmp
{
    "registry-mirrors": ["$DOCKER_IO_MIRROR"]
}
EOF
    if diff /etc/docker/daemon.json.tmp /etc/docker/daemon.json 1>/dev/null 2>&1; then
        echo "info: already configured"
        rm /etc/docker/daemon.json.tmp
    else
        mv /etc/docker/daemon.json.tmp /etc/docker/daemon.json
        e2e::__restart_docker
    fi
}

function e2e::create_kindconfig() {
    local tmpfile=${1}
    cat <<EOF > $tmpfile
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
kubeadmConfigPatches:
- |
  kind: ClusterConfiguration
  apiVersion: kubeadm.k8s.io/v1beta1
  scheduler:
    extraArgs:
      v: "4"
  controllerManager:
    extraArgs:
      v: "4"
- |
  kind: ClusterConfiguration
  apiVersion: kubeadm.k8s.io/v1beta2
  scheduler:
    extraArgs:
      v: "4"
  controllerManager:
    extraArgs:
      v: "4"
- |
  # backward compatibility for Kubernetes 1.12 and prior versions
  kind: ClusterConfiguration
  apiVersion: kubeadm.k8s.io/v1alpha3
  schedulerExtraArgs:
    v: "4"
  controllerManagerExtraArgs:
    v: "4"
EOF
    if [ -n "$DOCKER_IO_MIRROR" -o -n "$GCR_IO_MIRROR" -o -n "$QUAY_IO_MIRROR" ]; then
cat <<EOF >> $tmpfile
containerdConfigPatches:
- |-
EOF
        if [ -n "$DOCKER_IO_MIRROR" ]; then
cat <<EOF >> $tmpfile
  [plugins."io.containerd.grpc.v1.cri".registry.mirrors."docker.io"]
    endpoint = ["$DOCKER_IO_MIRROR"]
EOF
        fi
        if [ -n "$GCR_IO_MIRROR" ]; then
cat <<EOF >> $tmpfile
  [plugins."io.containerd.grpc.v1.cri".registry.mirrors."gcr.io"]
    endpoint = ["$GCR_IO_MIRROR"]
EOF
        fi
        if [ -n "$QUAY_IO_MIRROR" ]; then
cat <<EOF >> $tmpfile
  [plugins."io.containerd.grpc.v1.cri".registry.mirrors."quay.io"]
    endpoint = ["$QUAY_IO_MIRROR"]
EOF
        fi
    fi
    # control-plane
    cat <<EOF >> $tmpfile
nodes:
- role: control-plane
EOF
    if [[ "$KIND_DATA_HOSTPATH" != "none" || "$KIND_ETCD_DATADIR" != "none" ]]; then
        cat <<EOF >> $tmpfile
  extraMounts:
EOF
        if [[ "$KIND_DATA_HOSTPATH" != "none" ]]; then
            if [ ! -d "$KIND_DATA_HOSTPATH" ]; then
                echo "error: '$KIND_DATA_HOSTPATH' is not a directory"
                exit 1
            fi
            local hostWorkerPath="${KIND_DATA_HOSTPATH}/control-plane"
            test -d $hostWorkerPath || mkdir $hostWorkerPath
            cat <<EOF >> $tmpfile
  - containerPath: /mnt/disks/
    hostPath: "$hostWorkerPath"
    propagation: HostToContainer
EOF
        fi
        if [[ "$KIND_ETCD_DATADIR" != "none" ]]; then
            if [ ! -d "$KIND_ETCD_DATADIR" ]; then
                echo "error: '$KIND_ETCD_DATADIR' is not a directory"
                exit 1
            fi
            cat <<EOF >> $tmpfile
  - containerPath: /var/lib/etcd
    hostPath: "$KIND_ETCD_DATADIR"
EOF
        fi
    fi
    # workers
    for ((i = 1; i <= $KUBE_WORKERS; i++)) {
        cat <<EOF >> $tmpfile
- role: worker
EOF
        if [[ "$KIND_DATA_HOSTPATH" != "none" ]]; then
            if [ ! -d "$KIND_DATA_HOSTPATH" ]; then
                echo "error: '$KIND_DATA_HOSTPATH' is not a directory"
                exit 1
            fi
            local hostWorkerPath="${KIND_DATA_HOSTPATH}/worker${i}"
            test -d $hostWorkerPath || mkdir $hostWorkerPath
            cat <<EOF >> $tmpfile
  extraMounts:
  - containerPath: /mnt/disks/
    hostPath: "$hostWorkerPath"
    propagation: HostToContainer
EOF
        fi
    }
}

hack::ensure_kind
hack::ensure_kubectl
hack::ensure_helm

if [ -n "$DOCKER_IO_MIRROR" -a -n "${DOCKER_IN_DOCKER_ENABLED:-}" ]; then
    e2e::__configure_docker_mirror_for_dind
fi

e2e::image_build

kubetest2_args=(
    $PROVIDER
)

if [ -n "$RUNNER_SUITE_NAME" ]; then
    kubetest2_args+=(
        --suite-name "$RUNNER_SUITE_NAME"
    )
fi

if [ -z "$SKIP_UP" ]; then
    kubetest2_args+=(--up)
fi

if [ -z "$SKIP_DOWN" ]; then
    kubetest2_args+=(--down)
fi

if [ -z "$SKIP_TEST" ]; then
    kubetest2_args+=(--test exec)
fi

if [ "$PROVIDER" == "kind" ]; then
    tmpfile=$(mktemp)
    trap "test -f $tmpfile && rm $tmpfile" EXIT
    e2e::create_kindconfig $tmpfile
    echo "info: print the contents of kindconfig"
    cat $tmpfile
    image=""
    for v in ${!kind_node_images[*]}; do
        if [[ "$KUBE_VERSION" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ && "$KUBE_VERSION" == "$v" ]]; then
            image=${kind_node_images[$v]}
            echo "info: image for $KUBE_VERSION: $image"
        elif [[ "$KUBE_VERSION" =~ ^v[0-9]+\.[0-9]+$ && "$KUBE_VERSION" == "${v%.*}" ]]; then
            image=${kind_node_images[$v]}
            echo "info: image for $KUBE_VERSION: $image"
        fi
    done
    if [ -z "$image" ]; then
        echo "error: no image for $KUBE_VERSION, exit"
        exit 1
    fi
    kubetest2_args+=(--image-name $image)
    kubetest2_args+=(
        # add some retires because kind may fail to start the cluster when the
        # load is high
        --up-retries 3
        --cluster-name "$CLUSTER"
        --config "$tmpfile"
        --verbosity 4
    )
elif [ "$PROVIDER" == "gke" ]; then
    if [ -z "$GCP_PROJECT" ]; then
        echo "error: GCP_PROJECT is required"
        exit 1
    fi
    if [ -z "$GCP_CREDENTIALS" ]; then
        echo "error: GCP_CREDENTIALS is required"
        exit 1
    fi
    if [ -z "$GCP_REGION" -a -z "$GCP_ZONE" ]; then
        echo "error: either GCP_REGION or GCP_ZONE must be specified"
        exit 1
    elif [ -n "$GCP_REGION" -a -n "$GCP_ZONE" ]; then
        echo "error: GCP_REGION or GCP_ZONE cannot be both set"
        exit 1
    fi
    echo "info: activating GCP service account"
    gcloud auth activate-service-account --key-file "$GCP_CREDENTIALS"
    gcloud config set project "$GCP_PROJECT"
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
    ! read -r -d '' nodePoolsJSON <<EOF
{"default":{"Nodes":${KUBE_WORKERS},"MachineType":"${GCP_MACHINE_TYPE}"}}
EOF
    kubetest2_args+=(
        --cluster-name "$CLUSTER"
        --project "$GCP_PROJECT"
        --gcp-service-account "$GCP_CREDENTIALS"
        --environment prod
        --node-pools "$nodePoolsJSON"
    )
    if [ -n "$GCP_REGION" ]; then
        kubetest2_args+=(
            --region "$GCP_REGION"
        )
    fi
    if [ -n "$GCP_ZONE" ]; then
        kubetest2_args+=(
            --zone "$GCP_ZONE"
        )
    fi
elif [ "$PROVIDER" == "eks" ]; then
    export KUBE_SSH_USER=ec2-user
    hack::ensure_aws_k8s_tester
    if [ -z "$AWS_REGION" ]; then
        echo "error: AWS_REGION is required"
        exit 1
    fi
    if [ -z "$AWS_ZONE" ]; then
        AWS_ZONE=${AWS_REGION}a
    fi
    if [ -z "$AWS_ACCESS_KEY_ID" ]; then
        echo "error: AWS_ACCESS_KEY_ID is required"
        exit 1
    fi
    if [ -z "$AWS_SECRET_ACCESS_KEY" ]; then
        echo "error: AWS_SECRET_ACCESS_KEY is required"
        exit 1
    fi
    echo "info: activating AWS credentials"
    aws configure set default.region "$AWS_REGION"
    aws configure set aws_access_key_id "$AWS_ACCESS_KEY_ID"
    aws configure set aws_secret_access_key "$AWS_SECRET_ACCESS_KEY"
    if [ -z "$SKIP_UP" ]; then
        echo "info: make $HOME/.ssh/kube_aws_rsa writable if we're going to start the cluster"
        # aws-k8s-tester tries to create or update ~/.ssh/kube_aws_rsa with new key
        # pair every time, however previous created file is read-only and can't be
        # updated
        if test -f $HOME/.ssh/kube_aws_rsa; then
            chmod 0600 $HOME/.ssh/kube_aws_rsa
        fi
    fi
    echo "info: exporting AWS_K8S_TESTER config environments"
    mngName=$CLUSTER-mng
    export AWS_K8S_TESTER_EKS_NAME=$CLUSTER
    export AWS_K8S_TESTER_EKS_CONFIG_PATH=/tmp/kubetest2.eks.$CLUSTER
    export AWS_K8S_TESTER_EKS_PARAMETERS_VERSION="1.15"
    export AWS_K8S_TESTER_EKS_PARAMETERS_ENCRYPTION_CMK_CREATE="false"
    export AWS_K8S_TESTER_EKS_ADD_ON_MANAGED_NODE_GROUPS_ENABLE="true"
    export AWS_K8S_TESTER_EKS_ADD_ON_MANAGED_NODE_GROUPS_MNGS=$(printf '{"%s":{"name":"%s","ami-type":"AL2_x86_64","asg-min-size":%d,"asg-max-size":%d,"asg-desired-capacity":%d,"instance-types":["c5.xlarge"],"volume-size":40}}' "$mngName" "$mngName" "$KUBE_WORKERS" "$KUBE_WORKERS" "$KUBE_WORKERS")
    env | grep ^AWS_K8S_TESTER --color=never
    # override KUBECONFIG
    KUBECONFIG=$AWS_K8S_TESTER_EKS_CONFIG_PATH.kubeconfig.yaml
else
    echo "error: unsupported provider '$PROVIDER'"
    exit 1
fi

if [ "${HOSTNAME:-}" == "tidb-operator-dev" -a ! -f /usr/local/bin/helm ]; then
    ln -s $OUTPUT_BIN/helm /usr/local/bin/helm
fi

if [ "${HOSTNAME:-}" == "tidb-operator-dev" -a ! -f /usr/local/bin/kind ]; then
    ln -s $KIND_BIN /usr/local/bin/kind
fi

# Environments for hack/run-e2e.sh
export PROVIDER
export CLUSTER
export KUBECONFIG
export GCP_PROJECT
export GCP_REGION
export GCP_ZONE
export GCP_CREDENTIALS
export AWS_ACCESS_KEY_ID
export AWS_SECRET_ACCESS_KEY
export AWS_REGION
export AWS_ZONE
export IMAGE_TAG
export SKIP_GINKGO
export SKIP_IMAGE_LOAD
export TIDB_OPERATOR_IMAGE=$DOCKER_REPO/tidb-operator:${IMAGE_TAG}
export TIDB_BACKUP_MANAGER_IMAGE=$DOCKER_REPO/tidb-backup-manager:${IMAGE_TAG}
export E2E_IMAGE=$DOCKER_REPO/tidb-operator-e2e:${IMAGE_TAG}
export PATH=$OUTPUT_BIN:$PATH
export GIT_BRANCH
export GIT_COMMIT
export PR_ID

if [ -n "${ARTIFACTS}" ]; then
    export REPORT_DIR=${ARTIFACTS}
fi

if [ -n "${ARTIFACTS}" -a -z "$SKIP_DUMP" ]; then
    kubetest2_args+=(--dump)
fi

hack::ensure_kubetest2
echo "info: run 'kubetest2 ${kubetest2_args[@]} -- hack/run-e2e.sh $@'"
$KUBETSTS2_BIN ${kubetest2_args[@]} -- hack/run-e2e.sh "$@"
