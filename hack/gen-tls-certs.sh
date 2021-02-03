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

set -o errexit
set -o nounset
set -o pipefail

ROOT=$(unset CDPATH && cd $(dirname "${BASH_SOURCE[0]}")/.. && pwd)
cd $ROOT

source "${ROOT}/hack/lib.sh"

# parameters begin
TIDB_CLUSTER=${TIDB_CLUSTER:-"basic-tls"}
NAMESPACE=${NAMESPACE:-"default"}

CA_EXPIRATION=${CA_EXPIRATION:-"87600h"}
COMMON_NAME=${COMMON_NAME:-"TiDB"}
ORGANIZATION_UNIT=${ORGANIZATION_UNIT:-"TiDB"}
ORGANIZATION=${ORGANIZATION:-"PingCAP"}
COUNTRY=${COUNTRY:-"CN"}
LOCALITY=${LOCALITY:-"Beijing"}
STATE_OR_PROVINCE_NAME=${STATE_OR_PROVINCE_NAME:-"Beijing"}
# paramenters end

TLS_DIR=${TLS_DIR:-"${OUTPUT}/tls"}
CA_DIR=${CA_DIR:-"${TLS_DIR}/ca"}

function usage() {
    cat <<'EOF'
This script is used to generate TLS certs for tidb cluster/client using cfssl.

Usage: hack/gen-tls-certs.sh [-h]

    -h      show this message and exit

Environments:

    TIDB_CLUSTER           should match the tidb cluster you want to enable TLS features, defaults to "basic-tls"
    NAMESPACE              namespace of the tidb cluster, defaults to "default"
    # following are CA cert configs
    CA_EXPIRATION          expiration time of CA certs, defaults to "87600h" (10*365 Days)
    COMMON_NAME            COMMON_NAME(CN) of the CA cert, defaults to "TiDB"
    ORGANIZATION_UNIT      ORGANIZATION_UNIT(OU) of the CA cert, defaults to "TiDB"
    ORGANIZATION           ORGANIZATION(O) of the CA cert, defaults to "PingCAP"
    COUNTRY                COUNTRY(C) of the CA cert, defaults to "CN"
    LOCALITY               LOCALITY(L) of the CA cert, defaults to "Beijing"
    STATE_OR_PROVINCE_NAME STATE_OR_PROVINCE_NAME(ST) of the CA cert, defaults to "Beijing"

Examples:

0) view help

    ./hack/gen-tls-certs.sh -h

1) generate certs using default settings

    ./hack/gen-tls-certs.sh

2) generate certs with customized settings

    TIDB_CLUSTER=my-tidb-cluster NAMESPACE=my-namespace COMMON_NAME=my-org ORGANIZATION_UNIT=my-org ORGANIZATION=my-org ./hack/gen-tls-certs.sh

EOF

}

while getopts "h?" opt; do
    case "$opt" in
    h | \?)
        usage
        exit 0
        ;;
    esac
done

function gen_ca() {
    echo "generating CA"
    mkdir -p $CA_DIR
    pushd $CA_DIR

    cat <<EOF >ca-config.json
{
    "signing": {
        "default": {
            "expiry": "${CA_EXPIRATION}"
        },
        "profiles": {
            "internal": {
                "expiry": "${CA_EXPIRATION}",
                "usages": [
                    "signing",
                    "key encipherment",
                    "server auth",
                    "client auth"
                ]
            },
            "server": {
                "expiry": "${CA_EXPIRATION}",
                "usages": [
                    "signing",
                    "key encipherment",
                    "server auth"
                ]
            },
            "client": {
                "expiry": "${CA_EXPIRATION}",
                "usages": [
                    "signing",
                    "key encipherment",
                    "client auth"
                ]
            }
        }
    }
}
EOF

    cat <<EOF >ca-csr.json
{
    "CN": "${COMMON_NAME}",
    "key": {
        "algo": "rsa",
        "size": 2048
    },
    "names": [
        {
            "C": "${COUNTRY}",
            "L": "${LOCALITY}",
            "O": "${ORGANIZATION}",
            "ST": "${STATE_OR_PROVINCE_NAME}",
            "OU": "${ORGANIZATION_UNIT}"
        }
    ]
}
EOF

    $CFSSL_BIN gencert -initca ca-csr.json | $CFSSLJSON_BIN -bare ca -
    popd
}

