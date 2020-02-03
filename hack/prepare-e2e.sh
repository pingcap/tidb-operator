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

#
# This script runs before all e2e jobs to prepare shared files.
# This is optional. We only use it in CI to speed up our e2e process.
#

set -o errexit
set -o nounset
set -o pipefail

ROOT=$(unset CDPATH && cd $(dirname "${BASH_SOURCE[0]}")/.. && pwd)
cd $ROOT

source "${ROOT}/hack/lib.sh"

DOCKER_REGISTRY=${DOCKER_REGISTRY:-localhost:5000}
IMAGE_TAG=${IMAGE_TAG:-latest}

hack::ensure_kind
hack::ensure_kubectl
hack::ensure_helm

DOCKER_REGISTRY=$DOCKER_REGISTRY IMAGE_TAG=$IMAGE_TAG make docker
DOCKER_REGISTRY=$DOCKER_REGISTRY IMAGE_TAG=$IMAGE_TAG make e2e-docker
docker save -o output/tidb-operator-$IMAGE_TAG.tar.gz $DOCKER_REGISTRY/pingcap/tidb-operator:$IMAGE_TAG
docker save -o output/tidb-operator-e2e-$IMAGE_TAG.tar.gz $DOCKER_REGISTRY/pingcap/tidb-operator-e2e:$IMAGE_TAG
