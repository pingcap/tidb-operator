#!/usr/bin/env bash
set -euo pipefail

# Run a local registry on master node
docker exec kube-master docker run -d --restart=always -v /registry:/var/lib/registry -p5001:5000 --name=registry registry:2

# run nginx on each node to proxy to docker registry on master node
master_ip=$(kubectl get po -n kube-system -lcomponent=etcd -o=jsonpath='{.items[0].status.hostIP}')
sed "s/10.192.0.2/${master_ip}/g" ./manifests/local-dind/registry-proxy-deployment.yaml | kubectl apply -f -
