# Preserving alert rules during monitor image upgrades

The Operator keeps the effective alert rules persistence version independent of
the initializer image tag. Upgrading initializer or reloader images does not
implicitly move saved rules into a different version directory.

When `spec.alertManagerRulesVersion` is unset:

- For an existing monitor, the Operator recovers the effective
  `--sub-store-path` from its owned StatefulSets or legacy Deployment.
- For a new monitor, it initially uses `tidb:<spec.initializer.version>`.
- Before changing a workload, it records this choice in the controller-owned
  `tidb.pingcap.com/monitor-rules-version` annotation on the TidbMonitor.

Both TiDB and DM initialization and the reloader use the same recorded value.
It survives Pod and StatefulSet replacement while the TidbMonitor remains.
Do not remove or manage this annotation as an image-version setting.

To select a different rule version intentionally, set the existing field:

```yaml
spec:
  alertManagerRulesVersion: v8.5.7
```

Do not include the `tidb:` prefix in the field. An explicit value takes priority
and becomes the recorded version. Removing the field afterwards retains that
last selected version; it does not resume following initializer image tags.
Changing this field selects another persistence directory, not a rule-content
merge or automatic migration of files from the previous directory.

Upgrade the Operator before changing initializer image tags. Existing workloads
must still be available for the first reconciliation to recover their version.
If shard versions disagree, custom commands hide the effective version, or
previous workload status exists but no workload remains, reconciliation stops
and asks for an explicit `alertManagerRulesVersion` rather than guessing.
API read or annotation write failures also prevent workload rollout.

If an older Operator already rolled out a new directory, explicitly select the
original rules version to recover those saved rules. Likewise, when recreating
the TidbMonitor itself while retaining a PVC, restore the recorded annotation
or explicitly select the old version: the controller cannot infer a lost rule
version from the PVC's contents. Keep backups of custom rules during upgrades.

This directory compatibility does not replace image-specific migration steps.
In particular, a non-root reloader must be paired with an initializer that
prepares its active and persistent rule directories for the runtime UID/GID.
