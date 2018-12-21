# Multiple statefulsets support

## Background

Now, TiDB Operator creates only one statefulset for one PD/TiKV/TiDB component separately. This works very well in most cases.

But it can't meet or hardly meet these requests:

* When using local volume, only horizontal scaling is guaranteed, vertical scaling may fail if there are not enough resources on the node. We need vertical scaling anyway
* Users can't specify an exact number of pods in a given AZ. For example deploy 3 TiKV in AZ-1, and 2 TiKV in AZ-2
* For on-premise deployment, k8s nodes may not have the same size. To make most of the resources, it's better to allow different profiles for a single cluster.
* No isolation between different tidb-servers, a big AP query may affect TP query. The worst case is that AP query cause a tidb-server OOM, and this tidb-server is also serving TP request

However, we can achieve all the above issues by using multiple statefulsets:

* Add a new statefulset with bigger resources to scale TiDB cluster vertically
* Different statefulsets can use different AZs or different resources and any other attributes

So this proposal suggests extending one statefulset with multiple statefulsets for a single component.

## Proposal

### API change

Users can create multiple statefulsets in `charts/tidb-cluster/values.yaml`, for example: change one tikv(Figure 1) to more tikvs(Figure 2):

Figure 1:

``` yaml
tikv:
  replicas: 3
  image: pingcap/tikv:v2.1.0
  …
```

Figure 2:

``` yaml
tikv:
  replicas: 3
  image: pingcap/tikv:v2.1.0
  …
  extra:
  - name: name-1
    replicas: 2
    …
  - name: name-2
    replicas: 2
    …
```

The default values.yaml is the same as of now. We can add document about how to extend the values.yaml to include the `extra` part listed here. But only a name is required, other fields can inherit from the upper level. And the generated `TidbCluster` object includes all the fields. And each statefulset has a separate ConfigMap named by the statefulset name, for example `name-1-tikv`. Also we can name the upper level statefulset as `default` so we can handle the names uniformly.

We must add a name attribute(for example: name-1 or name-2) to the section to distinguish the different statefulsets. The name of the default statefulset is `default`.

The same as TidbCluster object, from Figure 3 to Figure 4:

Figure 3:

``` go
type TidbClusterSpec struct {
  …
  TiKV            TiKVSpec            `json:"tikv,omitempty"`
  …
}

type TiKVStatus struct {
StatefulSet     *apps.StatefulSetStatus     `json:"statefulSet,omitempty"`
  …
}
```

Figure 4:

``` go
type TidbClusterSpec struct {
  TiKV            []TiKVSpec            `json:"tikv,omitempty"`
  …
}

type TiKVSpec struct {
  Name            string           `json:"name"`
  Replicas        int32            `json:"replicas"`
  …
}

type TiKVStatus struct {
StatefulSets     []apps.StatefulSetStatus     `json:"statefulSet,omitempty"`
  …
}
```

### Create:

The statefulset name will have a name suffix(Figure 2), for example: `<cluserName>-tikv-<name-suffix>` to distinguish different statefulsets.

The TiDB Operator should maintain multiple statefulset objects in the kube apiserver. The startup scripts are the same as before basically. An exception is PD startup script: the bootstrapping Pod may be the first statefulset first Pod.

The TiDB Operator will create extra Services together with multiple statefulsets:

* For PD: no change
* For TiKV: no change
* For TiDB: N+1 Services should be created, N client services and 1 main client service

### Upgrade:

With multiple statefulsets supported, the user can modify the spec in the `values.yaml` then the Operator will range over all the statefulsets and select only one statefulset to upgrade at a time. So the basic progress is the same with older design.

The user can modify all the statefulsets, but we only support upgrade one Pod at a time.

For example, the user can change the images from v2.1.0(Figure 5) to v2.2.0(Figure 6) to upgrade the cluster from v2.1.0 to v2.2.0:

Figure 5:

``` yaml
tikv:
  replicas: 3
  image: pingcap/tikv:v2.1.0
  …
  extra:
  - name: name-1
    replicas: 3
    …
  - name: name-2
    replicas: 2
    …
```

Figure 6:

``` yaml
tikv:
  replicas: 3
  image: pingcap/tikv:v2.2.0
  …
  extra:
  - name: name-1
    replicas: 3
    …
  - name: name-2
    replicas: 2
    …
```

The operator does not guarantee the order of multiple statefulsets’ upgrade. If the user care about the order indeed. Just: modify one statefulset, wait it done, and then modify the other one.

### Scale In/Out:

Like the Upgrade section, the user can sale in/out all the statefulsets at the same time, operator will only scale in/out one Pod at a time.
The user can just modify one statefulset if they want to operate the special statefulset.

For example, the user can change the replicas in region-1 from 3(Figure 7) to 5(Figure 8) to add more replicas in the region-1:

Figure 7:

``` yaml
tikv:
  replicas: 3
  image: pingcap/tikv:v2.1.0
  …
  extra:
  - name: name-1
    replicas: 3
    …
  - name: name-2
    replicas: 2
    …
```

Figure 8:

``` yaml
tikv:
  replicas: 3
  image: pingcap/tikv:v2.1.0
  …
  extra:
  - name: name-1
    replicas: 5
    …
  - name: name-2
    replicas: 2
    …
```

The operator does not guarantee the order of multiple statefulsets’ scale in/out. If the user care the order indeed. Just: modify one statefulset, wait it done, and then modify the other one.

### Failover:

When a PD/TiKV/TiDB peer is failure, the failover module will add a new peer to the failed statefulset. Others are the same with older design.

## Overall:

The unit tests and e2e tests will have many changes when we add multiple statefulsets feature to TiDB Operator.

It is a break change, so this feature may be added before the TiDB Operator GA.
