set -e

usage() {
    cat <<EOF
Generate certificate suitable for use with an webhook service.

This script uses k8s' CertificateSigningRequest API to a generate a
certificate signed by k8s CA suitable for use with webhook
services. This requires permissions to create and approve CSR. See
https://kubernetes.io/docs/tasks/tls/managing-tls-in-a-cluster for
detailed explantion and additional instructions.

The server key/cert k8s CA cert are stored in a k8s secret.

       -n,--namespace        Namespace where service and secret reside.
       -s,--service          Service which cert created for
       -d,--days             days how long cert verified
EOF
    exit 1
}

optstring=":-:n:s:d"

while getopts "$optstring" opt; do
    case $opt in
        -)
            case "$OPTARG" in
                namespace)
                    namespace="${2}"
                    ;;
                service)
                    service="${4}"
                    ;;
                days)
                    days="${6}"
                    ;;
                *)
                    usage
                    ;;
            esac
            ;;
        n)
            namespace="${2}"
            ;;
        s)
            service="${4}"
            ;;
        d)
            days="${4}"
            ;;
        *)
            usage
            ;;
    esac
done

csrName=${service}.${namespace}-csr
secret=${service}-cert

echo namespace=${namespace}
echo service=${service}
echo secret=${secret}
echo csr=${csrName}
echo days=${days}


if [ ! -x "$(command -v openssl)" ]; then
    echo "openssl not found"
    exit 1
fi

tmpdir=$(mktemp -d)

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
EOF

openssl genrsa -out ${tmpdir}/server-key.pem 2048
openssl req -days ${days} -new -key ${tmpdir}/server-key.pem -subj "/CN=${service}.${namespace}.svc" -out ${tmpdir}/server.csr -config ${tmpdir}/csr.conf

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
EOF

# verify CSR has been created
while true; do
    kubectl get csr ${csrName}
    if [ "$?" -eq 0 ]; then
        break
    fi
    sleep 1
done

# approve and fetch the signed certificate
kubectl certificate approve ${csrName}
# verify certificate has been signed
for x in $(seq 600); do
    serverCert=$(kubectl get csr ${csrName} -o jsonpath='{.status.certificate}')
    if [[ ${serverCert} != '' ]]; then
        break
    fi
    sleep 1
done
if [[ ${serverCert} == '' ]]; then
    echo "ERROR: After csr ${csrName} approved, the signed certificate does not appear in the resource. Give up after 10 minutes of attempts." >&2
    exit 1
fi

echo ${serverCert} | openssl base64 -d -A -out ${tmpdir}/server-cert.pem

# clean-up any previously created secret for our service. Ignore errors if not present.
kubectl delete secret ${secret} 2>/dev/null || true

# create the secret with CA cert and server cert/key
kubectl create secret generic ${secret} \
        --from-file=key.pem=${tmpdir}/server-key.pem \
        --from-file=cert.pem=${tmpdir}/server-cert.pem \
        --dry-run -o yaml |
    kubectl -n ${namespace} apply -f -

# verify secret has been created
while true; do
    kubectl get secret ${secret}
    if [ "$?" -eq 0 ]; then
        break
    fi
    sleep 1
done

kubectl label secret ${secret} app.kubernetes.io/managed-by=tidb-operator
kubectl label secret ${secret} app.kubernetes.io/component=initializer
kubectl label secret ${secret} tidb.pingcap.com/cert-service=${service}
