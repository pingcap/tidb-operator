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

ROOT=$(cd $(dirname "${BASH_SOURCE[0]}")/../..; pwd -P)

source $ROOT/hack/lib/vars.sh

OUTPUT_DIR=$ROOT/_output
IMAGE_DIR=$OUTPUT_DIR/image
CACHE_DIR=$OUTPUT_DIR/cache

declare -A NEED_PREFIX
NEED_PREFIX["prestop-checker"]=1

function image::build() {
    local targets=()
    local with_push=0
    local dockerfile="Dockerfile"
    local dockerignore=""
    while [[ $# -gt 0 ]]; do
        case $1 in
        --push)
            with_push=1
            shift
            ;;
        --dockerfile=*)
            dockerfile="${1#*=}"
            shift
            ;;
        --dockerignore=*)
            dockerignore="${1#*=}"
            shift
            ;;
        -*|--*)
            echo "Unknown option $1"
            exit 1
            ;;
        *)
            targets+=("$1") # save positional arg
            shift # past argument
            ;;
        esac
    done



    mkdir -p ${IMAGE_DIR}
    mkdir -p ${CACHE_DIR}

    local args=""
    if [[ -n "$V_PLATFORMS" ]]; then
        args="--platform $V_PLATFORMS"
    fi

    # Check if current builder's driver is 'docker-container'
    if docker buildx ls | grep "*" | grep -q "docker-container"; then
      echo "'docker-container' exists, no need to execute the 'docker buildx create --use' command."
    else
      echo "'docker-container' does not exist, executing 'docker buildx create --use'..."
      docker buildx create --use
    fi


    for target in ${targets[@]}; do
        local image=${target}
        if [[ -n "${NEED_PREFIX[$target]+x}" ]]; then
            image=tidb-operator-${image}
        fi
        echo "build image ${image} using ${dockerfile}"
        
        # Handle custom dockerignore file
        local dockerignore_backup=""
        if [[ -n "$dockerignore" ]]; then
            # Backup original .dockerignore if it exists
            if [[ -f "$ROOT/.dockerignore" ]]; then
                dockerignore_backup="$ROOT/.dockerignore.backup.$$"
                mv "$ROOT/.dockerignore" "$dockerignore_backup"
            fi
            # Copy custom dockerignore
            cp "$ROOT/$dockerignore" "$ROOT/.dockerignore"
        fi
        
        docker buildx build \
            --target $target \
            -o type=oci,dest=$IMAGE_DIR/${target}.tar \
            -t ${V_IMG_PROJECT}/${image}:${V_RELEASE} \
            --cache-from=type=local,src=$CACHE_DIR \
            --cache-to=type=local,dest=$CACHE_DIR \
            --build-arg=TARGET="${target}" \
            $args \
            -f $ROOT/image/${dockerfile} $ROOT
            
        # Restore original .dockerignore
        if [[ -n "$dockerignore" ]]; then
            rm -f "$ROOT/.dockerignore"
            if [[ -n "$dockerignore_backup" && -f "$dockerignore_backup" ]]; then
                mv "$dockerignore_backup" "$ROOT/.dockerignore"
            fi
        fi
    done

    case $V_IMG_HUB in
    kind)
        for target in ${targets[@]}; do
            if [[ $with_push -eq 1 ]]; then
                echo "load ${target} image into kind cluster"
                $V_KIND load image-archive $IMAGE_DIR/${target}.tar --name ${V_KIND_CLUSTER}
            fi
        done
        ;;
    *)
        echo "Unknown image hub: ${V_IMG_HUB}"
        echo "Please see ./hack/lib/vars.sh#V_IMG_HUB"
        return 1
        ;;
    esac
}

# Prepare tidb components' images for e2e tests.
function image:prepare() {
    echo "load tidb components' images into kind cluster"
    for component in pd tikv tidb tiflash; do
        for version in "$V_TIDB_CLUSTER_VERSION" "$V_TIDB_CLUSTER_VERSION_PREV"; do
            docker pull gcr.io/pingcap-public/dbaas/$component:"$version" -q && \
            $V_KIND load docker-image gcr.io/pingcap-public/dbaas/$component:"$version" --name ${V_KIND_CLUSTER}
        done
    done
}
