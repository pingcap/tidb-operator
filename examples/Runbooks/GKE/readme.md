# Table of Contents
1. [GKE Pre-reqs](#gke-pre-reqs)
2. [GKE with Persistnet Disk](#eks-with-persistent-disk)
3. [GKE with SSD](#gke-with-ssd)

# GKE Pre-reqs

1. Following TiDB GKE Pre-reqs [documentation](https://docs.pingcap.com/tidb-in-kubernetes/stable/deploy-on-gcp-gke/#prerequisites)
2. Configure [Google Cloud Service](https://docs.pingcap.com/tidb-in-kubernetes/stable/deploy-on-gcp-gke/#configure-the-google-cloud-service)

```
gcloud auth login
gcloud config set core/project <<PROJECT>>
gcloud config set compute/region <<REGION>>
```

# GKE with Persistent Disk

1. Create GKE Cluster and Node Pool

```
gcloud container clusters create tidb --region us-east1 --machine-type n1-standard-4 --num-nodes=1
```

2. Create separate node pools for PD, TiKV, and TiDB

```
gcloud container node-pools create pd --cluster tidb --machine-type n2-standard-4 --num-nodes=1 \
    --node-labels=dedicated=pd --node-taints=dedicated=pd:NoSchedule
gcloud container node-pools create tikv --cluster tidb --machine-type n2-highmem-8 --num-nodes=1 \
    --node-labels=dedicated=tikv --node-taints=dedicated=tikv:NoSchedule
gcloud container node-pools create tidb --cluster tidb --machine-type n2-standard-8 --num-nodes=1 \
    --node-labels=dedicated=tidb --node-taints=dedicated=tidb:NoSchedule
```

3. Create Namespaces

```
kubectl create namespace tidb-cluster
kubectl create namespace tidb-admin
```

4. Install CRDs for TiDB Operator

```
kubectl create -f https://raw.githubusercontent.com/pingcap/tidb-operator/v1.6.2/manifests/crd.yaml
```

5. Install TiDB Operator via Helm

```
helm repo add pingcap https://charts.pingcap.org/
helm install --namespace tidb-admin tidb-operator pingcap/tidb-operator --version v1.6.2
```

6. Deploy TiDB Cluster and Monitoring Service

```
kubectl apply -f pd-sc.yaml
kubectl apply -f pd-tidb-cluster.yaml -n tidb-cluster && \
kubectl apply -f tidb-monitor.yaml -n tidb-cluster
```

# GKE with SSD

1. Create GKE Cluster and Node Pool

```
gcloud container clusters create tidb --region us-east1 --machine-type n1-standard-4 --num-nodes=1
```

2. Create separate node pools for PD, TiKV, and TiDB

```
gcloud container node-pools create pd --cluster tidb --machine-type n2-standard-4 --num-nodes=1 \
    --node-labels=dedicated=pd --node-taints=dedicated=pd:NoSchedule
gcloud container node-pools create tikv --cluster tidb --machine-type n2-highmem-8 --num-nodes=1 --local-ssd-count 1 \
  --node-labels dedicated=tikv --node-taints dedicated=tikv:NoSchedule
gcloud container node-pools create tidb --cluster tidb --machine-type n2-standard-8 --num-nodes=1 \
    --node-labels=dedicated=tidb --node-taints=dedicated=tidb:NoSchedule
```

3. Deploy the Local Volume Provisioner

```
kubectl apply -f local-ssd-provisioner.yaml
```

4. Create Namespaces

```
kubectl create namespace tidb-cluster
kubectl create namespace tidb-admin
```

5. Install CRDs for TiDB Operator

```
kubectl create -f https://raw.githubusercontent.com/pingcap/tidb-operator/v1.6.2/manifests/crd.yaml
```

6. Install TiDB Operator via Helm

```
helm repo add pingcap https://charts.pingcap.org/
helm install --namespace tidb-admin tidb-operator pingcap/tidb-operator --version v1.6.2
```

7. Deploy TiDB Cluster and Monitoring Service

```
kubectl apply -f localdisk-tidb-cluster.yaml -n tidb-cluster && \
kubectl apply -f tidb-monitor.yaml -n tidb-cluster
```
