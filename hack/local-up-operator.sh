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
# This command runs tidb-operator in Kubernetes.
#

# Default provider is kind
PROVIDER=${PROVIDER:-kind}

# Function to display usage information
function usage() {
    cat <<'EOF'
This script runs tidb-operator in Kubernetes using the appropriate provider.

Usage: local-up-operator.sh [-hd] [-p PROVIDER]

    -h      Show this message and exit
    -i      Install dependencies only

Environments:

    PROVIDER              Kubernetes provider. Defaults: kind.
    CLUSTER               The name of the e2e cluster.
    KUBECONFIG            Path to the kubeconfig file, defaults: ~/.kube/config
    KUBECONTEXT           Context in kubeconfig file, defaults to the current context
    NAMESPACE             Kubernetes namespace in which we run our tidb-operator.
    DOCKER_REGISTRY       Image docker registry
    IMAGE_TAG             Image tag
    SKIP_IMAGE_BUILD      Skip build and push images

EOF
}

# Determine the appropriate script based on the provider
case "$PROVIDER" in
    kind)
        echo "Running with kind provider..."
        bash hack/local-up-by-kind.sh "$@"
        ;;
    k3s)
        echo "Running with k3s provider..."
        bash hack/local-up-by-k3s.sh "$@"
        ;;
    *)
        echo "Unsupported provider: $PROVIDER"
        usage
        exit 1
        ;;
esac
