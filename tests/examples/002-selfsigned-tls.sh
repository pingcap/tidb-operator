#!/bin/bash

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

ROOT=$(unset CDPATH && cd $(dirname "${BASH_SOURCE[0]}")/../.. && pwd)
cd $ROOT

source "${ROOT}/hack/lib.sh"
source "${ROOT}/tests/examples/t.sh"

NS=$(basename ${0%.*})
CERT_MANAGER_VERSION=0.14.1

PORT_FORWARD_PID=

function cleanup() {
    if [ -n "$PORT_FORWARD_PID" ]; then
        echo "info: kill port-forward background process (PID: $PORT_FORWARD_PID)"
        kill $PORT_FORWARD_PID
    fi
    kubectl delete -f examples/selfsigned-tls/ --ignore-not-found
    kubectl delete -f https://github.com/jetstack/cert-manager/releases/download/v${CERT_MANAGER_VERSION}/cert-manager.yaml --ignore-not-found
    kubectl delete ns $NS
}

trap cleanup EXIT

kubectl create ns $NS
hack::wait_for_success 10 3 "t::ns_is_active $NS"

kubectl apply --validate=false -f https://github.com/jetstack/cert-manager/releases/download/v${CERT_MANAGER_VERSION}/cert-manager.yaml
hack::wait_for_success 10 3 "t::crds_are_ready certificaterequests.cert-manager.io certificates.cert-manager.io challenges.acme.cert-manager.io clusterissuers.cert-manager.io issuers.cert-manager.io orders.acme.cert-manager.io"
for d in cert-manager cert-manager-cainjector cert-manager-webhook; do
    hack::wait_for_success 300 3 "t::deploy_is_ready cert-manager $d"
    if [ $? -ne 0 ]; then
        echo "fatal: timed out waiting for the deployment $d to be ready"
        exit 1
    fi
done

kubectl -n $NS apply -f examples/selfsigned-tls/

hack::wait_for_success 300 3 "t::tc_is_ready $NS tls"
if [ $? -ne 0 ]; then
    echo "fatal: failed to wait for the cluster to be ready"
    exit 1
fi

echo "info: verify mysql client can connect with tidb server with SSL enabled"
kubectl -n $NS port-forward svc/tls-tidb 4000:4000 &> /tmp/port-forward.log &
PORT_FORWARD_PID=$!

host=127.0.0.1
port=4000
for ((i=0; i < 10; i++)); do
	nc -zv -w 3 $host $port
	if [ $? -eq 0 ]; then
		break
	else
		echo "info: failed to connect to $host:$port, sleep 1 second then retry"
		sleep 1
	fi
done

hack::wait_for_success 100 3 "mysql -h 127.0.0.1 -P 4000 -uroot -e 'select tidb_version();'"
if [ $? -ne 0 ]; then
    echo "fatal: failed to connect to TiDB"
    exit 1
fi

has_ssl=$(mysql -h 127.0.0.1 -P 4000 -uroot --ssl -e "SHOW VARIABLES LIKE '%ssl%';" | awk '/have_ssl/ {print $2}')
if [[ "$has_ssl" != "YES" ]]; then
	echo "fatal: ssl is not enabled successfully, has_ssl is '$has_ssl'"
	exit 1
fi
echo "info: ssl is enabled successfully, has_ssl is '$has_ssl'"
