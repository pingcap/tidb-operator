#!/bin/bash
set -o errexit
set -o nounset
set -o pipefail

CURDIR=$(cd $(dirname ${BASH_SOURCE[0]}); pwd )

CA_BUNDLE=$(kubectl get configmap -n kube-system extension-apiserver-authentication -o=jsonpath='{.data.client-ca-file}' | base64 | tr -d '\n')
echo $CA_BUNDLE
sed -i "s/caBundle: .*$/caBundle: ${CA_BUNDLE}/g" $CURDIR/webhook.yaml
