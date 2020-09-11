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

set -o errexit
set -o nounset
set -o pipefail

ROOT=$(unset CDPATH && cd $(dirname "${BASH_SOURCE[0]}")/.. && pwd)
cd $ROOT

source $ROOT/hack/lib.sh

PROVIDER=${PROVIDER:-}
CLUSTER=${CLUSTER:-}
GCP_PROJECT=${GCP_PROJECT:-}
GCP_REGION=${GCP_REGION:-}
GCP_ZONE=${GCP_ZONE:-}
GCP_CREDENTIALS=${GCP_CREDENTIALS:-}
GCP_SDK=${GCP_SDK:-/google-cloud-sdk}
KUBE_SSH_USER=${KUBE_SSH_USER:-vagrant}
IMAGE_TAG=${IMAGE_TAG:-}
SKIP_IMAGE_LOAD=${SKIP_IMAGE_LOAD:-}
TIDB_OPERATOR_IMAGE=${TIDB_OPERATOR_IMAGE:-localhost:5000/pingcap/tidb-operator:latest}
TIDB_BACKUP_MANAGER_IMAGE=${TIDB_BACKUP_MANAGER_IMAGE:-localhost:5000/pingcap/tidb-backup-manager:latest}
E2E_IMAGE=${E2E_IMAGE:-localhost:5000/pingcap/tidb-operator-e2e:latest}
KUBECONFIG=${KUBECONFIG:-$HOME/.kube/config}
KUBECONTEXT=${KUBECONTEXT:-}
REPORT_DIR=${REPORT_DIR:-}
REPORT_PREFIX=${REPORT_PREFIX:-}
GINKGO_NODES=${GINKGO_NODES:-}
GINKGO_PARALLEL=${GINKGO_PARALLEL:-n} # set to 'y' to run tests in parallel
# If 'y', Ginkgo's reporter will not print out in color when tests are run
# in parallel
GINKGO_NO_COLOR=${GINKGO_NO_COLOR:-n}
GINKGO_STREAM=${GINKGO_STREAM:-y}
SKIP_GINKGO=${SKIP_GINKGO:-}
# We don't delete namespace on failure by default for easier debugging in local development.
DELETE_NAMESPACE_ON_FAILURE=${DELETE_NAMESPACE_ON_FAILURE:-false}
GITHUB_RUN_ID=${GITHUB_RUN_ID:-}
BUILD_NUMBER=${BUILD_NUMBER:-$GITHUB_RUN_ID}
GIT_BRANCH=${GIT_BRANCH:-}
GIT_COMMIT=${GIT_COMMIT:-}
PR_ID=${PR_ID:-}
CODECOV_TOKEN=${CODECOV_TOKEN:-}

if [ -z "$KUBECONFIG" ]; then
    echo "error: KUBECONFIG is required"
    exit 1
fi

echo "KUBE_SSH_USER: $KUBE_SSH_USER"
echo "TIDB_OPERATOR_IMAGE: $TIDB_OPERATOR_IMAGE"
echo "TIDB_BACKUP_MANAGER_IMAGE: $TIDB_BACKUP_MANAGER_IMAGE"
echo "E2E_IMAGE: $E2E_IMAGE"
echo "KUBECONFIG: $KUBECONFIG"
echo "KUBECONTEXT: $KUBECONTEXT"
echo "REPORT_DIR: $REPORT_DIR"
echo "REPORT_PREFIX: $REPORT_PREFIX"
echo "GINKGO_NODES: $GINKGO_NODES"
echo "GINKGO_PARALLEL: $GINKGO_PARALLEL"
echo "GINKGO_NO_COLOR: $GINKGO_NO_COLOR"
echo "GINKGO_STREAM: $GINKGO_STREAM"
echo "DELETE_NAMESPACE_ON_FAILURE: $DELETE_NAMESPACE_ON_FAILURE"
echo "GIT_BRANCH: $GIT_BRANCH"
echo "GIT_COMMIT: $GIT_COMMIT"
echo "PR_ID: $PR_ID"

