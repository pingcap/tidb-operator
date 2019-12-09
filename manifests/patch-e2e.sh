#!/usr/bin/env bash

set -e

usage() {
    cat <<EOF

       -n,--namespace        Namespace where webhook service reside
EOF
    exit 1
}

optstring=":-:n"

while getopts "$optstring" opt; do
    case $opt in
        -)
            case "$OPTARG" in
                namespace)
                    namespace="${2}"
                    ;;
                *)
                    usage
                    ;;
            esac
            ;;
        n)
            namespace="${2}"
            ;;
        *)
            usage
            ;;
    esac
done

namespace=${namespace:-tidb-admin}

CURDIR=$(cd $(dirname ${BASH_SOURCE[0]}); pwd )
os=`uname -s| tr '[A-Z]' '[a-z]'`

CA_BUNDLE=$(kubectl get configmap -n kube-system extension-apiserver-authentication -o=jsonpath='{.data.client-ca-file}' | base64 | tr -d '\n')
echo $CA_BUNDLE


case ${os} in
  linux)
    sed -i "s/caBundle: .*$/caBundle: ${CA_BUNDLE}/g" $CURDIR/webhook.yaml
    sed -i "s/namespace: \${NAMESPACE}/namespace: ${namespace}/g" $CURDIR/webhook.yaml
    ;;
  darwin)
    sed -i "" "s/caBundle: .*$/caBundle: ${CA_BUNDLE}/g" $CURDIR/webhook.yaml
    sed -i "" "s/namespace: \${NAMESPACE}/namespace: ${namespace}/g" $CURDIR/webhook.yaml
    ;;
    *)
    echo "invalid os ${os}, only support Linux and Darwin" >&2
    ;;
esac
