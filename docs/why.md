# Why we need a new TiDB Operator

From its initial design through multiple iterations, TiDB Operator(v1) has faced and resolved many challenges. However, some issues remain inadequately addressed and have grown more pressing as TiDB and Kubernetes have evolved. These unresolved problems hinder the TiDB Operator(v1)'s maintenance and development.

## Issues

### StatefulSet is not flexible enough

StatefulSet is Kubernetes' native abstraction for managing stateful applications. Compared to Deployment, it provides critical features such as stable network identities and per-Pod PVC management, which are essential for stateful workloads. Consequently, TiDB Operator(v1) initially adopted StatefulSet as the foundational resource for managing TiDB clusters. However, it has become evident that StatefulSet has significant limitations as a low-level abstraction for effectively managing TiDB.

StatefulSet presents three significant limitations:
- Cannot scale in a specific Pod by its index.
- Lack of native support for pre/post hooks during scale-in or scale-out operations.
- Inability to modify the volume template after creation.

In a Deployment, deleting a Pod triggers the creation of a new Pod to replace the old one. However, in a StatefulSet, deleting a Pod means the Pod being "restarted" and no new Pod with different identity will be created. This behavior is useful for typical stateful applications, as their state is tied to the Pod's name. Changing the Pod's identity often involves complex operations, such as a raft membership change.

However, for TiKV, both "restart" and "replace" operations are required to effectively manage the cluster. It's because the TiKV is designed like "Cattle" more than "Pet". We hope TiKVs can be easily scaled in/out and replaced if one of them is "sick". In other words, we hope TiKVs:

- have stable network identity and per-Pod PVC just like StatefulSets
- can "restart" a specified Pod like StatefulSets
- can "replace" a specified Pod like Deployments

Unfortunately, it's very hard to "replace" a specific Pod in a StatefulSet because the index of Pod cannot be specified when scaling in. Try to resolve this problem, we introduce a forked version of StatefulSet called AdvancedStatefulSet. AdvancedStatefulSet can partially address this issue, but the solution is not particularly elegant. For example, we have to specify indexies of Pods that we want to delete in annotations of TidbCluster forever.

Another limitation of (Advanced)StatefulSet is the lack of native support for pre/post hooks. (Advanced)StatefulSet is designed as a declarative API, where users typically specify the desired state directly. However, certain additional tasks, such as leader eviction and raft membership changes, must be performed before or after scaling operations. To handle this, we modify the [partitions](https://kubernetes.io/docs/concepts/workloads/controllers/statefulset/#partitions) field each time a Pod is deleted or created to control the scale-in/scale-out process. This workaround deviates from the intended purpose of a declarative API.

The inability to modify the volume template in StatefulSet is also annoying. To resize a volume, we are forced to delete and re-create the StatefulSet, which is very hacky.

### TidbCluster is tooooo large

TidbCluster is an all-in-one CRD. This design was originally intended to simplify TiDB deployment and make it easy to understand and use by users. By consolidating all TiDB cluster components into a single CRD, users only needed to focus on and inspect the information and structure of that CRD.

However, as the TiDB Operator(v1) evolved, more fields and components were added to the all-in-one CRD (TidbCluster). To date, it encompasses **eight** components in total. This has caused the TidbCluster CRD to deviate entirely from its original design goals, instead significantly increasing the cognitive burden on users when understanding and using it.

Additionally, the all-in-one CRD currently stores the state information for all components. As cluster size grows, using `kubectl` to view the state of a specific component or instance becomes increasingly challenging. This not only undermines usability but can also lead to performance issues.

And the implementation for heterogeneous clusters is also weird because of the all-in-one TidbCluster design. Two TidbCluster CRs will be created for heterogeneous cluster but one of them may only contains a sepcific component.

### Too many unmanaged kubernetes fields

Kubernetes provides a rich set of capabilities for running Pods, most of which are defined in a structure called `PodTemplate`. As Kubernetes evolves, more fields are continuously added to `PodTemplate`. However, the TiDB Operator(v1) does not handle these fields from `PodTemplate` in a unified way. Instead, each new field is added to the TidbCluster on a case-by-case basis.

Take `securityContext` as an example. While the TiDB Operator itself does not directly interact with the user's security configurations, `securityContext` is essential for almost all users running TiDB on Kubernetes.

All of these fields may be reformatted in the TidbCluster CRD without any actual changes to their functionality. This leads to the content of TidbCluster diverging significantly from the original `PodTemplate`, ultimately reducing usability. Even Kubernetes experts need to manually verify whether the TiDB Operator supports some specific fields when deploying TiDB.

Additionally, this approach prevents the TiDB Operator from quickly supporting new Kubernetes features. As a result, many practical and valuable Kubernetes functionalities remain unsupported by the current TiDB Operator.

### Missing validation

Another issue is the need for a validation webhook enhance the user experience when managing TiDB clusters with the TiDB Operator in Kubernetes.

### Compatibility with Kubernetes and TiDB

Both Kubernetes and TiDB evolve rapidly. A more robust design is required to maintain compatibility with the continuous updates and changes in both Kubernetes and TiDB.

## Future

### Autoscaling

Autoscaling is a valuable but complex feature for TiDB. We aim to explore practical scenarios where it can address challenges that are difficult to resolve in traditional IDC environments.

### Kubectl plugin

We hope to provide a similar user experience like [tiup](https://github.com/pingcap/tiup) by a kubectl plugin.