function e2e::__wait_for_ds() {
    local ns="$1"
    local name="$2"
    local retries="${3:-300}"
    echo "info: waiting for pods of daemonset $ns/$name are ready (retries: $retries, interval: 1s)"
    for ((i = 0; i < retries; i++)) {
        read a b <<<$($KUBECTL_BIN --context $KUBECONTEXT -n $ns get ds/$name -ojsonpath='{.status.desiredNumberScheduled} {.status.numberReady}{"\n"}')
        if [[ "$a" -gt 0 && "$a" -eq "$b" ]]; then
            echo "info: all pods of daemonset $ns/$name are ready (desired: $a, ready: $b)"
            return 0
        fi
        echo "info: pods of daemonset $ns/$name (desired: $a, ready: $b)"
        sleep 1
    }
    echo "info: timed out waiting for pods of daemonset $ns/$name are ready"
    return 1
}

function e2e::__wait_for_deploy() {
    local ns="$1"
    local name="$2"
    local retries="${3:-300}"
    echo "info: waiting for pods of deployment $ns/$name are ready (retries: $retries, interval: 1s)"
    for ((i = 0; i < retries; i++)) {
        read a b <<<$($KUBECTL_BIN --context $KUBECONTEXT -n $ns get deploy/$name -ojsonpath='{.spec.replicas} {.status.readyReplicas}{"\n"}')
        if [[ "$a" -gt 0 && "$a" -eq "$b" ]]; then
            echo "info: all pods of deployment $ns/$name are ready (desired: $a, ready: $b)"
            return 0
        fi
        echo "info: pods of deployment $ns/$name (desired: $a, ready: $b)"
        sleep 1
    }
    echo "info: timed out waiting for pods of deployment $ns/$name are ready"
    return 1
}

function e2e::setup_local_pvs() {
    echo "info: preparing local disks"
    if [ "$PROVIDER" == "kind" ]; then
        for n in $($KIND_BIN get nodes --name=$CLUSTER); do
            docker exec -i $n bash <<'EOF'
test -d /mnt/disks || mkdir -p /mnt/disks
df -h /mnt/disks
if mountpoint /mnt/disks &>/dev/null; then
    echo "info: /mnt/disks is a mountpoint"
else
    echo "info: /mnt/disks is not a mountpoint, creating local volumes on the rootfs"
fi
cd /mnt/disks
for ((i = 1; i <= 32; i++)) {
    if [ ! -d vol$i ]; then
        mkdir vol$i
    fi
    if ! mountpoint vol$i &>/dev/null; then
        mount --bind vol$i vol$i
    fi
}
EOF
        done
    elif [ "$PROVIDER" == "gke" ]; then
        echo "info: provider is $PROVIDER, skipped"
    elif [ "$PROVIDER" == "eks" ]; then
        echo "info: provider is $PROVIDER, skipped"
    elif [ "$PROVIDER" == "openshift" ]; then
        CRC_IP=$(crc ip)
        ssh -i ~/.crc/machines/crc/id_rsa -o StrictHostKeyChecking=no core@$CRC_IP <<'EOF'
sudo bash -c '
test -d /mnt/disks || mkdir -p /mnt/disks
df -h /mnt/disks
if mountpoint /mnt/disks &>/dev/null; then
    echo "info: /mnt/disks is a mountpoint"
else
    echo "info: /mnt/disks is not a mountpoint, creating local volumes on the rootfs"
fi
cd /mnt/disks
for ((i = 1; i <= 32; i++)) {
    if [ ! -d vol$i ]; then
        mkdir vol$i
    fi
    if ! mountpoint vol$i &>/dev/null; then
        mount --bind vol$i vol$i
    fi
}
'
EOF
    fi
    echo "info: installing local-volume-provisioner"
    $KUBECTL_BIN --context $KUBECONTEXT apply -f ${ROOT}/manifests/local-dind/local-volume-provisioner.yaml
    e2e::__wait_for_ds kube-system local-volume-provisioner
}

function e2e::__ecr_url() {
    local account_id=$(aws sts get-caller-identity --output text | awk '{print $1}')
    local region=$(aws configure get region)
    echo "${account_id}.dkr.ecr.${region}.amazonaws.com"
}

function e2e::get_kube_version() {
    $KUBECTL_BIN --context $KUBECONTEXT version --short | awk '/Server Version:/ {print $3}'
}

