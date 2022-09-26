#!/usr/bin/env bash

# Copyright 2021 PingCAP, Inc.
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
SKIP_CRD_FILES=(
    "pingcap.com_dataresources.yaml"
)

cd ${ROOT}
source hack/lib.sh

CONTROLLER_GEN=${OUTPUT_BIN}/controller-gen
hack::ensure_controller_gen

API_PACKAGES="github.com/pingcap/tidb-operator/pkg/apis/..."
CRD_OUTPUT_DIR=${ROOT}/manifests/crd
CRD_OPTIONS="preserveUnknownFields=false,allowDangerousTypes=true,maxDescLen=0"

# generate CRDs
${CONTROLLER_GEN} \
    crd:crdVersions=v1beta1,${CRD_OPTIONS} \
    paths=${API_PACKAGES} \
    output:crd:dir=${CRD_OUTPUT_DIR}/v1beta1
${CONTROLLER_GEN} \
    crd:crdVersions=v1,${CRD_OPTIONS} \
    paths=${API_PACKAGES} \
    output:crd:dir=${CRD_OUTPUT_DIR}/v1

for file in ${SKIP_CRD_FILES[@]}; do
    rm -f ${CRD_OUTPUT_DIR}/v1beta1/${file}
    rm -f ${CRD_OUTPUT_DIR}/v1/${file}
done

# merge all CRDs
cat ${CRD_OUTPUT_DIR}/v1/*.yaml > ${ROOT}/manifests/crd.yaml
cat ${CRD_OUTPUT_DIR}/v1beta1/*.yaml > ${ROOT}/manifests/crd_v1beta1.yaml
