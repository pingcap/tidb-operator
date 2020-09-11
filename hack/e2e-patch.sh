#!/bin/bash

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

# This script patches the chart templates of tidb-operator to use
# in E2E tests.

set -e

echo "hack/e2e-patch.sh: PWD $PWD"

CONTROLLER_MANAGER_DEPLOYMENT=charts/tidb-operator/templates/controller-manager-deployment.yaml
SCHEDULER_DEPLOYMENT=charts/tidb-operator/templates/scheduler-deployment.yaml
DISCOVERY_DEPLOYMENT=charts/tidb-cluster/templates/discovery-deployment.yaml

# enable coverage profile
sed -i 's/\/usr\/local\/bin\/tidb-controller-manager/\/e2e-entrypoint.sh\n          - \/usr\/local\/bin\/tidb-controller-manager\n          - -test.coverprofile=\/tmp\/e2e.coverage.txt\n          - --/g' \
    $CONTROLLER_MANAGER_DEPLOYMENT
sed -i 's/\/usr\/local\/bin\/tidb-scheduler/\/e2e-entrypoint.sh\n          - \/usr\/local\/bin\/tidb-scheduler\n          - -test.coverprofile=\/tmp\/e2e.coverage.txt\n          - --/g' \
    $SCHEDULER_DEPLOYMENT
sed -i 's/\/usr\/local\/bin\/tidb-discovery/\/e2e-entrypoint.sh\n          - \/usr\/local\/bin\/tidb-discovery\n          - -test.coverprofile=\/tmp\/e2e.coverage.txt\n          - --/g' \
    $DISCOVERY_DEPLOYMENT

# -v is duplicated for operator and go test
sed -i '/\-v=/d' $CONTROLLER_MANAGER_DEPLOYMENT
sed -i '/\-v=/d' $SCHEDULER_DEPLOYMENT
echo "hack/e2e-patch.sh: enable coverage profile"

# populate needed environment variables
cat << EOF >> $CONTROLLER_MANAGER_DEPLOYMENT
          - name: COMPONENT
            value: "controller-manager"
          - name: BRANCH
            value: "$SRC_BRANCH"
          - name: BUILD
            value: "$BUILD_NUMBER"
          - name: COMMIT
            value: "$GIT_COMMIT"
          - name: PR
            value: "$PR_ID"
          - name: CODECOV_TOKEN
            value: "$CODECOV_TOKEN"
EOF

cat << EOF >> $SCHEDULER_DEPLOYMENT
        - name: COMPONENT
          value: "scheduler"
        - name: BRANCH
          value: "$SRC_BRANCH"
        - name: BUILD
          value: "$BUILD_NUMBER"
        - name: COMMIT
          value: "$GIT_COMMIT"
        - name: PR
          value: "$PR_ID"
        - name: CODECOV_TOKEN
          value: "$CODECOV_TOKEN"
EOF

cat << EOF >> $DISCOVERY_DEPLOYMENT
          - name: COMPONENT
            value: "discovery"
          - name: BRANCH
            value: "$SRC_BRANCH"
          - name: BUILD
            value: "$BUILD_NUMBER"
          - name: COMMIT
            value: "$GIT_COMMIT"
          - name: PR
            value: "$PR_ID"
          - name: CODECOV_TOKEN
            value: "$CODECOV_TOKEN"
EOF

echo "hack/e2e-patch.sh: set environment variables in charts"
