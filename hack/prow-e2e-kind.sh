#!/usr/bin/env bash

# Copyright 2025 PingCAP, Inc.
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

# Default parameters (override via --key=value)
GINKGO_FOCUS="${GINKGO_FOCUS:-}"
GINKGO_SKIP="${GINKGO_SKIP:-}"
GINKGO_NODES="${GINKGO_NODES:-1}"
DELETE_NAMESPACE_ON_FAILURE="${DELETE_NAMESPACE_ON_FAILURE:-false}"
E2E_EXTRA_ARGS="${E2E_EXTRA_ARGS:-}"
KIND_CP_MEM="${KIND_CP_MEM:-2G}"
KIND_CP_CPU="${KIND_CP_CPU:-4096}"
KIND_WK_MEM="${KIND_WK_MEM:-8G}"
KIND_WK_CPU="${KIND_WK_CPU:-4096}"
TEST_TIMEOUT="${TEST_TIMEOUT:-7200}"

function usage() {
    cat <<'EOF'
Usage: hack/prow-e2e-kind.sh [options]

Runs tidb-operator e2e tests in a Kind cluster for Prow CI.

Options (also via env vars):
  --focus=FOCUS           Ginkgo focus regex (required)
  --skip=SKIP             Ginkgo skip regex (optional)
  --nodes=N               GINKGO_NODES (default: 1)
  --delete-namespace       Set DELETE_NAMESPACE_ON_FAILURE=true
  --extra-args=ARGS       Extra args for e2e.sh, e.g. "--install-dm-mysql=false"
  --kind-cp-mem=MEM       Kind control-plane memory reservation (default: 2G)
  --kind-cp-cpu=N         Kind control-plane cpu shares (default: 4096)
  --kind-wk-mem=MEM       Kind worker memory limit (default: 8G)
  --kind-wk-cpu=N         Kind worker cpu shares (default: 4096)
  --timeout=N             Test timeout seconds (default: 7200)
  -h, --help              Show this help

Examples:
  ./hack/prow-e2e-kind.sh --focus='TiDBCluster' --skip='\[TiDBCluster:\sBasic\]' --nodes=4
  ./hack/prow-e2e-kind.sh --focus='DMCluster' --nodes=2
  ./hack/prow-e2e-kind.sh --focus='\[Serial\]' --extra-args='--install-operator=false' --delete-namespace
EOF
}

# Parse arguments
for arg in "$@"; do
    case "$arg" in
        -h|--help)
            usage
            exit 0
            ;;
        --focus=*)
            GINKGO_FOCUS="${arg#*=}"
            ;;
        --skip=*)
            GINKGO_SKIP="${arg#*=}"
            ;;
        --nodes=*)
            GINKGO_NODES="${arg#*=}"
            ;;
        --delete-namespace)
            DELETE_NAMESPACE_ON_FAILURE="true"
            ;;
        --extra-args=*)
            E2E_EXTRA_ARGS="${arg#*=}"
            ;;
        --kind-cp-mem=*)
            KIND_CP_MEM="${arg#*=}"
            ;;
        --kind-cp-cpu=*)
            KIND_CP_CPU="${arg#*=}"
            ;;
        --kind-wk-mem=*)
            KIND_WK_MEM="${arg#*=}"
            ;;
        --kind-wk-cpu=*)
            KIND_WK_CPU="${arg#*=}"
            ;;
        --timeout=*)
            TEST_TIMEOUT="${arg#*=}"
            ;;
        *)
            echo "Unknown option: $arg" >&2
            usage >&2
            exit 1
            ;;
    esac
done

if [ -z "$GINKGO_FOCUS" ]; then
    echo "Error: --focus is required" >&2
    usage >&2
    exit 1
fi

