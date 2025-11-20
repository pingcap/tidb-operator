# Table of Contents
1. [EKS Pre-reqs](#eks-pre-reqs)
2. [EKS with EBS](#eks-with-ebs)
3. [EKS with SSD](#eks-with-ssd)



# EKS Pre-reqs

Following TiDB EKS Pre-reqs [documentation](https://docs.pingcap.com/tidb-in-kubernetes/stable/deploy-on-aws-eks/#prerequisites)

# EKS with EBS

1. Create EKS Cluster ([AWS Docs](https://docs.aws.amazon.com/eks/latest/userguide/create-cluster.html))

Cluster Add-Ons
* CoreDNS
* kube-proxy
* Amazon VPC CNI
* Amazon EKS EBS CSI Driver

2. (Optional) Login with AWS CLI

```
aws sso login --profile <<USER_PROFILE>>
```

3. Update Kubeconfig

```
aws eks update-kubeconfig --region us-east-1 --name <<CLUSTER_NAME>> --profile <<USER_PROFILE>>
```

4. Create Nodegroup

```
eksctl create nodegroup -f cluster.yaml --profile DBaaS-DevUser-Role-219248915861
```

5. Create Namespaces

```
kubectl create namespace tidb-cluster
kubectl create namespace tidb-admin
```

6. Install CRDs for TiDB Operator

```
kubectl create -f https://raw.githubusercontent.com/pingcap/tidb-operator/v1.6.4/manifests/crd.yaml
```

7. Install TiDB Operator via Helm

```
helm repo add pingcap https://charts.pingcap.org/
helm install --namespace tidb-admin tidb-operator pingcap/tidb-operator --version v1.6.4
```

8. Deploy TiDB Cluster and Monitoring Service

```
kubectl apply -f tidb-cluster.yaml -n tidb-cluster && \
kubectl apply -f tidb-monitor.yaml -n tidb-cluster
```

# EKS with SSD

1. Create EKS Cluster ([AWS Docs](https://docs.aws.amazon.com/eks/latest/userguide/create-cluster.html))

Cluster Add-Ons
* CoreDNS
* kube-proxy
* Amazon VPC CNI

2. (Optional) Login with AWS CLI

```
aws sso login --profile <<USER_PROFILE>>
```

3. Update Kubeconfig

```
aws eks update-kubeconfig --region us-east-1 --name <<CLUSTER_NAME>> --profile <<USER_PROFILE>>
```

4. Create Nodegroup

```
eksctl create nodegroup -f ssd-cluster.yaml --profile DBaaS-DevUser-Role-219248915861
```

5. Deploy the Local Volume Provisioner

```
kubectl apply -f local-volume-provisioner.yaml
```

6. Create Namespaces

```
kubectl create namespace tidb-cluster
kubectl create namespace tidb-admin
```

7. Install CRDs for TiDB Operator

```
kubectl create -f https://raw.githubusercontent.com/pingcap/tidb-operator/v1.6.4/manifests/crd.yaml
```

8. Install TiDB Operator via Helm

```
helm repo add pingcap https://charts.pingcap.org/
helm install --namespace tidb-admin tidb-operator pingcap/tidb-operator --version v1.6.4
```

9. Deploy TiDB Cluster and Monitoring Service

```
kubectl apply -f ssd-tidb-cluster.yaml -n tidb-cluster && \
kubectl apply -f tidb-monitor.yaml -n tidb-cluster
```
