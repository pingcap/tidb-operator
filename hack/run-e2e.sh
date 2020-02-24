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
IMAGE_TAG=${IMAGE_TAG:-}
TIDB_OPERATOR_IMAGE=${TIDB_OPERATOR_IMAGE:-localhost:5000/pingcap/tidb-operator:latest}
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

if [ -z "$KUBECONFIG" ]; then
    echo "error: KUBECONFIG is required"
    exit 1
fi

echo "TIDB_OPERATOR_IMAGE: $TIDB_OPERATOR_IMAGE"
echo "E2E_IMAGE: $E2E_IMAGE"
echo "KUBECONFIG: $KUBECONFIG"
echo "KUBECONTEXT: $KUBECONTEXT"
echo "REPORT_DIR: $REPORT_DIR"
echo "REPORT_PREFIX: $REPORT_PREFIX"
echo "GINKGO_NODES: $GINKGO_NODES"
echo "GINKGO_PARALLEL: $GINKGO_PARALLEL"
echo "GINKGO_NO_COLOR: $GINKGO_NO_COLOR"
echo "GINKGO_STREAM: $GINKGO_STREAM"

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
        # disks are created under /mnt/stateful_partition directory
        # https://cloud.google.com/container-optimized-os/docs/concepts/disks-and-filesystem
        for n in $($KUBECTL_BIN --context $KUBECONTEXT get nodes -ojsonpath='{range .items[*]}{.metadata.name}{"\n"}{end}'); do
            gcloud compute ssh $n --command 'sudo bash -c '"'"'
test -d /mnt/stateful_partition/disks || mkdir -p /mnt/stateful_partition/disks
df -h /mnt/stateful_partition/disks
test -d /mnt/disks || mkdir -p /mnt/disks
cd /mnt/disks
for ((i = 1; i <= 32; i++)) {
    if [ ! -d vol$i ]; then
        mkdir vol$i
    fi
    if ! mountpoint vol$i &>/dev/null; then
        if [ ! -d /mnt/stateful_partition/disks/vol$i ]; then
            mkdir /mnt/stateful_partition/disks/vol$i
        fi
        mount --bind /mnt/stateful_partition/disks/vol$i vol$i
    fi
}
'"'"
        done
    elif [ "$PROVIDER" == "eks" ]; then
        while IFS=$'\n' read -r line; do
            read -r id dns <<< $line
            echo "info: prepare disks on $dns"
            ssh -T -o "StrictHostKeyChecking no" -i ~/.ssh/kube_aws_rsa ec2-user@$dns <<'EOF'
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
echo "info: increase max open files for containers"
if ! grep -qF "OPTIONS" /etc/sysconfig/docker; then
    echo 'OPTIONS="--default-ulimit nofile=1024000:1024000"' >> /etc/sysconfig/docker
fi
systemctl restart docker
'
EOF
        done <<< "$(e2e::__eks_instances)"
    fi
    echo "info: installing local-volume-provisioner"
    $KUBECTL_BIN --context $KUBECONTEXT apply -f ${ROOT}/manifests/local-dind/local-volume-provisioner.yaml
    e2e::__wait_for_ds kube-system local-volume-provisioner
}

function e2e::__eks_instances() {
    aws ec2 describe-instances --filter Name=tag:eks:cluster-name,Values=$CLUSTER --query 'Reservations[*].Instances[*].{InstanceId:InstanceId,PublicDnsName:PublicDnsName}' --output text
}

function e2e::__ecr_url() {
    local account_id=$(aws sts get-caller-identity | awk '/Account/ { gsub("\x27", "", $2); print $2}')
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
        $HELM_BIN init --service-account tiller --output yaml \
            | sed 's@apiVersion: extensions/v1beta1@apiVersion: apps/v1@' \
            | sed 's@  replicas: 1@  replicas: 1\n  selector: {"matchLabels": {"app": "helm", "name": "tiller"}}@' \
            | $KUBECTL_BIN --context $KUBECONTEXT apply -f -
        echo "info: wait for tiller to be ready"
        e2e::__wait_for_deploy kube-system tiller-deploy
    else
        $HELM_BIN init --service-account=tiller --wait
    fi
    $HELM_BIN version
}