ROOT=$(unset CDPATH && cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
cd "$ROOT"

echo "Prow E2E Kind: focus=$GINKGO_FOCUS skip=$GINKGO_SKIP nodes=$GINKGO_NODES"
echo "Kind resources: cp mem=$KIND_CP_MEM cpu=$KIND_CP_CPU, worker mem=$KIND_WK_MEM cpu=$KIND_WK_CPU"

# Install dependencies if not present
export DEBIAN_FRONTEND=noninteractive
if ! command -v go >/dev/null 2>&1; then
    echo "Installing Go..."
    apt-get update -qq
    apt-get install -y -qq --no-install-recommends ca-certificates curl tar
    curl -fsSL https://go.dev/dl/go1.23.10.linux-amd64.tar.gz | tar -C /usr/local -xz
fi
export PATH=/usr/local/go/bin:${PATH:-}
export GOPATH=${GOPATH:-/go}
export GOCACHE=${GOCACHE:-/tmp/go-cache}

if ! command -v docker >/dev/null 2>&1; then
    echo "Installing Docker CLI..."
    apt-get update -qq
    apt-get install -y -qq --no-install-recommends ca-certificates curl
    install -m 0755 -d /etc/apt/keyrings
    curl -fsSL https://download.docker.com/linux/ubuntu/gpg -o /etc/apt/keyrings/docker.asc
    chmod a+r /etc/apt/keyrings/docker.asc
    echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.asc] https://download.docker.com/linux/ubuntu $(. /etc/os-release 2>/dev/null && echo "$VERSION_CODENAME") stable" | tee /etc/apt/sources.list.d/docker.list > /dev/null
    apt-get update -qq
    apt-get install -y -qq --no-install-recommends docker-ce-cli docker-buildx-plugin
fi

if ! command -v make >/dev/null 2>&1; then
    echo "Installing build tools..."
    apt-get update -qq
    apt-get install -y -qq --no-install-recommends make cmake git jq coreutils
fi

echo "Waiting for Docker daemon..."
timeout_seconds=120
start_time=$(date +%s)
while ! docker info >/dev/null 2>&1; do
    current_time=$(date +%s)
    elapsed=$((current_time - start_time))
    if [ "$elapsed" -ge "$timeout_seconds" ]; then
        echo "Error: Docker daemon did not start within ${timeout_seconds} seconds." >&2
        exit 1
    fi
    sleep 5
done
echo "Docker daemon is responsive."

# Build and prepare
./hack/e2e-patch-codecov.sh || true
unset GOSUMDB
E2E=y make build e2e-build
make gocovmerge || true

GITHASH=$(git rev-parse HEAD)
IMAGE_TAG="e2e-${GITHASH:0:6}"
echo "Building images locally..."
E2E=y DOCKER_REPO=tidb-operator-e2e IMAGE_TAG="${IMAGE_TAG}" make docker e2e-docker

# Kind cluster setup
mount --make-rshared / || true
mkdir -p /kind-data/control-plane/coverage
mkdir -p /kind-data/worker1/coverage
mkdir -p /kind-data/worker2/coverage
mkdir -p /kind-data/worker3/coverage
mkdir -p /mnt/tmpfs/etcd

PROVIDER=kind KIND_DATA_HOSTPATH=/kind-data KIND_ETCD_DATADIR=/mnt/tmpfs/etcd \
    SKIP_BUILD=y SKIP_IMAGE_BUILD=y SKIP_TEST=y SKIP_DOWN=y ./hack/e2e.sh

test -f /root/.kube/config || { echo "ERROR: /root/.kube/config not found"; exit 1; }
install -d -m 0755 /kubeconfig
install -m 0644 /root/.kube/config /kubeconfig/admin.conf
export KUBECONFIG=/kubeconfig/admin.conf

echo "Creating coverage directories in kind nodes..."
docker exec tidb-operator-control-plane mkdir -p /mnt/disks/coverage || true
docker exec tidb-operator-worker mkdir -p /mnt/disks/coverage || true
docker exec tidb-operator-worker2 mkdir -p /mnt/disks/coverage || true
docker exec tidb-operator-worker3 mkdir -p /mnt/disks/coverage || true

KIND_BIN="kind"
if ! command -v kind >/dev/null 2>&1; then
    KIND_VERSION="${KIND_VERSION:-v0.19.0}"
    curl -fsSL "https://kind.sigs.k8s.io/dl/${KIND_VERSION}/kind-linux-amd64" -o /usr/local/bin/kind
    chmod +x /usr/local/bin/kind
    KIND_BIN="/usr/local/bin/kind"
fi
echo "Loading images into kind cluster..."
for img in $(docker images --format "{{.Repository}}:{{.Tag}}" | grep "tidb-operator-e2e/.*:${IMAGE_TAG}$"); do
    "${KIND_BIN}" load docker-image "${img}" --name tidb-operator || { echo "Failed to load ${img} into kind"; exit 1; }
