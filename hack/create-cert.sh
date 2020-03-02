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

set -e

usage() {
    cat <<EOF
create-cert.sh
create certification for pump and drainer conponent

  --namespace    The namespace of the component
  --component    The name of the component (e.g. pump, drainer)
  --cluster      The TiDB cluster name of the component for
EOF
    exit 1
}

while [[ $# -gt 0 ]]; do
    case ${1} in
        --namespace)
            namespace="$2"
            shift
            ;;
        --component)
            component="$2"
            shift
            ;;
        --cluster)
            cluster="$2"
            shift
            ;;
        *)
            usage
            ;;
    esac
    shift
done

if [ -z ${namespace} ]; then
	usage
fi

if [ -z ${component} ]; then
	usage
fi

if [ -z ${cluster} ]; then
	usage
fi

service=${cluster}-${component}
secret=${cluster}-${component}

if [ ! -x "$(command -v openssl)" ]; then
    echo "openssl not found"
    exit 1
fi

csrName=${service}.${namespace}
tmpdir=$(mktemp -d)

echo ${tmpdir}

cat <<EOF >> ${tmpdir}/csr.conf
[req]
req_extensions = v3_req
distinguished_name = req_distinguished_name
[req_distinguished_name]
[ v3_req ]
basicConstraints = CA:FALSE
keyUsage = nonRepudiation, digitalSignature, keyEncipherment
extendedKeyUsage = serverAuth
subjectAltName = @alt_names
[alt_names]
DNS.1 = ${service}
DNS.2 = ${service}.${namespace}
DNS.3 = ${service}.${namespace}.svc
DNS.4 = *.${service}
DNS.5 = *.${service}.${namespace}
DNS.5 = *.${service}.${namespace}.svc
IP.1 = 127.0.0.1
EOF

openssl genrsa -out ${tmpdir}/server-key.pem 2048
openssl req -new -key ${tmpdir}/server-key.pem -subj "/CN=${service}.${namespace}.svc" -out ${tmpdir}/server.csr -config ${tmpdir}/csr.conf

 # clean-up any previously created CSR for our service. Ignore errors if not present.
kubectl delete csr ${csrName} 2>/dev/null || true

 # create  server cert/key CSR and  send to k8s API
cat <<EOF | kubectl create -f -
apiVersion: certificates.k8s.io/v1beta1
kind: CertificateSigningRequest
metadata:
  name: ${csrName}
spec:
  groups:
  - system:authenticated
  request: $(cat ${tmpdir}/server.csr | base64 | tr -d '\n')
  usages:
  - digital signature
  - key encipherment
  - server auth
  - client auth
EOF

 # verify CSR has been created
while true; do
    kubectl get csr ${csrName}
    if [ "$?" -eq 0 ]; then
        break
    fi
    echo "INFO: failed to get csr, try again after 1 s."
    sleep 1
done

 # approve and fetch the signed certificate
kubectl certificate approve ${csrName}
# verify certificate has been signed
for x in $(seq 10); do
    serverCert=$(kubectl get csr ${csrName} -o jsonpath='{.status.certificate}')
    if [[ ${serverCert} != '' ]]; then
        break
    fi
    sleep 1
done
if [[ ${serverCert} == '' ]]; then
    echo "ERROR: After approving csr ${csrName}, the signed certificate did not appear on the resource. Giving up after 10 attempts." >&2
    exit 1
fi

 echo ${serverCert} | openssl base64 -d -A -out ${tmpdir}/server-cert.pem

 # create the secret with CA cert and server cert/key
kubectl create secret tls ${secret} \
        --key=${tmpdir}/server-key.pem \
        --cert=${tmpdir}/server-cert.pem \
        --dry-run -o yaml |
    kubectl -n ${namespace} apply -f -
