# Table of Contents
1. [K8s Pre-reqs](#k8s-pre-reqs)
2. [Install TiDB on K8s with network storage](#Install-TiDB-on-K8s-with-network-storage)
3. [Install TiDB on K8s with local storage](#Install-TiDB-on-K8s-with-local-storage)


# K8s Pre-reqs

Follow TiDB K8s Pre-reqs [documentation](https://docs.pingcap.com/tidb-in-kubernetes/stable/deploy-on-aws-eks/#prerequisites](https://docs.pingcap.com/tidb-in-kubernetes/stable/prerequisites/)).

# Install TiDB on K8s with network storage
## NOT FINISHED - DO NOT USE
1. Create Namespaces

```
kubectl create namespace tidb-cluster
kubectl create namespace tidb-admin
```

2. Install CRDs for TiDB Operator

```
kubectl create -f https://raw.githubusercontent.com/pingcap/tidb-operator/v1.6.5/manifests/crd.yaml
```

3. Install TiDB Operator via Helm

```
helm repo add pingcap https://charts.pingcap.org/
helm install --namespace tidb-admin tidb-operator pingcap/tidb-operator --version v1.6.5
```

4. Deploy TiDB Cluster and Monitoring Service

```
kubectl apply -f tidb-cluster.yaml -n tidb-cluster && \
kubectl apply -f tidb-monitor.yaml -n tidb-cluster
```
# Install TiDB on K8s with local storage
## NOT FINISHED - DO NOT USE
1. Deploy the Local Volume Provisioner

```
kubectl apply -f local-volume-provisioner.yaml
```

2. Create Namespaces

```
kubectl create namespace tidb-cluster
kubectl create namespace tidb-admin
```

3. Install CRDs for TiDB Operator

```
kubectl create -f https://raw.githubusercontent.com/pingcap/tidb-operator/v1.6.5/manifests/crd.yaml
```

4. Install TiDB Operator via Helm

```
helm repo add pingcap https://charts.pingcap.org/
helm install --namespace tidb-admin tidb-operator pingcap/tidb-operator --version v1.6.5
```

5. Deploy TiDB Cluster and Monitoring Service

```
kubectl apply -f ssd-tidb-cluster.yaml -n tidb-cluster && \
kubectl apply -f tidb-monitor.yaml -n tidb-cluster
```
