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

ROOT=$(
    cd $(dirname "${BASH_SOURCE[0]}")/../..
    pwd -P
)

source $ROOT/hack/lib/vars.sh
source $ROOT/hack/lib/kind.sh
source $ROOT/hack/lib/image.sh

OUTPUT_DIR=$ROOT/_output
KUBECTL=$OUTPUT_DIR/bin/kubectl
HELM=$OUTPUT_DIR/bin/helm
GINKGO=$OUTPUT_DIR/bin/ginkgo
GENERATEJWT=$OUTPUT_DIR/bin/generate_jwt

CI=${CI:-""}
OLD_VERSION_BRANCH=${OLD_VERSION_BRANCH:-"feature/v2"}
# Comma-separated list of packages to exclude from e2e tests
E2E_EXCLUDED_PACKAGES=${E2E_EXCLUDED_PACKAGES:-"upgrade"}

function e2e::ensure_kubectl() {
    if ! command -v $KUBECTL &>/dev/null; then
        echo "kubectl not found, installing..."
        curl -L -o $KUBECTL https://dl.k8s.io/release/v1.30.2/bin/${V_OS}/${V_ARCH}/kubectl
        chmod +x $KUBECTL
    fi
}

function e2e::switch_kube_context() {
    echo "switching to kind context ${V_KIND_CLUSTER}"
    $KUBECTL config use-context kind-${V_KIND_CLUSTER}
}

function e2e::ensure_cert_manager() {
    echo "installing cert-manager..."
    $HELM upgrade \
        cert-manager oci://quay.io/jetstack/charts/cert-manager \
        -i \
        --wait \
        --version v1.18.2 \
        --namespace cert-manager \
        --create-namespace \
        --set crds.enabled=true
}

function e2e::ensure_trust_manager() {
    echo "installing trust-manager..."
    $HELM upgrade \
        trust-manager oci://quay.io/jetstack/charts/trust-manager \
        -i \
        --wait \
        --version v0.18.0 \
        --namespace cert-manager \
        --create-namespace \
        --set secretTargets.enabled=true \
        --set secretTargets.authorizedSecretsAll=true
}

function e2e::delete_crds() {
    echo "deleting operator crds and dependent CRs"

    # Safety check: ensure this only runs on kind clusters
    local current_context
    current_context=$($KUBECTL config current-context)
    if [[ ! "$current_context" =~ ^kind- ]]; then
        echo "!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!" >&2
        echo "!!! DANGER: Current kubectl context is NOT a kind cluster.  !!!" >&2
        echo "!!! Context: ${current_context}                               !!!" >&2
        echo "!!! Aborting CRD deletion to prevent accidental data loss.    !!!" >&2
        echo "!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!" >&2
        exit 1
    fi

    # Get all CRD names managed by the operator
    local crd_names
    crd_names=$($KUBECTL get crd -o name | grep "pingcap.com" | sed 's|customresourcedefinition.apiextensions.k8s.io/||')
    if [[ -z "$crd_names" ]]; then
        echo "No PingCAP CRDs found to delete."
        return
    fi

    # For each CRD, delete all its custom resources (CRs)
    for crd in $crd_names; do
        local resource
        resource=$(echo "$crd" | cut -d. -f1)
        echo "Deleting all objects of type: $resource"
        if ! $KUBECTL delete "$resource" --all --all-namespaces --wait=false >/dev/null 2>&1; then
            echo "No resources of type $resource found, or failed to initiate deletion."
        fi
    done

    # Now, wait for all CRs to be fully terminated
    echo "Waiting for all CRs to be deleted..."
    for crd in $crd_names; do
        local resource
        resource=$(echo "$crd" | cut -d. -f1)
        for i in {1..60}; do # Wait for up to 2 minutes
            local count
            count=$($KUBECTL get "$resource" --all-namespaces -o name --ignore-not-found=true 2>/dev/null | wc -l | tr -d ' ')
            if [[ "$count" -eq 0 ]]; then
                echo "All resources of type $resource are deleted."
                break
            fi
            echo "Waiting for $count resources of type $resource to be deleted..."
            sleep 2
            if [[ $i -eq 60 ]]; then
                echo "Timeout waiting for $resource to be deleted. Forcing removal of finalizers..."
                local crs_to_patch
                crs_to_patch=$($KUBECTL get "$resource" --all-namespaces -o jsonpath='{range .items[*]}{.metadata.namespace}{" "}{.metadata.name}{"\n"}{end}' 2>/dev/null)
                if [[ -n "$crs_to_patch" ]]; then
                    echo "$crs_to_patch" | while read -r ns name; do
                        echo "Removing finalizers from $resource/$name in namespace $ns"
                        $KUBECTL patch "$resource" "$name" -n "$ns" --type=merge -p '{"metadata":{"finalizers":[]}}' > /dev/null
                    done
                fi
                sleep 5 # Give it a moment after patching
            fi
        done
    done

    # Finally, delete the CRDs themselves
    echo "Deleting operator CRDs..."
    if ! echo "$crd_names" | xargs -r -n 1 -I {} $KUBECTL delete crd {}; then
        echo "Failed to delete some CRDs, but continuing..."
    fi

    # Wait for CRDs to be deleted
    echo "Waiting for all CRDs to be deleted..."
    for crd in $crd_names; do
        for i in {1..30}; do
            if ! $KUBECTL get crd "$crd" >/dev/null 2>&1; then
                echo "CRD $crd has been deleted."
                break
            fi
            sleep 1
             if [[ $i -eq 30 ]]; then
                echo "Timeout waiting for CRD $crd to be deleted."
            fi
        done
    done
}

