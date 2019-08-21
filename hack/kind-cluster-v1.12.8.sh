#!/usr/bin/env bash

k8s_version=v1.12.8
clusterName=${1:-kind}

echo "############ install kind ##############"
if hash kind 2>/dev/null;then
    echo "kind have installed"
else
    echo "start install kind"
    GO111MODULE="on" go get sigs.k8s.io/kind@v0.4.0
    echo PATH=\$PATH:$(go env GOPATH)/bin >> /etc/profile && source /etc/profile
fi

echo "############# start create cluster:[${clusterName}] #############"

data_dir=/var/kind/${clusterName}/data

echo "clean data dir: ${data_dir}"
if [ -d ${data_dir} ]; then
    rm -rf ${data_dir}
fi

echo "init the mount dirs for kind cluster"
mkdir -p ${data_dir}/worker{,1,2,3,4,5}/vol{1..9}

cat <<EOF > kind-config.yaml
kind: Cluster
apiVersion: kind.sigs.k8s.io/v1alpha3
networking:
  disableDefaultCNI: true
nodes:
- role: control-plane
- role: worker
  extraPortMappings:
  - containerPort: 5001
    hostPort: 5000
    listenAddress: 0.0.0.0
    protocol: TCP
  extraMounts:
  - containerPath: /mnt/disks/vol1
    hostPath: ${data_dir}/worker/vol1
  - containerPath: /mnt/disks/vol2
    hostPath: ${data_dir}/worker/vol2
  - containerPath: /mnt/disks/vol3
    hostPath: ${data_dir}/worker/vol3
  - containerPath: /mnt/disks/vol4
    hostPath: ${data_dir}/worker/vol4
  - containerPath: /mnt/disks/vol5
    hostPath: ${data_dir}/worker/vol5
  - containerPath: /mnt/disks/vol6
    hostPath: ${data_dir}/worker/vol6
  - containerPath: /mnt/disks/vol7
    hostPath: ${data_dir}/worker/vol7
  - containerPath: /mnt/disks/vol8
    hostPath: ${data_dir}/worker/vol8
  - containerPath: /mnt/disks/vol9
    hostPath: ${data_dir}/worker/vol9
- role: worker
  extraMounts:
  - containerPath: /mnt/disks/vol1
    hostPath: ${data_dir}/worker1/vol1
  - containerPath: /mnt/disks/vol2
    hostPath: ${data_dir}/worker1/vol2
  - containerPath: /mnt/disks/vol3
    hostPath: ${data_dir}/worker1/vol3
  - containerPath: /mnt/disks/vol4
    hostPath: ${data_dir}/worker1/vol4
  - containerPath: /mnt/disks/vol5
    hostPath: ${data_dir}/worker1/vol5
  - containerPath: /mnt/disks/vol6
    hostPath: ${data_dir}/worker1/vol6
  - containerPath: /mnt/disks/vol7
    hostPath: ${data_dir}/worker1/vol7
  - containerPath: /mnt/disks/vol8
    hostPath: ${data_dir}/worker1/vol8
  - containerPath: /mnt/disks/vol9
    hostPath: ${data_dir}/worker1/vol9
- role: worker
  extraMounts:
  - containerPath: /mnt/disks/vol1
    hostPath: ${data_dir}/worker2/vol1
  - containerPath: /mnt/disks/vol2
    hostPath: ${data_dir}/worker2/vol2
  - containerPath: /mnt/disks/vol3
    hostPath: ${data_dir}/worker2/vol3
  - containerPath: /mnt/disks/vol4
    hostPath: ${data_dir}/worker2/vol4
  - containerPath: /mnt/disks/vol5
    hostPath: ${data_dir}/worker2/vol5
  - containerPath: /mnt/disks/vol6
    hostPath: ${data_dir}/worker2/vol6
  - containerPath: /mnt/disks/vol7
    hostPath: ${data_dir}/worker2/vol7
  - containerPath: /mnt/disks/vol8
    hostPath: ${data_dir}/worker2/vol8
  - containerPath: /mnt/disks/vol9
    hostPath: ${data_dir}/worker2/vol9
- role: worker
  extraMounts:
  - containerPath: /mnt/disks/vol1
    hostPath: ${data_dir}/worker3/vol1
  - containerPath: /mnt/disks/vol2
    hostPath: ${data_dir}/worker3/vol2
  - containerPath: /mnt/disks/vol3
    hostPath: ${data_dir}/worker3/vol3
  - containerPath: /mnt/disks/vol4
    hostPath: ${data_dir}/worker3/vol4
  - containerPath: /mnt/disks/vol5
    hostPath: ${data_dir}/worker3/vol5
  - containerPath: /mnt/disks/vol6
    hostPath: ${data_dir}/worker3/vol6
  - containerPath: /mnt/disks/vol7
    hostPath: ${data_dir}/worker3/vol7
  - containerPath: /mnt/disks/vol8
    hostPath: ${data_dir}/worker3/vol8
  - containerPath: /mnt/disks/vol9
    hostPath: ${data_dir}/worker3/vol9
- role: worker
  extraMounts:
  - containerPath: /mnt/disks/vol1
    hostPath: ${data_dir}/worker4/vol1
  - containerPath: /mnt/disks/vol2
    hostPath: ${data_dir}/worker4/vol2
  - containerPath: /mnt/disks/vol3
    hostPath: ${data_dir}/worker4/vol3
  - containerPath: /mnt/disks/vol4
    hostPath: ${data_dir}/worker4/vol4
  - containerPath: /mnt/disks/vol5
    hostPath: ${data_dir}/worker4/vol5
  - containerPath: /mnt/disks/vol6
    hostPath: ${data_dir}/worker4/vol6
  - containerPath: /mnt/disks/vol7
    hostPath: ${data_dir}/worker4/vol7
  - containerPath: /mnt/disks/vol8
    hostPath: ${data_dir}/worker4/vol8
  - containerPath: /mnt/disks/vol9
    hostPath: ${data_dir}/worker4/vol9
- role: worker
  extraMounts:
  - containerPath: /mnt/disks/vol1
    hostPath: ${data_dir}/worker5/vol1
  - containerPath: /mnt/disks/vol2
    hostPath: ${data_dir}/worker5/vol2
  - containerPath: /mnt/disks/vol3
    hostPath: ${data_dir}/worker5/vol3
  - containerPath: /mnt/disks/vol4
    hostPath: ${data_dir}/worker5/vol4
  - containerPath: /mnt/disks/vol5
    hostPath: ${data_dir}/worker5/vol5
  - containerPath: /mnt/disks/vol6
    hostPath: ${data_dir}/worker5/vol6
  - containerPath: /mnt/disks/vol7
    hostPath: ${data_dir}/worker5/vol7
  - containerPath: /mnt/disks/vol8
    hostPath: ${data_dir}/worker5/vol8
  - containerPath: /mnt/disks/vol9
    hostPath: ${data_dir}/worker5/vol9
EOF

echo "start to create k8s cluster"
kind create cluster --config kind-config.yaml --image kindest/node:${k8s_version} --name=${clusterName}
export KUBECONFIG="$(kind get kubeconfig-path --name=${clusterName})"

echo "init e2e test env"
kubectl apply -f manifests/local-dind/kube-flannel.yaml
kubectl apply -f manifests/local-dind/local-volume-provisioner.yaml
kubectl apply -f manifests/tiller-rbac.yaml
kubectl apply -f manifests/crd.yaml

echo "############# success create cluster:[${clusterName}] #############"
