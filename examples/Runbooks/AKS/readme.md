# Table of Contents
1. [AKS Pre-reqs](#aks-pre-reqs)
2. [AKS with Managed Disk](#aks-with-managed-disk)

# AKS Pre-reqs

1. Following TiDB AKS Pre-reqs [documentation](https://docs.pingcap.com/tidb-in-kubernetes/stable/deploy-on-azure-aks/#prerequisites)

```
az login
az provider register --namespace Microsoft.ContainerService
az extension add -n k8s-extension
```

# AKS with Managed Disk

1. Create AKS Cluster and Node Pool

```
# create AKS cluster
az aks create \
    --resource-group <<RESOURCE_GROUP_NAME>> \
    --name <<CLUSTER_NAME>> \
    --location <<LOCATION>> \
    --generate-ssh-keys \
    --vm-set-type VirtualMachineScaleSets \
    --node-vm-size Standard_D2s_v3 \
    --load-balancer-sku standard \
    --node-count 3 \
    --zones 1 2 3 \
    --enable-ultra-ssd
```
ex)
```
az aks create \
    --resource-group test \
    --name test \
    --location "East US 2" \
    --generate-ssh-keys \
    --vm-set-type VirtualMachineScaleSets \
    --load-balancer-sku standard \
    --node-count 3 \
    --zones 2 3
    --enable-ultra-ssd
```

2. Create component node pools

```
az aks nodepool add --name admin \
    --cluster-name ${clusterName} \
    --resource-group ${resourceGroup} \
    --zones 1 2 3 \
    --node-count 1 \
    --labels dedicated=admin

az aks nodepool add --name pd \
    --cluster-name ${clusterName} \
    --resource-group ${resourceGroup} \
    --node-vm-size ${nodeType} \
    --zones 1 2 3 \
    --node-count 3 \
    --labels dedicated=pd \
    --node-taints dedicated=pd:NoSchedule

az aks nodepool add --name tidb \
    --cluster-name ${clusterName} \
    --resource-group ${resourceGroup} \
    --node-vm-size ${nodeType} \
    --zones 1 2 3 \
    --node-count 2 \
    --labels dedicated=tidb \
    --node-taints dedicated=tidb:NoSchedule

az aks nodepool add --name tikv \
    --cluster-name ${clusterName} \
    --resource-group ${resourceGroup} \
    --node-vm-size ${nodeType} \
    --zones 1 2 3 \
    --node-count 3 \
    --labels dedicated=tikv \
    --node-taints dedicated=tikv:NoSchedule \
    --node-vm-size Standard_D2s_v3 \
    --enable-ultra-ssd
```

ex)

```
az aks nodepool add --name admin \
    --cluster-name test \
    --resource-group test \
    --zones 2 3 \
    --node-count 1 \
    --labels dedicated=admin

az aks nodepool add --name pd \
    --cluster-name test \
    --resource-group test \
    --node-vm-size Standard_F4s_v2 \
    --zones 2 3 \
    --node-count 3 \
    --labels dedicated=pd \
    --node-taints dedicated=pd:NoSchedule

az aks nodepool add --name tidb \
    --cluster-name test \
    --resource-group test \
    --node-vm-size Standard_F8s_v2 \
    --zones 2 3 \
    --node-count 2 \
    --labels dedicated=tidb \
    --node-taints dedicated=tidb:NoSchedule

az aks nodepool add --name tikv \
    --cluster-name test \
    --resource-group test \
    --node-vm-size Standard_E8s_v4 \
    --zones 2 3 \
    --node-count 3 \
    --labels dedicated=tikv \
    --node-taints dedicated=tikv:NoSchedule \
    --enable-ultra-ssd
```

3. Update Kubeconfig
```
az aks get-credentials --name <<NAME>> --resource-group <<RESOURCE_GROUP>>
```


4. Create Namespaces

```
kubectl create namespace tidb-cluster
kubectl create namespace tidb-admin
```

4. Install CRDs for TiDB Operator

```
kubectl create -f https://raw.githubusercontent.com/pingcap/tidb-operator/v1.6.5/manifests/crd.yaml
```

5. Install TiDB Operator via Helm

```
helm repo add pingcap https://charts.pingcap.org/
helm install --namespace tidb-admin tidb-operator pingcap/tidb-operator --version v1.6.5
```

6. Deploy TiDB Cluster and Monitoring Service

```
kubectl apply -f md-sc.yaml
kubectl apply -f md-tidb-cluster.yaml -n tidb-cluster && \
kubectl apply -f tidb-monitor.yaml -n tidb-cluster
```