done

# Adjust kind node resources
echo "Updating kind node resources..."
docker update tidb-operator-control-plane --memory-reservation="${KIND_CP_MEM}" -c "${KIND_CP_CPU}"
docker update tidb-operator-worker -m "${KIND_WK_MEM}" --memory-swap "${KIND_WK_MEM}" -c "${KIND_WK_CPU}"
docker update tidb-operator-worker2 -m "${KIND_WK_MEM}" --memory-swap "${KIND_WK_MEM}" -c "${KIND_WK_CPU}"
docker update tidb-operator-worker3 -m "${KIND_WK_MEM}" --memory-swap "${KIND_WK_MEM}" -c "${KIND_WK_CPU}"

# Run tests
export GINKGO_NO_COLOR=y
export DELETE_NAMESPACE_ON_FAILURE
export ARTIFACTS="${ARTIFACTS:-${ROOT}/_artifacts}"
mkdir -p "${ARTIFACTS}"

E2E_EXTRA=()
E2E_EXTRA+=(--ginkgo.focus="${GINKGO_FOCUS}")
[ -n "$GINKGO_SKIP" ] && E2E_EXTRA+=(--ginkgo.skip="${GINKGO_SKIP}")
[ -n "$E2E_EXTRA_ARGS" ] && E2E_EXTRA+=($E2E_EXTRA_ARGS)

timeout "${TEST_TIMEOUT}" env \
    E2E=y \
    SKIP_BUILD=y \
    SKIP_IMAGE_BUILD=y \
    SKIP_UP=y \
    SKIP_DOWN=y \
    SKIP_IMAGE_LOAD=y \
    KUBECONFIG=/kubeconfig/admin.conf \
    DOCKER_REPO=tidb-operator-e2e \
    IMAGE_TAG="${IMAGE_TAG}" \
    KIND_DATA_HOSTPATH=/kind-data \
    KIND_ETCD_DATADIR=/mnt/tmpfs/etcd \
    DELETE_NAMESPACE_ON_FAILURE="${DELETE_NAMESPACE_ON_FAILURE}" \
    GINKGO_NO_COLOR=y \
    GINKGO_NODES="${GINKGO_NODES}" \
    ./hack/e2e.sh -- "${E2E_EXTRA[@]}" || {
    EXIT_CODE=$?
    if [ "$EXIT_CODE" -eq 124 ]; then
        echo "ERROR: Test execution timed out after $((TEST_TIMEOUT / 3600)) hours"
    else
        echo "ERROR: Test execution failed with exit code $EXIT_CODE"
    fi
    exit $EXIT_CODE
}

# Coverage merge and upload
echo "Merging coverage files..."
mkdir -p /tmp/coverage-merge
cp /kind-data/control-plane/coverage/*.cov /tmp/coverage-merge/ 2>/dev/null || true
cp /kind-data/worker1/coverage/*.cov /tmp/coverage-merge/ 2>/dev/null || true
cp /kind-data/worker2/coverage/*.cov /tmp/coverage-merge/ 2>/dev/null || true
cp /kind-data/worker3/coverage/*.cov /tmp/coverage-merge/ 2>/dev/null || true

if [ -n "$(ls -A /tmp/coverage-merge/*.cov 2>/dev/null)" ]; then
    echo "Merging coverage files..."
    ./bin/gocovmerge /tmp/coverage-merge/*.cov > "${ARTIFACTS}/coverage.txt" || true
    echo "Coverage merge completed"
    if [ -f "${ARTIFACTS}/coverage.txt" ] && [ -n "${CODECOV_TOKEN:-}" ]; then
        echo "Uploading coverage to codecov..."
        GIT_COMMIT=$(git rev-parse HEAD)
        curl -fsSL https://codecov.io/bash -o /tmp/codecov.sh
        bash /tmp/codecov.sh -t "${CODECOV_TOKEN}" -F e2e -n tidb-operator -f "${ARTIFACTS}/coverage.txt" -C "${GIT_COMMIT}" || true
    else
        echo "Skipping coverage upload (file not found or token not set)"
    fi
else
    echo "No coverage files found, skipping merge and upload"
fi

echo "Test execution completed."