function e2e::setup_helm_server() {
    $KUBECTL_BIN --context $KUBECONTEXT apply -f ${ROOT}/manifests/tiller-rbac.yaml
    if hack::version_ge $(e2e::get_kube_version) "v1.16.0"; then
        # workaround for https://github.com/helm/helm/issues/6374
        # TODO remove this when we can upgrade to helm 2.15+, see https://github.com/helm/helm/pull/6462
        # \'$'\n is used to be compatible with BSD sed (Darwin)
        $HELM_BIN init --service-account tiller --output yaml \
            | sed 's@apiVersion: extensions/v1beta1@apiVersion: apps/v1@' \
            | sed 's@  replicas: 1@  replicas: 1\'$'\n  selector: {"matchLabels": {"app": "helm", "name": "tiller"}}@' \
            | $KUBECTL_BIN --context $KUBECONTEXT apply -f -
        echo "info: wait for tiller to be ready"
        e2e::__wait_for_deploy kube-system tiller-deploy
    else
        $HELM_BIN init --service-account=tiller --skip-refresh --wait
    fi
    $HELM_BIN version
}

# Used by non-kind providers to tag image with its id. This can force our e2e
# process to pull correct image even if IfNotPresent is used in an existing
# environment, e.g. testing in the same cluster.
function e2e::image_id_tag() {
    docker image inspect -f '{{.Id}}' "$1" | cut -d ':' -f 2 | head -c 10
}

function e2e::image_load() {
    local images=(
        $TIDB_OPERATOR_IMAGE
        $TIDB_BACKUP_MANAGER_IMAGE
        $E2E_IMAGE
    )
    echo "info: pull if images do not exist"
    for image in ${images[@]}; do
        if ! docker inspect -f '{{.Id}}' $image &>/dev/null; then
            echo "info: pulling $image"
            docker pull $image
        fi
    done
    if [ "$PROVIDER" == "kind" ]; then
        local nodes=$($KIND_BIN get nodes --name $CLUSTER | grep -v 'control-plane$')
        if [ -z "$nodes" ]; then
            # if no workers are found, we need to schedule pods to control-plane.
            local nodes=$($KIND_BIN get nodes --name $CLUSTER)
        fi
        echo "info: load images ${images[@]}"
        for n in ${images[@]}; do
            $KIND_BIN load docker-image --name $CLUSTER $n --nodes $(hack::join ',' ${nodes[@]})
        done
    elif [ "$PROVIDER" == "gke" ]; then
        unset DOCKER_CONFIG # We don't need this and it may be read-only and fail the command to fail
        gcloud auth configure-docker
        GCP_TIDB_OPERATOR_IMAGE=gcr.io/$GCP_PROJECT/tidb-operator:$CLUSTER-$(e2e::image_id_tag $TIDB_OPERATOR_IMAGE)
        GCP_TIDB_BACKUP_MANAGER_IMAGE=gcr.io/$GCP_PROJECT/tidb-backup-image:$CLUSTER-$(e2e::image_id_tag $TIDB_BACKUP_MANAGER_IMAGE)
        GCP_E2E_IMAGE=gcr.io/$GCP_PROJECT/tidb-operator-e2e:$CLUSTER-$(e2e::image_id_tag $E2E_IMAGE)
        docker tag $TIDB_OPERATOR_IMAGE $GCP_TIDB_OPERATOR_IMAGE
        docker tag $E2E_IMAGE $GCP_E2E_IMAGE
        docker tag $TIDB_BACKUP_MANAGER_IMAGE $GCP_TIDB_BACKUP_MANAGER_IMAGE
        echo "info: pushing $GCP_TIDB_OPERATOR_IMAGE"
        docker push $GCP_TIDB_OPERATOR_IMAGE
        echo "info: pushing $GCP_E2E_IMAGE"
        docker push $GCP_E2E_IMAGE
        echo "info: pushing $GCP_TIDB_BACKUP_MANAGER_IMAGE"
        docker push $GCP_TIDB_BACKUP_MANAGER_IMAGE
        TIDB_OPERATOR_IMAGE=$GCP_TIDB_OPERATOR_IMAGE
        E2E_IMAGE=$GCP_E2E_IMAGE
        TIDB_BACKUP_MANAGER_IMAGE=$GCP_TIDB_BACKUP_MANAGER_IMAGE
    elif [ "$PROVIDER" == "eks" ]; then
        for repoName in e2e/tidb-operator e2e/tidb-operator-e2e e2e/tidb-backup-manager; do
            local ret=0
            aws ecr describe-repositories --repository-names $repoName || ret=$?
            if [ $ret -ne 0 ]; then
                echo "info: creating repository $repoName"
                aws ecr create-repository --repository-name $repoName
            fi
        done
        local ecrURL=$(e2e::__ecr_url)
        echo "info: logging in $ecrURL"
        aws ecr get-login-password | docker login --username AWS --password-stdin $ecrURL
        AWS_TIDB_OPERATOR_IMAGE=$ecrURL/e2e/tidb-operator:$CLUSTER-$(e2e::image_id_tag $TIDB_OPERATOR_IMAGE)
        AWS_TIDB_BACKUP_MANAGER_IMAGE=$ecrURL/e2e/tidb-backup-manager:$CLUSTER-$(e2e::image_id_tag $TIDB_BACKUP_MANAGER_IMAGE)
        AWS_E2E_IMAGE=$ecrURL/e2e/tidb-operator-e2e:$CLUSTER-$(e2e::image_id_tag $E2E_IMAGE)
        docker tag $TIDB_OPERATOR_IMAGE $AWS_TIDB_OPERATOR_IMAGE
        docker tag $TIDB_BACKUP_MANAGER_IMAGE $AWS_TIDB_BACKUP_MANAGER_IMAGE
        docker tag $E2E_IMAGE $AWS_E2E_IMAGE
        echo "info: pushing $AWS_TIDB_OPERATOR_IMAGE"
        docker push $AWS_TIDB_OPERATOR_IMAGE
        echo "info: pushing $AWS_TIDB_BACKUP_MANAGER_IMAGE"
        docker push $AWS_TIDB_BACKUP_MANAGER_IMAGE
        echo "info: pushing $AWS_E2E_IMAGE"
        docker push $AWS_E2E_IMAGE
        TIDB_BACKUP_MANAGER_IMAGE=$AWS_TIDB_BACKUP_MANAGER_IMAGE
        TIDB_OPERATOR_IMAGE=$AWS_TIDB_OPERATOR_IMAGE
        E2E_IMAGE=$AWS_E2E_IMAGE
    else
        echo "info: unsupported provider '$PROVIDER', skip loading images"
    fi
}

