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

According to K8s [api compatibility](https://github.com/kubernetes/community/blob/master/contributors/devel/api_changes.md#on-compatibility):

> An API change is considered compatible if it:
>
> - adds new functionality that is not required for correct behavior (e.g., does not add a new required field)
> - does not change existing semantics, including:
> - the semantic meaning of default values and behavior
> - interpretation of existing API types, fields, and values
> - which fields are required and which are not
> - mutable fields do not become immutable
> - valid values do not become invalid
> - explicitly invalid values do not become valid
>
> Put another way:
>
> - Any API call (e.g. a structure POSTed to a REST endpoint) that succeeded before your change must succeed after your change.
> - Any API call that does not use your change must behave the same as it did before your change.
> - Any API call that uses your change must not cause problems (e.g. crash or degrade behavior) when issued against an API servers that do not include your change.
> - It must be possible to round-trip your change (convert to different API versions and back) with no loss of information.
> - Existing clients need not be aware of your change in order for them to continue to function as they did previously, even when your change is in use.
> - It must be possible to rollback to a previous version of API server that does not include your change and have no impact on API objects which do not use your change. API objects that use your change will be impacted in case of a rollback.
> - If your change does not meet these criteria, it is not considered compatible, and may break older clients, or result in newer clients causing undefined behavior. Such changes are generally disallowed, though exceptions have been made in extreme cases (e.g. security or obvious bugs).

So add an extra field to existing may be a better solution.

For example: change `values.yaml` from figure 1 to figure 2, and change the TidbCluster object from figure 3 to figure 4:

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

tikvs:
  - name: name-1
    replicas: 2
    image: pingcap/tikv:v2.1.0
    …
  - name: name-2
    replicas: 2
    image: pingcap/tikv:v2.1.0
    …
```

The `tikv` section is the main tikv statefulset. `tikvs` is the extra tikv statefulsets. They must include all the attributes.

And each statefulset has a separate ConfigMap named by the statefulset name, the `tikv` statefulset name is: `<clusterName>-tikv`, the `tikvs` statefulset name are `<clusterName>-tikv-name-1` and `<clusterName>-tikv-name-2`.

We must add a name attribute(for example: name-1 or name-2) to the `tikvs` section to distinguish the different statefulsets.

The `tikvs` section can be empty.

The same as TidbCluster object, from Figure 3 to Figure 4:

Figure 3:

``` go
type TidbClusterSpec struct {
  TiKV            TiKVSpec            `json:"tikv,omitempty"`
  …
}

type TiKVSpec struct {
  Replicas        int32            `json:"replicas"`
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
  TiKV         TiKVSpec         `json:"tikv,omitempty"`
  TiKVs        []TiKVSpec       `json:"tikvs,omitempty"`
  …
}

type TiKVSpec struct {
  Name            string           `json:"name"`
  Replicas        int32            `json:"replicas"`
  …
}

type TiKVStatus struct {
  StatefulSet     *apps.StatefulSetStatus     `json:"statefulSet,omitempty"`
  StatefulSets    map[string]apps.StatefulSetStatus    `json:"statefulSets,omitempty"`
  …
}
```

### Create:

The TiDB Operator should maintain multiple statefulset objects in the kube apiserver. The startup scripts are the same as before basically. An exception is PD startup script: the bootstrapping Pod may be the main statefulset first Pod.

The TiDB Operator will create extra Services together with multiple statefulsets:

* For PD: 1 headless service and 1 client service should be created;
* For TiKV: 1 headless service should be created;
* For TiDB: 1 headless service, 1 main client service and N client services should be created, N represents the count of the statefulsets.

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

tikvs:
  - name: name-1
    replicas: 2
    image: pingcap/tikv:v2.1.0
    …
  - name: name-2
    replicas: 2
    image: pingcap/tikv:v2.1.0
    …
```

Figure 6:

``` yaml
tikv:
  replicas: 3
  image: pingcap/tikv:v2.2.0
  …

tikvs:
  - name: name-1
    replicas: 2
    image: pingcap/tikv:v2.2.0
    …
  - name: name-2
    replicas: 2
    image: pingcap/tikv:v2.2.0
    …
```

The operator upgrades the main statefulset first, and then upgrades the statefulset in the order of statefulset slice.

The operator should add validation to CRD using [`admission webhooks`](https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers/#admission-webhooks). For example, validation will fail if the `name` filed in `tikvs` are the same.

The operator does not allow modification of the `name` of statefulset, it should return an error to the user if the he is trying to change the `name` of the statefulset,

### Scale In/Out:

Like the Upgrade section, the user can sale in/out all the statefulsets at the same time, operator will only scale in/out one Pod at a time.
The user can just modify one statefulset if they want to operate the special statefulset.

For example, the user can change the replicas in name-1 from 3(Figure 7) to 5(Figure 8) to add more replicas in the name-1:

Figure 7:

``` yaml
tikv:
  replicas: 3
  image: pingcap/tikv:v2.1.0
  …

tikvs:
  - name: name-1
    replicas: 3
    image: pingcap/tikv:v2.1.0
    …
  - name: name-2
    replicas: 2
    image: pingcap/tikv:v2.1.0
    …
```

Figure 8:

``` yaml
tikv:
  replicas: 3
  image: pingcap/tikv:v2.1.0
  …

tikvs:
  - name: name-1
    replicas: 5
    image: pingcap/tikv:v2.1.0
    …
  - name: name-2
    replicas: 2
    image: pingcap/tikv:v2.1.0
    …
```

The operator scales in/out the main statefulset first, and then scales in/out the statefulset in the order of statefulset slice.

### Failover:

When a PD/TiKV/TiDB peer is failure, the failover module will add a new peer to the failed statefulset. Others are the same with older design.

### Vertical scaling

We can achieve vertical scaling in two phases.

#### Phase 1

We do not implement this in tidb-operator. But users can achieve this:

- Add a new statefulset in the `tikvs` section, using more `requests` or `limits` and wait it done. Add a new `name-3`, change from figure 9 to figure 10;
- Scale down the old statefulset(`name-1`) in the `tikvs` section to zero and wait it done. See figure 11;
- Remove the old statefulset(`name-1`) in the `tikvs` section. See figure 12.

Figure 9:

``` yaml
tikv:
  replicas: 3
  image: pingcap/tikv:v2.1.0
  …

tikvs:
  - name: name-1
    replicas: 5
    image: pingcap/tikv:v2.1.0
    resources:
      limits:
        cpu: 1000m
        memory: 1Gi
      requests:
        cpu: 1000m
        memory: 1Gi
        storage: 1Gi
    …
  - name: name-2
    replicas: 2
    image: pingcap/tikv:v2.1.0
    …
```


Figure 10:

``` yaml
tikv:
  replicas: 3
  image: pingcap/tikv:v2.1.0
  …

tikvs:
  - name: name-1
    replicas: 5
    image: pingcap/tikv:v2.1.0
    resources:
      limits:
        cpu: 1000m
        memory: 1Gi
      requests:
        cpu: 1000m
        memory: 1Gi
        storage: 1Gi
    …
  - name: name-2
    replicas: 2
    image: pingcap/tikv:v2.1.0
    …
  - name: name-3
    replicas: 5
    image: pingcap/tikv:v2.1.0
    resources:
      limits:
        cpu: 8000m
        memory: 8Gi
      requests:
        cpu: 8000m
        memory: 8Gi
        storage: 8Gi
    …
```


Figure 11:

``` yaml
tikv:
  replicas: 3
  image: pingcap/tikv:v2.1.0
  …

tikvs:
  - name: name-1
    replicas: 0
    image: pingcap/tikv:v2.1.0
    resources:
      limits:
        cpu: 1000m
        memory: 1Gi
      requests:
        cpu: 1000m
        memory: 1Gi
        storage: 1Gi
    …
  - name: name-2
    replicas: 2
    image: pingcap/tikv:v2.1.0
    …
  - name: name-3
    replicas: 5
    image: pingcap/tikv:v2.1.0
    resources:
      limits:
        cpu: 8000m
        memory: 8Gi
      requests:
        cpu: 8000m
        memory: 8Gi
        storage: 8Gi
    …
```


Figure 12:

``` yaml
tikv:
  replicas: 3
  image: pingcap/tikv:v2.1.0
  …

tikvs:
  - name: name-2
    replicas: 2
    image: pingcap/tikv:v2.1.0
    …
  - name: name-3
    replicas: 5
    image: pingcap/tikv:v2.1.0
    resources:
      limits:
        cpu: 8000m
        memory: 8Gi
      requests:
        cpu: 8000m
        memory: 8Gi
        storage: 8Gi
    …
```

#### Phase 2

When the user change the requests or limits or both of them, the operator creates a new statefulset and set its replicas to 0, then:

- Scale up the new statefulset one at a time, and scale down the old statefulset one at a time.
- Remove the old statefulset if the new statefulset reach the replicas.

### Multiple zones

We can use different `nodeSelector`s in different statefulsets to deploy tidb to multiple zones, figure 13:

Figure 13:

``` yaml
tikv:
  replicas: 3
  image: pingcap/tikv:v2.1.0
  nodeSelector:
    kind: tikv
    zone: cn-bj1-01,cn-bj1-02
  …

tikvs:
  - name: name-2
    replicas: 2
    image: pingcap/tikv:v2.1.0
    nodeSelector:
      kind: tikv
      zone: cn-bj1-03,cn-bj1-04
    …
  - name: name-3
    replicas: 5
    image: pingcap/tikv:v2.1.0
    resources:
      limits:
        cpu: 8000m
        memory: 8Gi
      requests:
        cpu: 8000m
        memory: 8Gi
        storage: 8Gi
    …
```

The main `tikv` will be deployed in `cn-bj1-01` and `cn-bj1-02` zones and the `name-2` in `tikvs` section will be deployed in `cn-bj1-03` and `cn-bj1-04` zones.
