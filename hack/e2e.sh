#!/usr/bin/env bash
#
# E2E entrypoint script.
#

set -o errexit
set -o nounset
set -o pipefail

ROOT=$(unset CDPATH && cd $(dirname "${BASH_SOURCE[0]}")/.. && pwd)
cd $ROOT

function usage() {
    cat <<'EOF'
This script is entrypoint to run e2e tests.

Usage: hack/e2e.sh [-h] -- [extra test args]

    -h      show this message and exit

Environments:

	DOCKER_REGISTRY		image docker registry
	IMAGE_TAG			image tag
	SKIP_BUILD 			skip building binaries
	SKIP_IMAGE_BUILD	skip build and push images
EOF

}

while getopts "h?" opt; do
    case "$opt" in
    h|\?)
        usage
        exit 0
        ;;  
    esac
done

DOCKER_REGISTRY=${DOCKER_REGISTRY:-localhost:5000}
IMAGE_TAG=${IMAGE_TAG:-latest}
SKIP_BUILD=${SKIP_BUILD:-}
SKIP_IMAGE_BUILD=${SKIP_IMAGE_BUILD:-}

echo "DOCKER_REGISTRY: $DOCKER_REGISTRY"
echo "IMAGE_TAG: $IMAGE_TAG"
echo "SKIP_BUILD: $SKIP_BUILD"
echo "SKIP_IMAGE_BUILD: $SKIP_IMAGE_BUILD"

if [ -n "$SKIP_BUILD" ]; then
    echo "info: skip building images"
    export NO_BUILD=y
fi

if [ -n "$SKIP_IMAGE_BUILD" ]; then
    echo "info: skip building and pushing images"
else
    DOCKER_REGISTRY=$DOCKER_REGISTRY IMAGE_TAG=$IMAGE_TAG make docker-push
    DOCKER_REGISTRY=$DOCKER_REGISTRY IMAGE_TAG=$IMAGE_TAG make e2e-docker-push
fi

# in kind cluster, we must use local registry
# TODO: find a better way
export TIDB_OPERATOR_IMAGE=localhost:5000/pingcap/tidb-operator:${IMAGE_TAG}
export E2E_IMAGE=localhost:5000/pingcap/tidb-operator-e2e:${IMAGE_TAG}
export TEST_APISERVER_IMAGE=localhost:5000/pingcap/test-apiserver:${IMAGE_TAG}

hack/run-e2e.sh "$@"