hack::ensure_kubectl
hack::ensure_helm

if [ "$PROVIDER" == "gke" ]; then
    if [ -n "$GCP_CREDENTIALS" ]; then
        gcloud auth activate-service-account --key-file "$GCP_CREDENTIALS"
    fi
    if [ -n "$GCP_REGION" ]; then
        gcloud config set compute/region "$GCP_REGION"
    fi
    if [ -n "$GCP_ZONE" ]; then
        gcloud config set compute/zone "$GCP_ZONE"
    fi
    gcloud container clusters get-credentials "$CLUSTER"
elif [ "$PROVIDER" == "eks" ]; then
    aws eks update-kubeconfig --name "$CLUSTER"
fi

if [ -z "$KUBECONTEXT" ]; then
    echo "info: KUBECONTEXT is not set, current context is used"
    KUBECONTEXT=$($KUBECTL_BIN config current-context 2>/dev/null) || true
    if [ -z "$KUBECONTEXT" ]; then
        echo "error: current context cannot be detected"
        exit 1
    fi
    echo "info: current kubeconfig context is '$KUBECONTEXT'"
fi

if [ -z "$SKIP_IMAGE_LOAD" ]; then
    e2e::image_load
fi

e2e::setup_local_pvs
e2e::setup_helm_server

if [ -n "$SKIP_GINKGO" ]; then
    echo "info: skipping ginkgo"
    exit 0
fi

echo "info: start to run e2e process"

ginkgo_args=()

if [[ -n "${GINKGO_NODES:-}" ]]; then
    ginkgo_args+=("--nodes=${GINKGO_NODES}")
