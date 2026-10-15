# Group scale-in selection

All v2 component Group controllers use the shared updater to select instances
when scaling in. The executor first determines the eligible updated or outdated
instances. Within that candidate set, preferences run in this order:

1. Component-specific policies, including topology spreading.
2. Observed volume capacity exceeding the requested capacity.
3. Instances whose Pods are not running.
4. Instances that are not up to date or not ready.
5. The instance CR's `core.pingcap.com/priority` annotation (smaller non-negative
   integers first; unset or invalid values have the lowest priority).
6. The actual Pod's `controller.kubernetes.io/pod-deletion-cost` annotation
   (smaller signed 32-bit integers first; unset or invalid values count as zero).
7. Newer instance creation timestamps.
8. The first remaining candidate.

Each preference narrows the remaining candidates. A preference that finds no
preferred candidates leaves the set unchanged. These are best-effort preferences;
a low deletion cost cannot override topology, instance health, manual priority,
availability limits, or the component's deletion/offlining procedure.

For example, to prefer a particular TiDB Pod during the next scale-in:

```sh
kubectl -n <namespace> annotate pod <pod-name> \
  controller.kubernetes.io/pod-deletion-cost="-10" --overwrite
kubectl -n <namespace> scale tidbgroup <group-name> --replicas=<replicas>
```

Only annotations on the actual managed Pod are read. Placing this annotation on
the Group or instance CR metadata has no effect. Negative costs are allowed;
positive costs prefer retaining the Pod relative to Pods without the annotation.
The supported range is `-2147483648` to `2147483647`. Values with a leading `+`,
positive leading zeros, or invalid/out-of-range integers count as zero, matching
[Kubernetes deletion-cost parsing](https://github.com/kubernetes/kubernetes/blob/v1.32.0/pkg/apis/core/helper/helpers.go).

A missing Pod has cost zero. Pod lookup failures and ambiguous or mismatched
ownership stop that scale-in attempt so reconciliation can retry. Pod costs are
read when choosing among multiple instances and reused within that updater
execution; subsequent reconciliations read them again. Both updated and outdated
scale-in candidates use this preference. In-place update selection and cancellation
of offlining retain their existing preferences.
