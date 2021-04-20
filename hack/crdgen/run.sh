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

ROOT=$(unset CDPATH && cd $(dirname "${BASH_SOURCE[0]}")/../.. && pwd)

SCRIPT_DIR=${ROOT}/hack/crdgen
OUTPUT_DIR=${ROOT}/output
CONTROLLER_GEN=${OUTPUT_DIR}/bin/controller-gen
API_PACKAGES=${ROOT}/pkg/apis/...
CRD_OUTPUT_DIR=${ROOT}/manifests/crd
SKIP_CRD_FILES="pingcap.com_dataresources.yaml"

# build controller-gen
cd $SCRIPT_DIR
CONTROLLER_GEN_DIR=${CONTROLLER_GEN_DIR:-$(go list -f "{{ .Dir }}" sigs.k8s.io/controller-tools/cmd/controller-gen)}
go build -v -o ${CONTROLLER_GEN} ${CONTROLLER_GEN_DIR}


cd $ROOT
# run controller-gen to generate v1beta1 crd files
${CONTROLLER_GEN} \
    crd:crdVersions=v1beta1,preserveUnknownFields=false,allowDangerousTypes=true \
    paths=${API_PACKAGES} \
    output:crd:dir=${CRD_OUTPUT_DIR}/v1beta1

${CONTROLLER_GEN} \
    crd:crdVersions=v1,preserveUnknownFields=false,allowDangerousTypes=true \
    paths=${API_PACKAGES} \
    output:crd:dir=${CRD_OUTPUT_DIR}/v1

for file in $SKIP_CRD_FILES; do
    rm $CRD_OUTPUT_DIR/v1beta1/$file
    rm $CRD_OUTPUT_DIR/v1/$file
done
