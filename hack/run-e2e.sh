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

hack::ensure_kubectl
hack::ensure_helm

TIDB_OPERATOR_IMAGE=${TIDB_OPERATOR_IMAGE:-localhost:5000/pingcap/tidb-operator:latest}
E2E_IMAGE=${E2E_IMAGE:-localhost:5000/pingcap/tidb-operator-e2e:latest}
KUBECONFIG=${KUBECONFIG:-$HOME/.kube/config}
KUBECONTEXT=${KUBECONTEXT:-}
REPORT_DIR=${REPORT_DIR:-}
REPORT_PREFIX=${REPORT_PREFIX:-}

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

GINKGO_PARALLEL=${GINKGO_PARALLEL:-n} # set to 'y' to run tests in parallel
# If 'y', Ginkgo's reporter will not print out in color when tests are run
# in parallel
GINKGO_NO_COLOR=${GINKGO_NO_COLOR:-n}
GINKGO_STREAM=${GINKGO_STREAM:-y}

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

echo "info: start to run e2e process"
e2e_args=(
    /usr/local/bin/ginkgo
    ${ginkgo_args[@]:-}
    /usr/local/bin/e2e.test
    --
    --provider=skeleton
    --clean-start=true
    --delete-namespace-on-failure=false
    # tidb-operator e2e flags
    --operator-tag=e2e
    --operator-image=${TIDB_OPERATOR_IMAGE}
    --e2e-image=${E2E_IMAGE}
    --tidb-versions=v3.0.2,v3.0.3,v3.0.4,v3.0.5
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
    -v $ROOT:$ROOT
    -w $ROOT
    -v $KUBECONFIG:/etc/kubernetes/admin.conf:ro
    --env KUBECONFIG=/etc/kubernetes/admin.conf
    --env KUBECONTEXT=$KUBECONTEXT
)

if [ -n "$REPORT_DIR" ]; then
    docker_args+=(
        -v $REPORT_DIR:$REPORT_DIR
    )
fi

echo "info: docker ${docker_args[@]} $E2E_IMAGE ${e2e_args[@]}"
docker ${docker_args[@]} $E2E_IMAGE ${e2e_args[@]}
