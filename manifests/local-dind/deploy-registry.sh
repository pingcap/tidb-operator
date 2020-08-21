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

set -euo pipefail

# Run a local registry on master node
docker exec kube-master docker run -d --restart=always -v /registry:/var/lib/registry -p5001:5000 --name=registry registry:2

# run nginx on each node to proxy to docker registry on master node
master_ip=$(kubectl get po -n kube-system -lcomponent=etcd -o=jsonpath='{.items[0].status.hostIP}')
sed "s/10.192.0.2/${master_ip}/g" ./manifests/local-dind/registry-proxy-deployment.yaml | kubectl apply -f -
