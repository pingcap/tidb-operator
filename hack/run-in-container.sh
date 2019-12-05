#!/usr/bin/env bash

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
# Isolated container environment for development.
#
# Examples:
#
#   ./hack/run-in-container.sh # start an interactive shell
#   CLEANUP=y ./hack/run-in-container.sh # clean local volumes
#   ./hack/run-in-container.sh make test # exec into the container (start if not running) and run commands
#   ./hack/run-in-container.sh ./hack/e2e.sh -- --ginkgo.focus='aggregated'
#

set -o errexit
set -o nounset
set -o pipefail

ROOT=$(unset CDPATH && cd $(dirname "${BASH_SOURCE[0]}")/.. && pwd)
cd $ROOT

CLEANUP=${CLEANUP:-} # if set, cleaning up local volumes
DOCKER_LIB_VOLUME=${DOCKER_LIB_VOLUME:-tidb-operator-docker-lib}
DOCKER_GRAPH_VOLUME=${DOCKER_GRAPH_VOLUME:-tidb-operator-docker-graph}
DOCKER_GO_VOLUME=${DOCKER_GO_VOLUME:-tidb-operator-go}
NAME=${NAME:-tidb-operator-dev}

args=(bash)
if [ $# -gt 0 ]; then
    args=($@)
fi

docker_args=(
    -it --rm
    --name $NAME
)

# required by dind
docker_args+=(
    --privileged
    -e DOCKER_IN_DOCKER_ENABLED=true
    # Docker in Docker expects it to be a volume
    -v $DOCKER_LIB_VOLUME:/var/lib/docker
    -v $DOCKER_GRAPH_VOLUME:/docker-graph # legacy path for gcr.io/k8s-testimages/kubekins-e2e
    # golang cache
    -v $DOCKER_GO_VOLUME:/go
    # golang xdg cache directory
    -e XDG_CACHE_HOME=/go/cache
)

# required by kind
docker_args+=(
    -v /lib/modules:/lib/modules
    -v /sys/fs/cgroup:/sys/fs/cgroup
)

function cleanup() {
    local volumes=(
        $DOCKER_LIB_VOLUME
        $DOCKER_GRAPH_VOLUME
        $DOCKER_GO_VOLUME
    )
    for v in "${volumes[@]}"; do
        echo "info: cleaning up volume $v"
        docker volume rm $v || true
    done
}

if [ -n "$CLEANUP" ]; then
    cleanup
    exit
fi

ret=0
sts=$(docker inspect ${NAME} -f '{{.State.Status}}' 2>/dev/null) || ret=$?
if [ $ret -eq 0 ]; then
    if [[ "$sts" == "running" ]]; then
        echo "info: found a running container named '${NAME}', trying to exec into it" >&2
        exec docker exec -it ${NAME} "${args[@]}"
    else
        echo "info: found a non-running ($sts) container named '${NAME}', removing it first" >&2
        docker rm ${NAME}
    fi
fi

docker run ${docker_args[@]} \
    -v $ROOT:/go/src/github.com/pingcap/tidb-operator \
    -w /go/src/github.com/pingcap/tidb-operator \
    --entrypoint /usr/local/bin/runner.sh \
    gcr.io/k8s-testimages/kubekins-e2e:v20191108-9467d02-master \
    "${args[@]}"
