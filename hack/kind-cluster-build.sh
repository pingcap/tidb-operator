#!/usr/bin/env bash
set -e

ROOT=$(unset CDPATH && cd $(dirname "${BASH_SOURCE[0]}")/.. && pwd)
cd $ROOT

usage() {
    cat <<EOF
This script use kind to create Kubernetes cluster,about kind please refer: https://kind.sigs.k8s.io/
Before run this script,please ensure that:
* have installed docker
* have installed kind and kind's version == v0.4.0

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
workDir=${HOME}/kind/${clusterName}
mkdir -p ${workDir}

data_dir=${workDir}/data

echo "clean data dir: ${data_dir}"
if [ -d ${data_dir} ]; then
    rm -rf ${data_dir}
fi

configFile=${workDir}/kind-config.yaml

cat <<EOF > ${configFile}
kind: Cluster
apiVersion: kind.sigs.k8s.io/v1alpha3
nodes:
- role: control-plane
  extraPortMappings:
  - containerPort: 5000
    hostPort: 5000
    listenAddress: 0.0.0.0
    protocol: TCP
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

echo "deploy docker registry in kind"
registryNode=${clusterName}-control-plane
registryNodeIP=$(kubectl get nodes ${registryNode} -o template --template='{{range.status.addresses}}{{if eq .type "InternalIP"}}{{.address}}{{end}}{{end}}')
registryFile=${workDir}/registry.yaml

cat <<EOF >${registryFile}
apiVersion: extensions/v1beta1
kind: DaemonSet
metadata:
  name: registry
spec:
  selector:
    matchLabels:
      app: registry
  template:
    metadata:
      name: registry
      labels:
        app: registry
    spec:
      hostNetwork: true
      nodeSelector:
        kubernetes.io/hostname: ${registryNode}
      tolerations:
      - key: node-role.kubernetes.io/master
        operator: "Equal"
        effect: "NoSchedule"
      containers:
      - name: registry
        image: registry:2
        volumeMounts:
        - name: data
          mountPath: /data
      volumes:
      - name: data
        hostPath:
          path: /data
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: registry-proxy-config
data:
  nginx-conf: |
    user  nginx;
    worker_processes  1;
    error_log  /var/log/nginx/error.log warn;
    pid        /var/run/nginx.pid;
    events {
        worker_connections  1024;
    }
    http {
        upstream docker-registry {
            server ${registryNodeIP}:5000;
        }
        map \$upstream_http_docker_distribution_api_version \$docker_distribution_api_version {
            '' 'registry/2.0';
        }
        server {
            listen 5000;
            server_name localhost;
            client_max_body_size 0;
            chunked_transfer_encoding on;
            location /v2/ {
                if (\$http_user_agent ~ "^(docker\/1\.(3|4|5(?!\.[0-9]-dev))|Go ).*$") {
                    return 404;
                }
                add_header 'Docker-Distribution-Api-Version' \$docker_distribution_api_version always;
                proxy_pass http://docker-registry;
                proxy_set_header Host \$http_host;
                proxy_set_header X-Real-IP \$remote_addr;
                proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
                proxy_set_header X-Forwarded-Proto \$scheme;
                proxy_read_timeout 900;
            }
        }
        default_type  application/octet-stream;
        log_format  main  '\$remote_addr - \$remote_user [\$time_local] "\$request" '
                          '\$status \$body_bytes_sent "\$http_referer" '
                          '"\$http_user_agent" "\$http_x_forwarded_for"';
        access_log  /var/log/nginx/access.log  main;
        sendfile        on;
        keepalive_timeout  65;
    }

---
apiVersion: extensions/v1beta1
kind: DaemonSet
metadata:
  name: registry-proxy
  labels:
    app: registry-proxy
spec:
  selector:
    matchLabels:
      app: registry-proxy
  template:
    metadata:
      labels:
        app: registry-proxy
    spec:
      hostNetwork: true
      affinity:
        nodeAffinity:
          requiredDuringSchedulingIgnoredDuringExecution:
            nodeSelectorTerms:
            - matchExpressions:
              - key: kubernetes.io/hostname
                operator: NotIn
                values:
                  - ${registryNode}
      tolerations:
      - key: node-role.kubernetes.io/master
        operator: "Equal"
        effect: "NoSchedule"
      containers:
        - name: nginx
          image: nginx:1.15-alpine
          volumeMounts:
          - mountPath: /etc/nginx
            name: nginx-config
      volumes:
      - name: nginx-config
        configMap:
          name: registry-proxy-config
          items:
            - key: nginx-conf
              path: nginx.conf
EOF
kubectl apply -f ${registryFile}

echo "init tidb-operator env"
kubectl apply -f ${ROOT}/manifests/local-dind/local-volume-provisioner.yaml
kubectl apply -f ${ROOT}/manifests/tiller-rbac.yaml
kubectl apply -f ${ROOT}/manifests/crd.yaml
kubectl create ns tidb-operator-e2e
helm init --service-account=tiller --wait

# This is required because current tidb-operator-e2e use a DIND-only image `mirantis/hypokube:final`
# FIXME: remove this
docker pull gcr.io/google-containers/kube-scheduler:${k8sVersion}
docker tag gcr.io/google-containers/kube-scheduler:${k8sVersion} mirantis/hypokube:final
kind load docker-image --name=${clusterName} mirantis/hypokube:final

echo "############# success create cluster:[${clusterName}] #############"

echo "To start using your cluster, run:"
echo "    export KUBECONFIG=$(kind get kubeconfig-path --name=${clusterName})"
echo ""
echo <<EOF
NOTE: In kind, nodes run docker network and cannot access host network.
If you configured local HTTP proxy in your docker, images may cannot be pulled
because http proxy is inaccessible.

If you cannot remove http proxy settings, you can either whitelist image
domains in NO_PROXY environment or use 'docker pull <image> && kind load
docker-image <image>' command to load images into nodes.
EOF
