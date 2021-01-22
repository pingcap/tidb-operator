#!/bin/bash

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

# This script patches the chart templates of tidb-operator to use
# in E2E tests.

set -e

echo "hack/e2e-patch-codecov.sh: PWD $PWD"

CONTROLLER_MANAGER_DEPLOYMENT=charts/tidb-operator/templates/controller-manager-deployment.yaml
SCHEDULER_DEPLOYMENT=charts/tidb-operator/templates/scheduler-deployment.yaml
DISCOVERY_DEPLOYMENT=charts/tidb-cluster/templates/discovery-deployment.yaml

echo "eplace the entrypoint to generate and upload the coverage profile"
sed -i 's/\/usr\/local\/bin\/tidb-controller-manager/\/e2e-entrypoint.sh\n          - \/usr\/local\/bin\/tidb-controller-manager\n          - -test.coverprofile=\/tmp\/coverage.txt\n          - E2E/g' \
    $CONTROLLER_MANAGER_DEPLOYMENT
sed -i 's/\/usr\/local\/bin\/tidb-scheduler/\/e2e-entrypoint.sh\n          - \/usr\/local\/bin\/tidb-scheduler\n          - -test.coverprofile=\/tmp\/coverage.txt\n          - E2E/g' \
    $SCHEDULER_DEPLOYMENT
sed -i 's/\/usr\/local\/bin\/tidb-discovery/\/e2e-entrypoint.sh\n          - \/usr\/local\/bin\/tidb-discovery\n          - -test.coverprofile=\/tmp\/coverage.txt\n          - E2E/g' \
    $DISCOVERY_DEPLOYMENT

# -v is duplicated for operator and go test
sed -i '/\-v=/d' $CONTROLLER_MANAGER_DEPLOYMENT
sed -i '/\-v=/d' $SCHEDULER_DEPLOYMENT

# populate needed environment variables
echo "hack/e2e-patch-codecov.sh: setting environment variables in charts"
cat << EOF >> $CONTROLLER_MANAGER_DEPLOYMENT
          - name: COMPONENT
            value: "controller-manager"
          - name: SRC_BRANCH
            value: "$SRC_BRANCH"
          - name: BUILD_NUMBER
            value: "$BUILD_NUMBER"
          - name: GIT_COMMIT
            value: "$GIT_COMMIT"
          - name: PR_ID
            value: "$PR_ID"
          - name: CODECOV_TOKEN
            value: "$CODECOV_TOKEN"
EOF

# for SCHEDULER_DEPLOYMENT, no `env:` added with default values.
cat << EOF >> $SCHEDULER_DEPLOYMENT
        env:
        - name: COMPONENT
          value: "scheduler"
        - name: SRC_BRANCH
          value: "$SRC_BRANCH"
        - name: BUILD_NUMBER
          value: "$BUILD_NUMBER"
        - name: GIT_COMMIT
          value: "$GIT_COMMIT"
        - name: PR_ID
          value: "$PR_ID"
        - name: CODECOV_TOKEN
          value: "$CODECOV_TOKEN"
EOF

cat << EOF >> $DISCOVERY_DEPLOYMENT
          - name: COMPONENT
            value: "discovery"
          - name: SRC_BRANCH
            value: "$SRC_BRANCH"
          - name: BUILD_NUMBER
            value: "$BUILD_NUMBER"
          - name: GIT_COMMIT
            value: "$GIT_COMMIT"
          - name: PR_ID
            value: "$PR_ID"
          - name: CODECOV_TOKEN
            value: "$CODECOV_TOKEN"
EOF

