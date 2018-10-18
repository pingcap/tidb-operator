# TODO

* agent secret
* storage class

First you can modify configuration values.

* schema.yaml: don't modify this, use parameters to override it as shown below
* chart/tidb-mp/values.yaml:
  * Note that you can override any tidb-operator or tidb-cluster configuration value

TODO: review this once we have a Marketplace image published. In particular, set REPO in ./scripts/install

``` bash
# Install the k8s application CRD into your cluster
kubectl apply -f manifests/app-crd.yaml

export VERSION='1.0.0'
export PROJECT=${PROJECT:-$(gcloud config get-value project | tr ':' '/')}
export REGISTRY="gcr.io/$PROJECT"
export APP_NAME="tidb-operator-enterprise"

docker build \
  --build-arg "REGISTRY=$REGISTRY" \
  --build-arg "TAG=$VERSION" \
  --tag "$REGISTRY/$APP_NAME/deployer:$VERSION" .

gcloud docker -- push "$REGISTRY/$APP_NAME/deployer:$VERSION"

# We strongly recommend deploying into a new namespace
kubectl create namespace tidb
export NAMESPACE=tidb

./scripts/install
```

You can watch the deployment come up with

```
kubectl get pods -n tidb --watch
```

When the tidb containers are running, you can connect with a MySQL client.

``` bash
kubectl -n test-ns port-forward db-tidb-0 4000:4000 &
mysql -u root -P 4000 -h 127.0.0.1
```
