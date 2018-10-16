# TODO

* agent secret
* storage class

First you can modify configuration values.

* schema.yaml: don't modify this, use parameters. overrides the below
* chart/tidb-mp/values.yaml: these override the below
* chart/tidb-mp/charts/tidb-cluster/values.yaml
* chart/tidb-mp/charts/tidb-crd/values.yaml
* chart/tidb-mp/charts/tidb-operator/values.yaml

``` bash
export VERSION='v1.0.3'
export PROJECT=${PROJECT:-$(gcloud config get-value project | tr ':' '/')}
export REGISTRY="gcr.io/${PROJECT}"
export APP_NAME="tidb-operator-enterprise"

docker build \
  --build-arg "REGISTRY=$REGISTRY" \
  --build-arg "TAG=$VERSION" \
  --tag "$REGISTRY/$APP_NAME/deployer:$VERSION" .

gcloud docker -- push "$REGISTRY/$APP_NAME/deployer:$VERSION"

# We recommend deploying into a new namespace
kubectl create namespace tidb

PARAMETERS='{"name": "test-deployment", "namespace": "tidb", "tidb-cluster.monitor.ubbagent.reportingSecret":"secret", "tidb-cluster.pd.storageClassName":"pd-ssd", "tidb-cluster.tikv.storageClassName":"pd-ssd", "tidb-cluster.tidb.storageClassName":"pd-ssd"}'

# This uses a large docker image that takes a long time to download.
./scripts/mpdev /scripts/install \
  --deployer=$REGISTRY/$APP_NAME/deployer:$VERSION \
  --parameters="$PARAMETERS"
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
