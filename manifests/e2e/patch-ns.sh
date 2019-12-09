#!/usr/bin/env bash
set -o errexit
set -o nounset
set -o pipefail

CURDIR=$(cd $(dirname ${BASH_SOURCE[0]}); pwd )
os=`uname -s| tr '[A-Z]' '[a-z]'`

case ${os} in
  linux)
    sed -i "s/namespace:.*/namespace: \${NAMESPACE}/g" $CURDIR/webhook.yaml
    sed -i "s/namespace: \${NAMESPACE}/namespace: ${namespace}/g" $CURDIR/webhook.yaml
    ;;
  darwin)
    sed -i "" "s/namespace:.*/namespace: \${NAMESPACE}/g" $CURDIR/webhook.yaml
    sed -i "" "s/namespace: \${NAMESPACE}/namespace: ${namespace}/g" $CURDIR/webhook.yaml
    ;;
    *)
    echo "invalid os ${os}, only support Linux and Darwin" >&2
    ;;
esac

