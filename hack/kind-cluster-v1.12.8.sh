#!/usr/bin/env bash
set -e

clusterName=${1:-kind}
nodeNum=${2:-6}
k8s_version=v1.12.8

echo "############ install kind ##############"
if hash kind 2>/dev/null;then
    echo "kind have installed"
else
    echo "start install kind"
    GO111MODULE="on" go get sigs.k8s.io/kind@v0.4.0
    echo PATH=\${PATH}:$(go env GOPATH)/bin >> ${HOME}/.profile && source ${HOME}/.profile
fi

echo "############# start create cluster:[${clusterName}] #############"

data_dir=${HOME}/kind/${clusterName}/data

echo "clean data dir: ${data_dir}"
if [ -d ${data_dir} ]; then
    rm -rf ${data_dir}
fi

echo "init the mount dirs for kind cluster"
maxindex=$[nodeNum-1]
mkdir -p ${data_dir}/worker{0..${maxindex}}/vol{1..9}

configFile=${HOME}/kind/${clusterName}/kind-config.yaml

cat <<EOF > ${configFile}
kind: Cluster
apiVersion: kind.sigs.k8s.io/v1alpha3
networking:
  disableDefaultCNI: true
nodes:
- role: control-plane
EOF

for ((i=0;i<=${maxindex};i++))
do
cat <<EOF >>  ${configFile}
- role: worker
  extraMounts:
  - containerPath: /mnt/disks/vol1
    hostPath: ${data_dir}/worker${i}/vol1
  - containerPath: /mnt/disks/vol2
    hostPath: ${data_dir}/worker${i}/vol2
  - containerPath: /mnt/disks/vol3
    hostPath: ${data_dir}/worker${i}/vol3
  - containerPath: /mnt/disks/vol4
    hostPath: ${data_dir}/worker${i}/vol4
  - containerPath: /mnt/disks/vol5
    hostPath: ${data_dir}/worker${i}/vol5
  - containerPath: /mnt/disks/vol6
    hostPath: ${data_dir}/worker${i}/vol6
  - containerPath: /mnt/disks/vol7
    hostPath: ${data_dir}/worker${i}/vol7
  - containerPath: /mnt/disks/vol8
    hostPath: ${data_dir}/worker${i}/vol8
  - containerPath: /mnt/disks/vol9
    hostPath: ${data_dir}/worker${i}/vol9
EOF
done

echo "start to create k8s cluster"
kind create cluster --config ${configFile} --image kindest/node:${k8s_version} --name=${clusterName}
export KUBECONFIG="$(kind get kubeconfig-path --name=${clusterName})"

echo "init tidb-operator env"
kubectl apply -f manifests/local-dind/kube-flannel.yaml
kubectl apply -f manifests/local-dind/local-volume-provisioner.yaml
kubectl apply -f manifests/tiller-rbac.yaml
kubectl apply -f manifests/crd.yaml
kubectl create ns tidb-operator-e2e
helm init --service-account=tiller --wait

echo "############# success create cluster:[${clusterName}] #############"

echo "To start using your cluster, run:"
echo "    export KUBECONFIG=$(kind get kubeconfig-path --name=${clusterName})"
