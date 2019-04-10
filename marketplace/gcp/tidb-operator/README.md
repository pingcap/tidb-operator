# GKE service account

If you install on GKE, please select the option to create a new service account.
The default service account may not have the required permissions.

# Deletion

In the below, tidb is installed into the namespace "tidb".
Deleting the application will not delete the pods. You should delete the namespace and then delete the persistent volumes.

```
kubectl delete namespace tidb
kubectl get pv -l app.kubernetes.io/namespace=$NAMESPACE -o name | xargs -I {} kubectl patch {} -p '{"spec":{"persistentVolumeReclaimPolicy":"Delete"}}'
```


# Manual installation

First you can modify configuration values.

* schema.yaml: don't modify this, use parameters to override it as shown below
* chart/tidb-mp/values.yaml:
  * Note that you can override any tidb-operator or tidb-cluster configuration value


``` bash
# Install the k8s application CRD into your cluster
kubectl apply -f manifests/app-crd.yaml

VERSION='2.0'
PROJECT=$(gcloud config get-value project | tr ':' '/')
REGISTRY="gcr.io/$PROJECT/tidb-operator"

docker build \
  --build-arg "REGISTRY=$REGISTRY" \
  --build-arg "TAG=$VERSION" \
  --tag "$REGISTRY/deployer:$VERSION" .

gcloud docker -- push "$REGISTRY/deployer:$VERSION"

NAMESPACE=tidb
# We strongly recommend deploying into a new namespace
kubectl create namespace $NAMESPACE

REGISTRY=$REGISTRY NAMESPACE=$NAMESPACE VERSION=$VERSION ./scripts/install
```

You can watch the deployment come up with

```
kubectl get pods -n tidb --watch
```

When the tidb containers are running, you can connect with a MySQL client.

``` bash
kubectl -n $NAMESPACE port-forward db-tidb-0 4000:4000 &
mysql -u root -P 4000 -h 127.0.0.1
```
