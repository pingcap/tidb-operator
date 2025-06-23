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
KIND=$OUTPUT_DIR/bin/kind
KUBECTL=$OUTPUT_DIR/bin/kubectl
GINKGO=$OUTPUT_DIR/bin/ginkgo
GENERATEJWT=$OUTPUT_DIR/bin/generate_jwt

CI=${CI:-""}
OLD_VERSION_BRANCH=${OLD_VERSION_BRANCH:-"feature/v2"}
OLD_OPERATOR_IMAGE_TAG=${OLD_OPERATOR_IMAGE_TAG:-""}

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
    echo "checking if cert-manager is installed..."
    if $KUBECTL -n cert-manager get deployment cert-manager &>/dev/null; then
        echo "cert-manager already installed, skipping..."
        return
    fi

    echo "installing cert-manager..."
    $KUBECTL apply --server-side=true -f https://github.com/cert-manager/cert-manager/releases/download/v1.15.2/cert-manager.yaml

    echo "waiting for cert-manager to be ready..."
    $KUBECTL -n cert-manager wait --for=condition=Available --timeout=5m deployment/cert-manager
}

function e2e::install_crds() {
    echo "installing CRDs..."
    $KUBECTL apply --server-side=true -f $ROOT/manifests/crd
}

function e2e::install_operator() {
    echo "installing operator..."
    $KUBECTL -n $V_DEPLOY_NAMESPACE apply --server-side=true -f $OUTPUT_DIR/manifests/tidb-operator.yaml

    echo "waiting for operator to be ready..."
    $KUBECTL -n $V_DEPLOY_NAMESPACE wait --for=condition=Available --timeout=5m deployment/tidb-operator
}

function e2e::uninstall_operator() {
    echo "checking if operator is installed..."
    if ! $KUBECTL -n $V_DEPLOY_NAMESPACE get deployment tidb-operator &>/dev/null; then
        echo "operator not found, skipping uninstall..."
        return
    fi

    echo "uninstalling operator..."
    $KUBECTL -n $V_DEPLOY_NAMESPACE delete -f $OUTPUT_DIR/manifests/tidb-operator.yaml

    echo "waiting for operator to be deleted..."
    $KUBECTL -n $V_DEPLOY_NAMESPACE wait --for=delete --timeout=5m deployment/tidb-operator
}

