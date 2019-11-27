#!/bin/bash

# Copyright 2019 PingCAP, Inc.
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
# Isolated docker environment for development.
#
# Examples:
#
#   ./hack/run-in-docker.sh # start an interactive shell
#   ./hack/run-in-docker.sh make test
#   ./hack/run-in-docker.sh ./hack/e2e.sh -- --ginkgo.focus='aggregated'
#

set -o errexit
set -o nounset
set -o pipefail

ROOT=$(unset CDPATH && cd $(dirname "${BASH_SOURCE[0]}")/.. && pwd)
cd $ROOT

args=(bash)
if [ $# -gt 0 ]; then
    args=($@)
fi

docker_args=(
    -it --rm
)

# required by dind
docker_args+=(
    --privileged
    -e DOCKER_IN_DOCKER_ENABLED=true
    # Docker in Docker expects it to be a volume
    --tmpfs /var/lib/docker
    --tmpfs /docker-graph # legacy path for cr.io/k8s-testimages/kubekins-e2e
)

# required by kind
docker_args+=(
    -v /lib/modules:/lib/modules
    -v /sys/fs/cgroup:/sys/fs/cgroup
)

docker run ${docker_args[@]} \
    -v $ROOT:/go/src/github.com/pingcap/advanced-statefulset \
    -w /go/src/github.com/pingcap/advanced-statefulset \
    --entrypoint /usr/local/bin/runner.sh \
    gcr.io/k8s-testimages/kubekins-e2e:v20191108-9467d02-master \
    "${args[@]}"