function e2e::ensure_old_version_repo() {
    local sanitized_branch_name
    sanitized_branch_name=$(echo "${OLD_VERSION_BRANCH}" | tr '/' '-')
    local old_version_dir="${OUTPUT_DIR}/old-version-repo/${sanitized_branch_name}"

    if [[ -d "${old_version_dir}/.git" ]]; then
        echo "Old version repo already exists in ${old_version_dir}, updating..." >&2
        if ! (cd "${old_version_dir}" && git fetch origin && git reset --hard "origin/${OLD_VERSION_BRANCH}" >/dev/null 2>&1); then
            echo "Failed to update old version repo, re-cloning..." >&2
            rm -rf "${old_version_dir}"
        else
            (cd "${old_version_dir}" && git checkout "${OLD_VERSION_BRANCH}" >/dev/null 2>&1 && git pull >/dev/null 2>&1)
        fi
    fi

    if [[ ! -d "${old_version_dir}/.git" ]]; then
        echo "Cloning old version from branch ${OLD_VERSION_BRANCH} into ${old_version_dir}" >&2
        rm -rf "${old_version_dir}"
        local repo_url="https://github.com/pingcap/tidb-operator.git"
        if ! git clone --branch "${OLD_VERSION_BRANCH}" --depth 1 "${repo_url}" "${old_version_dir}"; then
            echo "Failed to clone branch ${OLD_VERSION_BRANCH}" >&2
            exit 1
        fi
    fi
    echo "${old_version_dir}"
}

function e2e::install_old_version() {
    echo "Preparing old version from branch ${OLD_VERSION_BRANCH}"

    # Ensure the old version repository is checked-out locally
    local old_version_dir
    old_version_dir=$(e2e::ensure_old_version_repo)

    pushd $old_version_dir

    # Build the operator image from the old version source
    echo "Building old version operator image from ${old_version_dir}"
    local commit_hash
    commit_hash=$(git rev-parse --short HEAD)

    # TODO(liubo02): del it, reuse kind
    make bin/kind
    make V_KIND=${V_KIND} push/prestop-checker

    local old_operator_image="${V_IMG_PROJECT}/tidb-operator:old-${commit_hash}"

    # Prepare build/cache directories (re-use paths from hack/lib/image.sh)
    mkdir -p ${IMAGE_DIR}
    mkdir -p ${CACHE_DIR}
    local image_tar="${IMAGE_DIR}/tidb-operator-${commit_hash}.tar"

    # If the OCI archive already exists, reuse it to skip rebuild
    if [[ -f "${image_tar}" ]]; then
        echo "Found cached image archive ${image_tar}, skipping build."
    else
        # Check if the image with this tag already exists locally
        if docker image inspect "${old_operator_image}" > /dev/null 2>&1; then
            echo "Image ${old_operator_image} already exists locally, exporting to archive..."
            docker save -o "${image_tar}" "${old_operator_image}"
        else
            echo "Building image ${old_operator_image}"
            # Honour platform settings if provided
            local build_args=""
            if [[ -n "${V_PLATFORMS}" ]]; then
                build_args="--platform ${V_PLATFORMS}"
            fi

            # Ensure a usable buildx builder
            if ! docker buildx ls | grep "*" | grep -q "docker-container"; then
                echo "'docker-container' builder not found, creating it..."
                docker buildx create --use
            fi

            docker buildx build \
                --target tidb-operator \
                -o type=oci,dest="${image_tar}" \
                -t "${old_operator_image}" \
                --cache-from=type=local,src=${CACHE_DIR} \
                --cache-to=type=local,dest=${CACHE_DIR} \
                ${build_args} \
                -f "${old_version_dir}/image/Dockerfile" "${old_version_dir}"
        fi
    fi

    echo "Loading old operator image into kind cluster"
    $V_KIND load image-archive "${image_tar}" --name "${V_KIND_CLUSTER}"

    # Apply CRDs from the old version
    echo "Deploying old version CRDs"
    e2e::delete_crds
    local old_crd_path="${old_version_dir}/manifests/crd"
    if ! $KUBECTL apply --server-side=true -f "${old_crd_path}"; then
        echo "Failed to apply old CRDs"
        exit 1
    fi

    # Patch the operator deployment to use the just-built image
    echo "Updating operator deployment to old version image"
    if ! $KUBECTL -n $V_DEPLOY_NAMESPACE set image deployment/tidb-operator "tidb-operator=${old_operator_image}"; then
        echo "Failed to set old operator image"
        exit 1
    fi

    echo "Waiting for old operator to be ready"
    if ! $KUBECTL -n $V_DEPLOY_NAMESPACE wait --for=condition=Available --timeout=5m deployment/tidb-operator; then
        echo "Timed out waiting for old operator to be ready"
        exit 1
    fi
}