function e2e::image_load() {
    local images=(
        $TIDB_OPERATOR_IMAGE
        $E2E_IMAGE
    )
    if [ "$PROVIDER" == "kind" ]; then
        echo "info: load images ${images[@]}"
        for n in ${images[@]}; do
            $KIND_BIN load docker-image --name $CLUSTER $n
        done
    elif [ "$PROVIDER" == "gke" ]; then
        unset DOCKER_CONFIG # We don't need this and it may be read-only and fail the command to fail
        gcloud auth configure-docker
        GCP_TIDB_OPERATOR_IMAGE=gcr.io/$GCP_PROJECT/tidb-operator:$CLUSTER-$IMAGE_TAG
        GCP_E2E_IMAGE=gcr.io/$GCP_PROJECT/tidb-operator-e2e:$CLUSTER-$IMAGE_TAG
        docker tag $TIDB_OPERATOR_IMAGE $GCP_TIDB_OPERATOR_IMAGE
        docker tag $E2E_IMAGE $GCP_E2E_IMAGE
        echo "info: pushing $GCP_TIDB_OPERATOR_IMAGE"
        docker push $GCP_TIDB_OPERATOR_IMAGE
        echo "info: pushing $GCP_E2E_IMAGE"
        docker push $GCP_E2E_IMAGE
        TIDB_OPERATOR_IMAGE=$GCP_TIDB_OPERATOR_IMAGE
        E2E_IMAGE=$GCP_E2E_IMAGE
    elif [ "$PROVIDER" == "eks" ]; then
        for repoName in e2e/tidb-operator e2e/tidb-operator-e2e; do
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
        AWS_TIDB_OPERATOR_IMAGE=$ecrURL/e2e/tidb-operator:$CLUSTER-$IMAGE_TAG
        AWS_E2E_IMAGE=$ecrURL/e2e/tidb-operator-e2e:$CLUSTER-$IMAGE_TAG
        docker tag $TIDB_OPERATOR_IMAGE $AWS_TIDB_OPERATOR_IMAGE
        docker tag $E2E_IMAGE $AWS_E2E_IMAGE
        echo "info: pushing $AWS_TIDB_OPERATOR_IMAGE"
        docker push $AWS_TIDB_OPERATOR_IMAGE
        echo "info: pushing $AWS_E2E_IMAGE"
        docker push $AWS_E2E_IMAGE
        TIDB_OPERATOR_IMAGE=$AWS_TIDB_OPERATOR_IMAGE
        E2E_IMAGE=$AWS_E2E_IMAGE
    else
        echo "info: unsupported provider '$PROVIDER', skip loading images"
    fi
}

hack::ensure_kubectl
hack::ensure_helm

if [ -z "$KUBECONTEXT" ]; then
    KUBECONTEXT=$(kubectl config current-context)
    echo "info: KUBECONTEXT is not set, current context $KUBECONTEXT is used"
fi

e2e::image_load
e2e::setup_local_pvs
e2e::setup_helm_server

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
    --
    --provider=skeleton
    --clean-start=true
    --delete-namespace-on-failure=false
    --repo-root=$ROOT
    # tidb-operator e2e flags
    --operator-tag=e2e
    --operator-image=${TIDB_OPERATOR_IMAGE}
    --e2e-image=${E2E_IMAGE}
    # two tidb versions can be configuraed: <defaultVersion>,<upgradeToVersion>
    --tidb-versions=v3.0.7,v3.0.8
    --chart-dir=/charts
    -v=4
)

if [ -n "$REPORT_DIR" ]; then
    e2e_args+=(
        --report-dir="${REPORT_DIR}"
        --report-prefix="${REPORT_PREFIX}"
    )
fi

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
)

if [ "$PROVIDER" == "eks" ]; then
    # aws credential is required to get token for EKS
    docker_args+=(
        -v $HOME/.aws:/root/.aws
    )
fi

if [ -n "$REPORT_DIR" ]; then
    docker_args+=(
        -v $REPORT_DIR:$REPORT_DIR
    )
fi

echo "info: docker ${docker_args[@]} $E2E_IMAGE ${e2e_args[@]}"
docker ${docker_args[@]} $E2E_IMAGE ${e2e_args[@]}
