#!/usr/bin/env bash

set -e

usage() {
    cat <<EOF

       -n,--namespace        Namespace where webhook service reside
EOF
    exit 1
}

optstring="n:c"
bundle=false

while getopts "$optstring" opt; do
    case $opt in
        n)
            namespace="$OPTARG"
            ;;
        c)
            bundle=true
            ;;
        *)
            usage
            ;;
    esac
done

namespace=${namespace:-tidb-admin}

CURDIR=$(cd $(dirname ${BASH_SOURCE[0]}); pwd )
os=`uname -s| tr '[A-Z]' '[a-z]'`

case ${os} in
  linux)
    sed -i "s/namespace: \${NAMESPACE}/namespace: ${namespace}/g" $CURDIR/webhook.yaml
    ;;
  darwin)
    sed -i "" "s/namespace: \${NAMESPACE}/namespace: ${namespace}/g" $CURDIR/webhook.yaml
    ;;
    *)
    echo "invalid os ${os}, only support Linux and Darwin" >&2
    ;;
esac

if [ "$bundle" = true ] ; then

CA_BUNDLE=$(kubectl get configmap -n kube-system extension-apiserver-authentication -o=jsonpath='{.data.client-ca-file}' | base64 | tr -d '\n')

case ${os} in
  linux)
    sed -i "s/caBundle: .*$/caBundle: ${CA_BUNDLE}/g" $CURDIR/webhook.yaml
    ;;
  darwin)
    sed -i "" "s/caBundle: .*$/caBundle: ${CA_BUNDLE}/g" $CURDIR/webhook.yaml
    ;;
    *)
    echo "invalid os ${os}, only support Linux and Darwin" >&2
    ;;
esac

fi
