#!/usr/bin/env bash

# Copyright 2026 PingCAP, Inc.
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

# Path constants (override via env if needed)
KIND_DATA_HOSTPATH="${KIND_DATA_HOSTPATH:-/kind-data}"
KIND_ETCD_DATADIR="${KIND_ETCD_DATADIR:-/mnt/tmpfs/etcd}"
KUBECONFIG_PATH="${KUBECONFIG_PATH:-/kubeconfig/admin.conf}"
KUBECONFIG_SOURCE="${KUBECONFIG_SOURCE:-/root/.kube/config}"
KUBECONFIG_DIR="${KUBECONFIG_DIR:-/kubeconfig}"
KIND_NODE_COVERAGE_DIR="${KIND_NODE_COVERAGE_DIR:-/mnt/disks/coverage}"
COVERAGE_MERGE_DIR="${COVERAGE_MERGE_DIR:-/tmp/coverage-merge}"
ARTIFACTS_DIR_DEFAULT="${ARTIFACTS_DIR_DEFAULT:-_artifacts}"

# Environment variables set by the caller (e.g. CI or a wrapper script):
#   CODECOV_TOKEN    - If set, coverage is uploaded to Codecov after tests (see upload_codecov).
#                      Typically defined in the CI/wrapper script that invokes this script.

ROOT=$(unset CDPATH && cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)

function usage() {
    cat <<'EOF'
Usage: hack/run-e2e-kind.sh [options]

Runs tidb-operator e2e tests in a Kind cluster (CI or local).

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
  ./hack/run-e2e-kind.sh --focus='TiDBCluster' --skip='\[TiDBCluster:\sBasic\]' --nodes=4
  ./hack/run-e2e-kind.sh --focus='DMCluster' --nodes=2
  ./hack/run-e2e-kind.sh --focus='\[Serial\]' --extra-args='--install-operator=false' --delete-namespace
EOF
}

function parse_args() {
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
}

# Install Go, Docker CLI, build tools; wait for Docker daemon.
function install_dependencies() {
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
    local timeout_seconds=120
    local start_time
    start_time=$(date +%s)
    while ! docker info >/dev/null 2>&1; do
        local current_time elapsed
        current_time=$(date +%s)
        elapsed=$((current_time - start_time))
        if [ "$elapsed" -ge "$timeout_seconds" ]; then
            echo "Error: Docker daemon did not start within ${timeout_seconds} seconds." >&2
            exit 1
        fi
        sleep 5
    done
    echo "Docker daemon is responsive."
}

# Build binaries, e2e image, and build-side helpers (gocovmerge, codecov patch).
function build_and_prepare() {
    local githash image_tag
    ./hack/e2e-patch-codecov.sh >&2 || true
    unset GOSUMDB
    E2E=y make build e2e-build >&2
    make gocovmerge >&2 || true

    githash=$(git rev-parse HEAD)
    image_tag="e2e-${githash:0:6}"
    echo "Building images locally..." >&2
    E2E=y DOCKER_REPO=tidb-operator-e2e IMAGE_TAG="${image_tag}" make docker e2e-docker >&2 || {
        echo "ERROR: Failed to build Docker images (make docker e2e-docker)" >&2
        exit 1
    }
    echo "${image_tag}"
}


# Create Kind cluster (e2e.sh up), kubeconfig, coverage dirs, load images, tune node resources.
function setup_kind_cluster() {
    local image_tag="$1"
    if [ -z "${image_tag}" ]; then
        echo "ERROR: setup_kind_cluster requires non-empty image_tag" >&2
        exit 1
    fi
    local kind_bin="kind"

    mount --make-rshared / || true
    mkdir -p "${KIND_DATA_HOSTPATH}/control-plane/coverage"
    mkdir -p "${KIND_DATA_HOSTPATH}/worker1/coverage" "${KIND_DATA_HOSTPATH}/worker2/coverage" "${KIND_DATA_HOSTPATH}/worker3/coverage"
    mkdir -p "${KIND_ETCD_DATADIR}"

    PROVIDER=kind KIND_DATA_HOSTPATH="${KIND_DATA_HOSTPATH}" KIND_ETCD_DATADIR="${KIND_ETCD_DATADIR}" \
        SKIP_BUILD=y SKIP_IMAGE_BUILD=y SKIP_TEST=y SKIP_DOWN=y ./hack/e2e.sh

    test -f "${KUBECONFIG_SOURCE}" || { echo "ERROR: ${KUBECONFIG_SOURCE} not found"; exit 1; }
    install -d -m 0755 "${KUBECONFIG_DIR}"
    install -m 0644 "${KUBECONFIG_SOURCE}" "${KUBECONFIG_PATH}"
    export KUBECONFIG="${KUBECONFIG_PATH}"

    echo "Creating coverage directories in kind nodes..."
    for node in tidb-operator-control-plane tidb-operator-worker tidb-operator-worker2 tidb-operator-worker3; do
        docker exec "${node}" mkdir -p "${KIND_NODE_COVERAGE_DIR}" || true
    done

    if ! command -v kind >/dev/null 2>&1; then
        local kind_version="${KIND_VERSION:-v0.19.0}"
        curl -fsSL "https://kind.sigs.k8s.io/dl/${kind_version}/kind-linux-amd64" -o /usr/local/bin/kind
        chmod +x /usr/local/bin/kind
        kind_bin="/usr/local/bin/kind"
    fi
    echo "Loading images into kind cluster..."
    local img
    for img in $(docker images --format "{{.Repository}}:{{.Tag}}" | grep "tidb-operator-e2e/.*:${image_tag}$"); do
        "${kind_bin}" load docker-image "${img}" --name tidb-operator || { echo "Failed to load ${img} into kind"; exit 1; }
    done

    echo "Updating kind node resources..."
    docker update tidb-operator-control-plane --memory-reservation="${KIND_CP_MEM}" -c "${KIND_CP_CPU}"
    docker update tidb-operator-worker -m "${KIND_WK_MEM}" --memory-swap "${KIND_WK_MEM}" -c "${KIND_WK_CPU}"
    docker update tidb-operator-worker2 -m "${KIND_WK_MEM}" --memory-swap "${KIND_WK_MEM}" -c "${KIND_WK_CPU}"
    docker update tidb-operator-worker3 -m "${KIND_WK_MEM}" --memory-swap "${KIND_WK_MEM}" -c "${KIND_WK_CPU}"
}

