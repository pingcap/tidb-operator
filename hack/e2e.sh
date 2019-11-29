#!/usr/bin/env bash
#
# E2E entrypoint script.
#

set -o errexit
set -o nounset
set -o pipefail

ROOT=$(unset CDPATH && cd $(dirname "${BASH_SOURCE[0]}")/.. && pwd)
cd $ROOT

source "${ROOT}/hack/lib.sh"

function usage() {
    cat <<'EOF'
This script is entrypoint to run e2e tests.

Usage: hack/e2e.sh [-h] -- [extra test args]

    -h      show this message and exit

Environments:

    DOCKER_REGISTRY     image docker registry
    IMAGE_TAG           image tag
    SKIP_BUILD          skip building binaries
    SKIP_IMAGE_BUILD    skip build and push images
    SKIP_UP             skip starting the cluster
    SKIP_DOWN           skip shutting down the cluster
    KUBE_VERSION        the version of Kubernetes to test against
    KUBE_WORKERS        the number of worker nodes (excludes master nodes), defaults: 3
    KIND_DATA_HOSTPATH  (for kind) the host path of data directory for kind cluster, defaults: none
    GINKGO_NODES        ginkgo nodes to run specs, defaults: 1
    GINKGO_PARALLEL     if set to `y`, will run specs in parallel, the number of nodes will be the number of cpus
    GINKGO_NO_COLOR     if set to `y`, suppress color output in default reporter

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

hack::ensure_kind
hack::ensure_kubectl
hack::ensure_helm

DOCKER_REGISTRY=${DOCKER_REGISTRY:-localhost:5000}
IMAGE_TAG=${IMAGE_TAG:-latest}
CLUSTER=${CLUSTER:-tidb-operator}
KUBECONFIG=${KUBECONFIG:-~/.kube/config}
KUBECONTEXT=kind-$CLUSTER
SKIP_BUILD=${SKIP_BUILD:-}
SKIP_IMAGE_BUILD=${SKIP_IMAGE_BUILD:-}
SKIP_UP=${SKIP_UP:-}
SKIP_DOWN=${SKIP_DOWN:-}
KIND_DATA_HOSTPATH=${KIND_DATA_HOSTPATH:-none}
KUBE_VERSION=${KUBE_VERSION:-v1.12.10}
KUBE_WORKERS=${KUBE_WORKERS:-3}

echo "DOCKER_REGISTRY: $DOCKER_REGISTRY"
echo "IMAGE_TAG: $IMAGE_TAG"
echo "CLUSTER: $CLUSTER"
echo "KUBECONFIG: $KUBECONFIG"
echo "KUBECONTEXT: $KUBECONTEXT"
echo "SKIP_BUILD: $SKIP_BUILD"
echo "SKIP_IMAGE_BUILD: $SKIP_IMAGE_BUILD"
echo "SKIP_UP: $SKIP_UP"
echo "SKIP_DOWN: $SKIP_DOWN"
echo "KIND_DATA_HOSTPATH: $KIND_DATA_HOSTPATH"
echo "KUBE_VERSION: $KUBE_VERSION"

# https://github.com/kubernetes-sigs/kind/releases/tag/v0.6.0
declare -A kind_node_images
kind_node_images["v1.11.10"]="kindest/node:v1.11.10@sha256:44e1023d3a42281c69c255958e09264b5ac787c20a7b95caf2d23f8d8f3746f2"
kind_node_images["v1.12.10"]="kindest/node:v1.12.10@sha256:e93e70143f22856bd652f03da880bfc70902b736750f0a68e5e66d70de236e40"
kind_node_images["v1.13.12"]="kindest/node:v1.13.12@sha256:ad1dd06aca2b85601f882ba1df4fdc03d5a57b304652d0e81476580310ba6289"
kind_node_images["v1.14.9"]="kindest/node:v1.14.9@sha256:00fb7d424076ed07c157eedaa3dd36bc478384c6d7635c5755746f151359320f"
kind_node_images["v1.15.6"]="kindest/node:v1.15.6@sha256:1c8ceac6e6b48ea74cecae732e6ef108bc7864d8eca8d211d6efb58d6566c40a"
kind_node_images["v1.16.3"]="kindest/node:v1.16.3@sha256:bced4bc71380b59873ea3917afe9fb35b00e174d22f50c7cab9188eac2b0fb88"

function e2e::image_build() {
    if [ -n "$SKIP_BUILD" ]; then
        echo "info: skip building images"
        export NO_BUILD=y
    fi
    if [ -n "$SKIP_IMAGE_BUILD" ]; then
        echo "info: skip building and pushing images"
        return
    fi
    DOCKER_REGISTRY=$DOCKER_REGISTRY IMAGE_TAG=$IMAGE_TAG make docker
    DOCKER_REGISTRY=$DOCKER_REGISTRY IMAGE_TAG=$IMAGE_TAG make e2e-docker
    DOCKER_REGISTRY=$DOCKER_REGISTRY IMAGE_TAG=$IMAGE_TAG make test-apiesrver-docker
}

function e2e::image_load() {
    local names=(
        pingcap/tidb-operator
        pingcap/tidb-operator-e2e
        pingcap/test-apiserver
    )
    for n in ${names[@]}; do
        $KIND_BIN load docker-image --name $CLUSTER $DOCKER_REGISTRY/$n:$IMAGE_TAG
    done
}

function e2e::cluster_exists() {
    local name="$1"
    $KIND_BIN get clusters | grep $CLUSTER &>/dev/null
}

function e2e::up() {
    if [ -n "$SKIP_UP" ]; then
        echo "info: skip starting a new cluster"
        return
    fi
    if e2e::cluster_exists $CLUSTER; then
        echo "info: deleting the cluster '$CLUSTER'"
        $KIND_BIN delete cluster --name $CLUSTER
    fi
    echo "info: starting a new cluster"
    tmpfile=$(mktemp)
    trap "test -f $tmpfile && rm $tmpfile" RETURN
    cat <<EOF > $tmpfile
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
EOF
    # control-plane
    cat <<EOF >> $tmpfile
nodes:
- role: control-plane
EOF
    if [[ "$KIND_DATA_HOSTPATH" != "none" ]]; then
        if [ ! -d "$KIND_DATA_HOSTPATH" ]; then
            echo "error: '$KIND_DATA_HOSTPATH' is not a directory"
            exit 1
        fi
        local hostWorkerPath="${KIND_DATA_HOSTPATH}/control-plane"
        test -d $hostWorkerPath || mkdir $hostWorkerPath
        cat <<EOF >> $tmpfile
  extraMounts:
  - containerPath: /mnt/disks/
    hostPath: "$hostWorkerPath"
    propagation: HostToContainer
EOF
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
    echo "info: print the contents of kindconfig"
    cat $tmpfile
    echo "info: end of the contents of kindconfig"
    echo "info: creating the cluster '$CLUSTER'"
    local image=""
    for v in ${!kind_node_images[*]}; do
        if [[ "$KUBE_VERSION" == "$v" ]]; then
            image=${kind_node_images[$v]}
            echo "info: image for $KUBE_VERSION: $image"
            break
        fi
    done
    if [ -z "$image" ]; then
        echo "error: no image for $KUBE_VERSION, exit"
        exit 1
    fi
    $KIND_BIN create cluster --config $KUBECONFIG --name $CLUSTER --image $image --config $tmpfile -v 4
    # make it able to schedule pods on control-plane, then less resources we required
    # This is disabled because when hostNetwork is used, pd requires 2379/2780
    # which may conflict with etcd on control-plane.
    #echo "info: remove 'node-role.kubernetes.io/master' taint from $CLUSTER-control-plane"
    #kubectl taint nodes $CLUSTER-control-plane node-role.kubernetes.io/master-
}

function e2e::__wait_for_ds() {
    local ns="$1"
    local name="$2"
    local retries="${3:-120}"
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

function e2e::setup_local_pvs() {
    echo "info: preparing disks"
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
    echo "info: installing local-volume-provisioner"
    $KUBECTL_BIN apply -f ${ROOT}/manifests/local-dind/local-volume-provisioner.yaml
    e2e::__wait_for_ds kube-system local-volume-provisioner
}

function e2e::setup_helm_server() {
    $KUBECTL_BIN apply -f ${ROOT}/manifests/tiller-rbac.yaml
    $HELM_BIN init --service-account=tiller --wait
    $HELM_BIN version
}

function e2e::down() {
    if [ -n "$SKIP_DOWN" ]; then
        echo "info: skip shutting down the cluster '$CLUSTER'"
        return
    fi
    if ! e2e::cluster_exists $CLUSTER; then
        echo "info: cluster '$CLUSTER' does not exist, skip shutting down the cluster"
        return
    fi
    $KIND_BIN delete cluster --name $CLUSTER
}

trap "e2e::down" EXIT
e2e::up
e2e::setup_local_pvs
e2e::setup_helm_server
e2e::image_build
e2e::image_load

export KUBECONFIG
export KUBECONTEXT
export TIDB_OPERATOR_IMAGE=$DOCKER_REGISTRY/pingcap/tidb-operator:${IMAGE_TAG}
export E2E_IMAGE=$DOCKER_REGISTRY/pingcap/tidb-operator-e2e:${IMAGE_TAG}
export TEST_APISERVER_IMAGE=$DOCKER_REGISTRY/pingcap/test-apiserver:${IMAGE_TAG}

hack/run-e2e.sh "$@"