function gen_component_certs() {
    local KIND=$1
    local COMP=$2
    local COMP_DIR=$TLS_DIR/$KIND/$2
    local PROFILE=$3
    echo "generating $COMP server certs"
    mkdir -p $COMP_DIR
    pushd $COMP_DIR
    $CFSSL_BIN print-defaults csr |
        $JQ_BIN ".hosts=[\"127.0.0.1\",\"::1\",\"${TIDB_CLUSTER}-${COMP}\",\"${TIDB_CLUSTER}-${COMP}.${NAMESPACE}\",\"${TIDB_CLUSTER}-${COMP}.${NAMESPACE}.svc\",\"${TIDB_CLUSTER}-${COMP}-peer\",\"${TIDB_CLUSTER}-${COMP}-peer.${NAMESPACE}\",\"${TIDB_CLUSTER}-${COMP}-peer.${NAMESPACE}.svc\",\"*.${TIDB_CLUSTER}-${COMP}-peer\",\"*.${TIDB_CLUSTER}-${COMP}-peer.${NAMESPACE}\",\"*.${TIDB_CLUSTER}-${COMP}-peer.${NAMESPACE}.svc\"]" |
        $JQ_BIN ".CN=\"${COMMON_NAME}\"" \
            >${COMP}-server.json
    $CFSSL_BIN gencert -ca=$CA_DIR/ca.pem -ca-key=$CA_DIR/ca-key.pem -config=$CA_DIR/ca-config.json -profile=$PROFILE ${COMP}-server.json | $CFSSLJSON_BIN -bare ${COMP}-server
    popd
}

function delete_secret() {
    set +o errexit
    local NAME=$1
    $KUBECTL_BIN delete secret ${TIDB_CLUSTER}-${NAME}-secret --namespace=${NAMESPACE}
    set -o errexit
}

function create_cluster_secret() {
    local COMP=$1
    delete_secret ${COMP}-cluster
    $KUBECTL_BIN create secret generic ${TIDB_CLUSTER}-${COMP}-cluster-secret --namespace=${NAMESPACE} \
        --from-file=tls.crt=${TLS_DIR}/cluster/${COMP}/${COMP}-server.pem \
        --from-file=tls.key=${TLS_DIR}/cluster/${COMP}/${COMP}-server-key.pem \
        --from-file=ca.crt=${CA_DIR}/ca.pem
}

hack::ensure_cfssl

echo "TLS_DIR: $TLS_DIR"
echo "CA_DIR: $CA_DIR"

if [[ -d $TLS_DIR ]]; then
    echo "tls dir exists, removing"
    rm -rf $TLS_DIR
fi

components=("pd" "tidb" "tikv" "pump" "ticdc" "tiflash" "importer" "lightning")

if [[ $# -eq 1 && $1 = "clean" ]]; then
    echo "cleaning up"
    for i in "${components[@]}"; do
        delete_secret $i-cluster
    done
    delete_secret cluster-client
    delete_secret tidb-server
    delete_secret tidb-client
    exit
fi

mkdir -p $TLS_DIR

gen_ca

# generate certs for cluster components
for i in "${components[@]}"; do
    gen_component_certs "cluster" $i "internal"
    create_cluster_secret $i
done
gen_component_certs "cluster" "client" "client"
# secret name not obey name convention of cluster components
delete_secret cluster-client
$KUBECTL_BIN create secret generic ${TIDB_CLUSTER}-cluster-client-secret --namespace=${NAMESPACE} \
    --from-file=tls.crt=${TLS_DIR}/cluster/client/client-server.pem \
    --from-file=tls.key=${TLS_DIR}/cluster/client/client-server-key.pem \
    --from-file=ca.crt=${CA_DIR}/ca.pem


# generate certs for mysql client and server
gen_component_certs "mysql" "tidb" "server"
# secret name not obey name convention of cluster components
delete_secret tidb-server
$KUBECTL_BIN create secret generic ${TIDB_CLUSTER}-tidb-server-secret --namespace=${NAMESPACE} \
    --from-file=tls.crt=${TLS_DIR}/mysql/tidb/tidb-server.pem \
    --from-file=tls.key=${TLS_DIR}/mysql/tidb/tidb-server-key.pem \
    --from-file=ca.crt=${CA_DIR}/ca.pem

gen_component_certs "mysql" "client" "client"
# secret name not obey name convention of cluster components
delete_secret tidb-client
$KUBECTL_BIN create secret generic ${TIDB_CLUSTER}-tidb-client-secret --namespace=${NAMESPACE} \
    --from-file=tls.crt=${TLS_DIR}/mysql/client/client-server.pem \
    --from-file=tls.key=${TLS_DIR}/mysql/client/client-server-key.pem \
    --from-file=ca.crt=${CA_DIR}/ca.pem