# Run e2e tests with timeout and env.
function run_tests() {
    local image_tag="$1"
    export GINKGO_NO_COLOR=y
    export DELETE_NAMESPACE_ON_FAILURE
    export ARTIFACTS="${ARTIFACTS:-${ROOT}/${ARTIFACTS_DIR_DEFAULT}}"
    mkdir -p "${ARTIFACTS}"

    local e2e_extra=()
    e2e_extra+=(--ginkgo.focus="${GINKGO_FOCUS}")
    [ -n "$GINKGO_SKIP" ] && e2e_extra+=(--ginkgo.skip="${GINKGO_SKIP}")
    if [ -n "$E2E_EXTRA_ARGS" ]; then
        set -f
        e2e_extra+=($E2E_EXTRA_ARGS)
        set +f
    fi

    timeout "${TEST_TIMEOUT}" env \
        E2E=y \
        SKIP_BUILD=y \
        SKIP_IMAGE_BUILD=y \
        SKIP_UP=y \
        SKIP_DOWN=y \
        SKIP_IMAGE_LOAD=y \
        KUBECONFIG="${KUBECONFIG_PATH}" \
        DOCKER_REPO=tidb-operator-e2e \
        IMAGE_TAG="${image_tag}" \
        KIND_DATA_HOSTPATH="${KIND_DATA_HOSTPATH}" \
        KIND_ETCD_DATADIR="${KIND_ETCD_DATADIR}" \
        DELETE_NAMESPACE_ON_FAILURE="${DELETE_NAMESPACE_ON_FAILURE}" \
        GINKGO_NO_COLOR=y \
        GINKGO_NODES="${GINKGO_NODES}" \
        ./hack/e2e.sh -- "${e2e_extra[@]}" || {
        local exit_code=$?
        if [ "$exit_code" -eq 124 ]; then
            echo "ERROR: Test execution timed out after $((TEST_TIMEOUT / 3600)) hours"
        else
            echo "ERROR: Test execution failed with exit code $exit_code"
        fi
        exit $exit_code
    }
}

# Merge coverage from kind node paths and upload to Codecov if token set.
function upload_codecov() {
    echo "Merging coverage files..."
    mkdir -p "${COVERAGE_MERGE_DIR}"
    cp "${KIND_DATA_HOSTPATH}/control-plane/coverage/"*.cov "${COVERAGE_MERGE_DIR}/" 2>/dev/null || true
    cp "${KIND_DATA_HOSTPATH}/worker1/coverage/"*.cov "${COVERAGE_MERGE_DIR}/" 2>/dev/null || true
    cp "${KIND_DATA_HOSTPATH}/worker2/coverage/"*.cov "${COVERAGE_MERGE_DIR}/" 2>/dev/null || true
    cp "${KIND_DATA_HOSTPATH}/worker3/coverage/"*.cov "${COVERAGE_MERGE_DIR}/" 2>/dev/null || true

    if [ -n "$(ls -A "${COVERAGE_MERGE_DIR}/"*.cov 2>/dev/null)" ]; then
        ./bin/gocovmerge "${COVERAGE_MERGE_DIR}/"*.cov > "${ARTIFACTS}/coverage.txt" || true
        echo "Coverage merge completed"
        if [ -f "${ARTIFACTS}/coverage.txt" ] && [ -n "${CODECOV_TOKEN:-}" ]; then
            # CODECOV_TOKEN is set by the caller (e.g. Prow or wrapper script)
            echo "Uploading coverage to codecov..."
            local git_commit
            git_commit=$(git rev-parse HEAD)
            curl -fsSL https://codecov.io/bash -o /tmp/codecov.sh
            bash /tmp/codecov.sh -t "${CODECOV_TOKEN}" -F e2e -n tidb-operator -f "${ARTIFACTS}/coverage.txt" -C "${git_commit}" || true
        else
            echo "Skipping coverage upload (file not found or token not set)"
        fi
    else
        echo "No coverage files found, skipping merge and upload"
    fi
}

function main() {
    parse_args "$@"

    cd "$ROOT"
    echo "Prow E2E Kind: focus=$GINKGO_FOCUS skip=$GINKGO_SKIP nodes=$GINKGO_NODES"
    echo "Kind resources: cp mem=$KIND_CP_MEM cpu=$KIND_CP_CPU, worker mem=$KIND_WK_MEM cpu=$KIND_WK_CPU"

    install_dependencies

    local image_tag
    image_tag=$(build_and_prepare)
    if [ -z "${image_tag}" ]; then
        echo "ERROR: build_and_prepare did not return image_tag" >&2
        exit 1
    fi
    setup_kind_cluster "${image_tag}"
    run_tests "${image_tag}"
    upload_codecov

    echo "Test execution completed."
}

main "$@"
