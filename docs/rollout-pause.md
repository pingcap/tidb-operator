# Pause a Group rollout

Set `spec.progressing: false` on a kernel component Group to stop the operator
from issuing further instance updates, scale-out, scale-in, or deferred deletion
cleanup for that Group. Existing instance controllers continue reconciling: a
Pod already selected for restart can be recreated, recover readiness, and finish
TiKV leader-eviction cleanup.

This setting is supported by PDGroup, TiKVGroup, TiDBGroup, TiFlashGroup,
TiKVWorkerGroup, TiCDCGroup, TiProxyGroup, TSOGroup, SchedulingGroup, SchedulerGroup,
RouterGroup, and ResourceManagerGroup. It defaults to true; omitting the field
preserves normal progression.

```sh
kubectl --context <context> -n <namespace> patch tikvgroups.core.pingcap.com <group> \
  --type=merge -p '{"spec":{"progressing":false}}'
```

To resume, set the same field to true. The operator continues toward the latest
desired template and replica count, including changes made while paused.

```sh
kubectl --context <context> -n <namespace> patch tikvgroups.core.pingcap.com <group> \
  --type=merge -p '{"spec":{"progressing":true}}'
```

The setting affects only the selected Group. `Cluster.spec.paused` takes
precedence: when true, it stops Group and instance reconciliation regardless of
`progressing`. Keep it false to allow ongoing restarts to finish. Changing
`progressing` does not change the instance revision or initiate a restart.

Pause takes effect when a Group reconciliation observes the setting. Operations
already issued, or issued by a reconciliation that has passed the pause check,
are not canceled. Multiple instances may already be updating. There is no
completion acknowledgement; Group and instance status continue updating, but
setting the flag alone does not mean all ongoing restarts have finished.

For components that create surge instances, both old and replacement instances
may remain while paused. Deferred cleanup resumes with the rollout. Mixed
revisions keep a Group unsynced, and extra replicas can keep Group Ready false
even when individual instances are ready.

Replica-count changes, including HPA requests, are delayed until resume. An
existing instance can recreate its missing Pod, but a deleted instance CR will
not be replaced by the Group updater while paused. Group deletion, Cluster
suspension, and the separate Cluster-wide pause retain their existing behavior.