function e2e::install_ginkgo() {
    if ! command -v $GINKGO &>/dev/null; then
        echo "ginkgo not found, installing..."
        make bin/ginkgo
    fi
}

# generate_jwt is a tool to generate JWT token for `tidb_auth_token` test
# ref: https://docs.pingcap.com/tidb/stable/security-compatibility-with-mysql#tidb_auth_token
function e2e::install_generate_jwt() {
    if ! command -v $GENERATEJWT &>/dev/null; then
        echo "generate_jwt not found, installing..."
        $ROOT/hack/download.sh go_install $GENERATEJWT github.com/cbcwestwolf/generate_jwt@latest
    fi
}

function e2e::run() {
    # Dynamically discover test packages, excluding specified packages to avoid compilation failures
    IFS=',' read -ra excluded_packages <<< "${E2E_EXCLUDED_PACKAGES}"
    local test_packages=()
    local excluded_packages_found=()

    # Find all directories containing *_test.go files
    while IFS= read -r -d '' test_dir; do
        local relative_path="${test_dir#$ROOT/tests/e2e/}"
        local should_exclude=false

        # Check if this directory should be excluded
        for excluded in "${excluded_packages[@]}"; do
            if [[ "$relative_path" == "$excluded" || "$relative_path" == "$excluded/"* ]]; then
                should_exclude=true
                excluded_packages_found+=("$relative_path")
                break
            fi
        done

        if [[ "$should_exclude" == false ]]; then
            test_packages+=("$test_dir")
        fi
    done < <(find "$ROOT/tests/e2e" -name "*_test.go" -exec dirname {} \; | sort -u | tr '\n' '\0')

    if [[ ${#excluded_packages_found[@]} -gt 0 ]]; then
        echo "Excluded packages: ${excluded_packages_found[*]}"
    fi

    if [[ ${#test_packages[@]} -eq 0 ]]; then
        echo "No test packages found!"
        exit 1
    fi

    echo "Running tests in packages: ${test_packages[*]}"

    # Use individual package paths instead of recursive mode to avoid scanning excluded packages
    if [[ "$CI" == "true" ]]; then
        echo "running e2e tests in CI mode with options: $*"
        $GINKGO -v --timeout=2h --randomize-all --randomize-suites --fail-on-empty --keep-going --trace "$*" "${test_packages[@]}"
    else
        echo "running e2e tests locally..."
        $GINKGO -v --race "$@" "${test_packages[@]}"
    fi
}

function e2e::run_upgrade() {
    e2e::install_old_version
    $GINKGO -v -r --tags=upgrade_e2e --timeout=1h --randomize-all --randomize-suites --fail-on-empty --race --trace --flake-attempts=2 "$ROOT/tests/e2e/upgrade"
}

function e2e::prepare() {
    e2e::install_ginkgo
    e2e::install_generate_jwt
    e2e::ensure_kubectl
    kind::ensure_cluster
    e2e::switch_kube_context
    e2e::ensure_cert_manager
    e2e::ensure_trust_manager

    # build the operator image and load it into the kind cluster
    image::build prestop-checker tidb-operator testing-workload tidb-backup-manager --push

    image:prepare

    # TODO(liubo02): use a lib script
    make KUBECTL=${KUBECTL} e2e/deploy
}

function e2e::e2e() {
    local ginkgo_opts=()
    local prepare=0
    local run=0
    local run_upgrade=0

    while [[ $# -gt 0 ]]; do
        case $1 in
        --prepare)
            prepare=1
            shift
            ;;
        run)
            run=1
            shift
            ;;
        run-upgrade)
            run_upgrade=1
            shift
            ;;
        *)
            if [[ $run -eq 1 || $run_upgrade -eq 1 ]]; then
                ginkgo_opts+=("${1}")
            else
                echo "Unknown option $1"
                exit 1
            fi
            shift
            ;;
        esac
    done

    if [[ $prepare -eq 1 ]]; then
        e2e::prepare
    fi

    if [[ $run -eq 1 ]]; then
        e2e::run "${ginkgo_opts[@]}"
    fi
    if [[ $run_upgrade -eq 1 ]]; then
        e2e::run_upgrade "${ginkgo_opts[@]}"
    fi
}
