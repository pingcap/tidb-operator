#!/usr/bin/env bash
set -o errexit
set -o nounset
set -o pipefail

CURDIR=$(cd $(dirname ${BASH_SOURCE[0]}); pwd )

CA_BUNDLE=$(kubectl get configmap -n kube-system extension-apiserver-authentication -o=jsonpath='{.data.client-ca-file}' | base64 | tr -d '\n')
echo $CA_BUNDLE

os=`uname -s| tr '[A-Z]' '[a-z]'`
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
