#!/bin/sh
set -ex

BASE=$(realpath $(dirname $0))
SED=$(which gsed || echo 'sed')

namespace=testing
cluster=cluster

kubectl create -f $BASE/../../manifests/crd.yaml || kubectl replace -f $BASE/../../manifests/crd.yaml

kubectl delete --force namespace $namespace || true

if [ -n "$1" ]; then
	eval $(minikube docker-env)
	make DOCKER_REPO=xx operator-docker
fi

kubectl create namespace $namespace

if [ -n "$ENABLE_SSL" ]; then 
	mkdir -p $BASE/cfssl/certs && cd $BASE/cfssl/certs
	cfssl gencert -initca ../ca-csr.json | cfssljson -bare ca -
	for i in pd tikv tiproxy tidb; do
		$SED -e "s/\${component}/$i/g" -e "s/\${namespace}/$namespace/g" -e "s/\${cluster}/$cluster/g" ../comp-csr.json > csr.json
		cfssl gencert -ca=ca.pem -ca-key=ca-key.pem -config=../ca-config.json -profile=internal csr.json | cfssljson -bare $i
		kubectl create secret generic ${cluster}-${i}-cluster-secret --namespace=${namespace} --from-file=tls.crt=$i.pem --from-file=tls.key=$i-key.pem --from-file=ca.crt=ca.pem
		if [ "$i" = "tidb" ]; then
			cfssl gencert -ca=ca.pem -ca-key=ca-key.pem -config=../ca-config.json -profile=server csr.json | cfssljson -bare $i
			kubectl create secret generic ${cluster}-${i}-server-secret --namespace=${namespace} --from-file=tls.crt=$i.pem --from-file=tls.key=$i-key.pem --from-file=ca.crt=ca.pem
			cfssl gencert -ca=ca.pem -ca-key=ca-key.pem -config=../ca-config.json -profile=client csr.json | cfssljson -bare $i
			kubectl create secret generic ${cluster}-${i}-client-secret --namespace=${namespace} --from-file=tls.crt=$i.pem --from-file=tls.key=$i-key.pem --from-file=ca.crt=ca.pem
		fi
	done
	cfssl gencert -ca=ca.pem -ca-key=ca-key.pem -config=../ca-config.json -profile=client ../client-csr.json | cfssljson -bare client
	kubectl create secret generic ${cluster}-cluster-client-secret --namespace=${namespace} --from-file=tls.crt=client.pem --from-file=tls.key=client-key.pem --from-file=ca.crt=ca.pem
fi

helm install operator $BASE/../../charts/tidb-operator/ --namespace testing --set "operatorImage=xx/tidb-operator:latest"
kubectl apply -f $BASE/cluster.yaml --namespace testing
