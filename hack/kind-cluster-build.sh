#!/usr/bin/env bash
set -e

usage() {
    cat <<EOF
This script use kind to create Kubernetes cluster,about kind please refer: https://kind.sigs.k8s.io/
Before run this script,please ensure that:
* have installed git (version >= v2.15.1)
* have installed docker

Options:
       -h,--help               prints the usage message
       -n,--name               name of the Kubernetes cluster,default value: kind
       -c,--nodeNum            the count of the cluster nodes,default value: 6
       -k,--k8sVersion         version of the Kubernetes cluster,default value: v1.12.8
       -v,--volumeNum          the volumes number of each kubernetes node,default value: 9
Usage:
    $0 --name testCluster --nodeNum 4 --k8sVersion v1.12.9
EOF
}

while [[ $# -gt 0 ]]
do
key="$1"

case $key in
    -n|--name)
    clusterName="$2"
    shift
    shift
    ;;
    -c|--nodeNum)
    nodeNum="$2"
    shift
    shift
    ;;
    -k|--k8sVersion)
    k8sVersion="$2"
    shift
    shift
    ;;
    -v|--volumeNum)
    volumeNum="$2"
    shift
    shift
    ;;
    -h|--help)
    usage
    exit 0
    ;;
    *)
    echo "unknown option: $key"
    usage
    exit 1
    ;;
esac
done

clusterName=${clusterName:-kind}
nodeNum=${nodeNum:-6}
k8sVersion=${k8sVersion:-v1.12.8}
volumeNum=${volumeNum:-9}

echo "clusterName: ${clusterName}"
echo "nodeNum: ${nodeNum}"
echo "k8sVersion: ${k8sVersion}"
echo "volumeNum: ${volumeNum}"

# check requirements
for requirement in kind kubectl helm docker
do
    echo "############ check ${requirement} ##############"
    if hash ${requirement} 2>/dev/null;then
        echo "${requirement} have installed"
    else
        echo "this script needs ${requirement}, please install ${requirement} first."
        exit 1
    fi
done

echo "############# start create cluster:[${clusterName}] #############"

data_dir=${HOME}/kind/${clusterName}/data

echo "clean data dir: ${data_dir}"
if [ -d ${data_dir} ]; then
    rm -rf ${data_dir}
fi

configFile=${HOME}/kind/${clusterName}/kind-config.yaml

cat <<EOF > ${configFile}
kind: Cluster
apiVersion: kind.sigs.k8s.io/v1alpha3
networking:
  disableDefaultCNI: true
nodes:
- role: control-plane
EOF

for ((i=0;i<${nodeNum};i++))
do
mkdir -p ${data_dir}/worker${i}
cat <<EOF >>  ${configFile}
- role: worker
  extraMounts:
EOF
    for ((k=1;k<=${volumeNum};k++))
    do
        mkdir -p ${data_dir}/worker${i}/vol${k}
        cat <<EOF >> ${configFile}
  - containerPath: /mnt/disks/vol${k}
    hostPath: ${data_dir}/worker${i}/vol${k}
EOF
    done
done

echo "start to create k8s cluster"
kind create cluster --config ${configFile} --image kindest/node:${k8sVersion} --name=${clusterName}
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