elif [[ ${GINKGO_PARALLEL} =~ ^[yY]$ ]]; then
    ginkgo_args+=("-p")
fi

if [[ "${GINKGO_NO_COLOR}" == "y" ]]; then
    ginkgo_args+=("--noColor")
fi

if [[ "${GINKGO_STREAM}" == "y" ]]; then
    ginkgo_args+=("--stream")
fi

e2e_args=(
    /usr/local/bin/ginkgo
    ${ginkgo_args[@]:-}
    /usr/local/bin/e2e.test
    -test.coverprofile=/tmp/e2e.coverage.txt
    --
    --clean-start=true
    --delete-namespace-on-failure="${DELETE_NAMESPACE_ON_FAILURE}"
    --repo-root="${ROOT}"
    # tidb-operator e2e flags
    --operator-tag=e2e
    --operator-image="${TIDB_OPERATOR_IMAGE}"
    --backup-image="${TIDB_BACKUP_MANAGER_IMAGE}"
    --e2e-image="${E2E_IMAGE}"
    --chart-dir=/charts
    -v=4
)

e2e_args+=(${@:-})

docker_args=(
    run
    --rm
    --net=host
    --privileged
    -v /:/rootfs
    -v $ROOT:$ROOT
    -w $ROOT
    -v $KUBECONFIG:/etc/kubernetes/admin.conf:ro
    --env KUBECONFIG=/etc/kubernetes/admin.conf
    --env KUBECONTEXT=$KUBECONTEXT
    --env KUBE_SSH_USER=$KUBE_SSH_USER
    --env BUILD=$BUILD_NUMBER
    --env BRANCH=$GIT_BRANCH
    --env COMMIT=$GIT_COMMIT
    --env PR=$PR_ID
    --env CODECOV_TOKEN=$CODECOV_TOKEN
)

if [ "$PROVIDER" == "eks" ]; then
    e2e_args+=(
        --provider=aws
        --gce-zone="${AWS_ZONE}" # reuse gce-zone to configure aws zone
    )
    docker_args+=(
        # aws credential is required to get token for EKS
        -v $HOME/.aws:/root/.aws
        # ~/.ssh/kube_aws_rsa must be mounted into e2e container to run ssh
        -v $HOME/.ssh/kube_aws_rsa:/root/.ssh/kube_aws_rsa
    )
elif [ "$PROVIDER" == "gke" ]; then
    e2e_args+=(
        --provider="${PROVIDER}"
        --gce-project="${GCP_PROJECT}"
        --gce-region="${GCP_REGION}"
        --gce-zone="${GCP_ZONE}"
        --gke-cluster="${CLUSTER}"
    )
    docker_args+=(
        -v ${GCP_CREDENTIALS}:${GCP_CREDENTIALS}
        --env GOOGLE_APPLICATION_CREDENTIALS=${GCP_CREDENTIALS}
    )
    # google-cloud-sdk is very large, we didn't pack it into our e2e image.
    # instead, we use the sdk installed in CI image.
    if [ ! -e "${GCP_SDK}/bin/gcloud" ]; then
        echo "error: ${GCP_SDK} is not google cloud sdk, please install it here or specify correct path via GCP_SDK env"
        exit 1
    fi
    docker_args+=(
        # gcloud config
        -v $HOME/.config/gcloud:/root/.config/gcloud
        -v ${GCP_SDK}:/google-cloud-sdk
        # ~/.ssh/google_compute_engine must be mounted into e2e container to run ssh
        -v $HOME/.ssh/google_compute_engine:/root/.ssh/google_compute_engine
    )
else
    e2e_args+=(
        --provider="${PROVIDER}"
    )
fi

if [ -n "$REPORT_DIR" ]; then
    e2e_args+=(
        --report-dir="${REPORT_DIR}"
        --report-prefix="${REPORT_PREFIX}"
    )
    docker_args+=(
        -v $REPORT_DIR:$REPORT_DIR
    )
fi

echo "info: docker ${docker_args[@]} $E2E_IMAGE ${e2e_args[@]}"
docker ${docker_args[@]} $E2E_IMAGE ${e2e_args[@]}