function e2e::delete_crds() {
    echo "deleting operator crds"
    if ! $KUBECTL get crd -o name | grep "pingcap.com" | xargs -r $KUBECTL delete; then
        echo "No CRDs found to delete, continuing..."
    fi
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

function e2e::get_old_operator_image_tag() {
    local old_version_dir
    old_version_dir=$(e2e::ensure_old_version_repo)

    local commit_hash
    commit_hash=$(cd "${old_version_dir}" && git rev-parse HEAD)
    local short_commit_hash
    short_commit_hash=${commit_hash:0:7}

    echo "Fetching tags for commit ${short_commit_hash} from gcr.io/pingcap-public/dbaas/tidb-operator..." >&2

    local image_tag
    # Using curl to avoid dependency on gcloud and authentication.
    # We filter for tags that contain the short commit hash but do not contain an underscore (to exclude arch-specific tags).
    image_tag=$(curl -s "https://gcr.io/v2/pingcap-public/dbaas/tidb-operator/tags/list" |
        grep -o '"[^"]*g'${short_commit_hash}'[^"]*"' |
        grep -v '_' |
        sed 's/"//g' |
        head -n 1)

    if [[ -z "$image_tag" ]]; then
        echo "Could not find a suitable image tag for commit ${short_commit_hash}" >&2
        echo "Debugging info: Listing all tags containing the commit hash..." >&2
        curl -s "https://gcr.io/v2/pingcap-public/dbaas/tidb-operator/tags/list" |
            grep -o '"[^"]*g'${short_commit_hash}'[^"]*"' |
            sed 's/"//g' |
            head -n 10 >&2
        exit 1
    fi

    echo "$image_tag"
}

function e2e::install_old_version() {
    if [[ -z "$OLD_OPERATOR_IMAGE_TAG" ]]; then
        echo "OLD_OPERATOR_IMAGE_TAG is not set, trying to determine it automatically."
        OLD_OPERATOR_IMAGE_TAG=$(e2e::get_old_operator_image_tag)
        echo "Determined OLD_OPERATOR_IMAGE_TAG: ${OLD_OPERATOR_IMAGE_TAG}"
    fi
    local old_operator_image="gcr.io/pingcap-public/dbaas/tidb-operator:${OLD_OPERATOR_IMAGE_TAG}"

    echo "Preparing manifests for old version from branch ${OLD_VERSION_BRANCH}"
    local old_version_dir
    old_version_dir=$(e2e::ensure_old_version_repo)

    echo "Deploying old version of CRDs"
    e2e::delete_crds
    local old_crd_path="${old_version_dir}/manifests/crd"
    if ! $KUBECTL apply --server-side=true -f "${old_crd_path}"; then
        echo "Failed to apply old CRDs"
        exit 1
    fi

    echo "Deploying old version of operator by setting image"
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

function e2e::reload_testing_workload() {
    image::build testing-workload --push
}

function e2e::install_ginkgo() {
    if ! command -v $GINKGO &>/dev/null; then
        echo "ginkgo not found, installing..."
        $ROOT/hack/download.sh go_install $GINKGO github.com/onsi/ginkgo/v2/ginkgo
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
    local skip_packages="github.com/pingcap/tidb-operator/tests/e2e/upgrade"
    if [[ "$CI" == "true" ]]; then
        echo "running e2e tests in CI mode with options: $*"
        $GINKGO -v -r --skip-package="$skip_packages" --timeout=2h --randomize-all --randomize-suites --fail-on-empty --keep-going --race --trace --flake-attempts=2 "$*" "$ROOT/tests/e2e/..."
        
        # To avoid the case affect other cases, run upgrade e2e separately.
        echo "running upgrade e2e tests"
        e2e::run_upgrade
    else
        echo "running e2e tests locally..."
        $GINKGO -r -v --skip-package="$skip_packages" "$@" "$ROOT/tests/e2e/..."
    fi
}

function e2e::run_upgrade() {
    e2e::install_old_version
    $GINKGO -v -r --tags=upgrade_e2e --timeout=1h --randomize-all --randomize-suites --fail-on-empty --race --trace "$ROOT/tests/e2e/upgrade"
}

function e2e::prepare() {
    if [[ -z "$OLD_OPERATOR_IMAGE_TAG" ]]; then
        echo "OLD_OPERATOR_IMAGE_TAG is not set, trying to determine it automatically."
        OLD_OPERATOR_IMAGE_TAG=$(e2e::get_old_operator_image_tag)
        echo "Determined OLD_OPERATOR_IMAGE_TAG: ${OLD_OPERATOR_IMAGE_TAG}"
    fi
    e2e::install_ginkgo
    e2e::install_generate_jwt
    e2e::ensure_kubectl
    kind::ensure_cluster
    e2e::switch_kube_context
    e2e::ensure_cert_manager

    e2e::install_crds

    # build the operator image and load it into the kind cluster
    image::build prestop-checker tidb-operator testing-workload tidb-backup-manager --push
    e2e::uninstall_operator
    e2e::install_operator

    image:prepare
}

function e2e::reinstall_operator() {
    image::build tidb-operator --push
    e2e::uninstall_operator
    e2e::install_operator
}

function e2e::reinstall_backup_manager() {
    image::build tidb-backup-manager --push
}

function e2e::e2e() {
    local ginkgo_opts=()
    local prepare=0
    local run=0
    local run_upgrade=0
    local reinstall_operator=0
    local reinstall_backup_manager=0
    local reload_testing_workload=0

    while [[ $# -gt 0 ]]; do
        case $1 in
        --prepare)
            prepare=1
            shift
            ;;
        --reinstall-operator)
            reinstall_operator=1
            shift
            ;;
        --reinstall-backup-manager)
            reinstall_backup_manager=1
            shift
            ;;
        --reload-testing-workload)
            reload_testing_workload=1
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
    elif [[ $reinstall_operator -eq 1 ]]; then
        e2e::reinstall_operator
    elif [[ $reinstall_backup_manager -eq 1 ]]; then
        e2e::reinstall_backup_manager
    elif [[ $reload_testing_workload -eq 1 ]]; then
        e2e::reload_testing_workload
    fi

    if [[ $run -eq 1 ]]; then
        e2e::run "${ginkgo_opts[@]}"
    elif [[ $run_upgrade -eq 1 ]]; then
        e2e::run_upgrade "${ginkgo_opts[@]}"
    fi
}
