#!/bin/bash

# Default provider is kind
PROVIDER=${PROVIDER:-kind}

# Function to display usage information
function usage() {
    cat <<'EOF'
This script runs tidb-operator in Kubernetes using the appropriate provider.

Usage: router.sh [-hd] [-p PROVIDER]

    -h      Show this message and exit
    -i      Install dependencies only
    -p      Specify the Kubernetes provider (kind or k3s). Defaults to kind.

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

# Parse command-line arguments
while getopts "hip:" opt; do
    case ${opt} in
        p)
            PROVIDER=${OPTARG}
            ;;
        *)
            usage
            exit 1
            ;;
    esac
done

shift $((OPTIND - 1))

# Determine the appropriate script based on the provider
case "$PROVIDER" in
    kind)
        echo "Running with kind provider..."
        # Call the script for kind provider (replace with actual path if needed)
        bash hack/local-up-by-kind.sh "$@"
        ;;
    k3s)
        echo "Running with k3s provider..."
        # Call the script for k3s provider (replace with actual path if needed)
        bash hack/local-up-by-k3s.sh "$@"
        ;;
    *)
        echo "Unsupported provider: $PROVIDER"
        usage
        exit 1
        ;;
esac
