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

ROOT=$(cd $(dirname "${BASH_SOURCE[0]}")/..; pwd -P)

RELEASE_TAG=${V_RELEASE:-"test"}
CHARTS_BUILD_DIR="output/chart"
CHART_ITEMS="tidb-operator tidb-drainer tidb-lightning"
BR_FEDERATION=${BR_FEDERATION:-"true"}

mkdir -p ${CHARTS_BUILD_DIR}

# Build charts
for chartItem in ${CHART_ITEMS}; do
    [ ! -d "charts/${chartItem}" ] && continue
    
    chartPrefixName="${chartItem}-${RELEASE_TAG}"
    
    # Update Chart.yaml
    sed -i.bak "s/^version:.*/version: ${RELEASE_TAG}/g" charts/${chartItem}/Chart.yaml
    sed -i.bak "s/^appVersion:.*/appVersion: ${RELEASE_TAG}/g" charts/${chartItem}/Chart.yaml
    rm -f charts/${chartItem}/Chart.yaml.bak
    
    # Update values.yaml
    sed -i.bak -E "s#pingcap/(tidb-operator|tidb-backup-manager):[^[:space:]]*#pingcap/\\1:${RELEASE_TAG}#g" charts/${chartItem}/values.yaml 2>/dev/null || true
    rm -f charts/${chartItem}/values.yaml.bak 2>/dev/null || true
    
    # Package chart
    tar -zcf ${CHARTS_BUILD_DIR}/${chartPrefixName}.tgz -C charts ${chartItem}
    sha256sum ${CHARTS_BUILD_DIR}/${chartPrefixName}.tgz > ${CHARTS_BUILD_DIR}/${chartPrefixName}.sha256
done

# Handle br-federation
if [[ "$BR_FEDERATION" == "true" ]] && [ -d "charts/br-federation" ]; then
    chartItem="br-federation"
    chartPrefixName="${chartItem}-${RELEASE_TAG}"
    sed -i.bak "s/^version:.*/version: ${RELEASE_TAG}/g" charts/${chartItem}/Chart.yaml
    sed -i.bak "s/^appVersion:.*/appVersion: ${RELEASE_TAG}/g" charts/${chartItem}/Chart.yaml
    rm -f charts/${chartItem}/Chart.yaml.bak
    sed -i.bak -E "s#pingcap/br-federation-manager:[^[:space:]]*#pingcap/br-federation-manager:${RELEASE_TAG}#g" charts/${chartItem}/values.yaml
    rm -f charts/${chartItem}/values.yaml.bak
    tar -zcf ${CHARTS_BUILD_DIR}/${chartPrefixName}.tgz -C charts ${chartItem}
    sha256sum ${CHARTS_BUILD_DIR}/${chartPrefixName}.tgz > ${CHARTS_BUILD_DIR}/${chartPrefixName}.sha256
fi