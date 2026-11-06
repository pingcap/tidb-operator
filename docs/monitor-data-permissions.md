# Monitor data permissions

TiDB Operator prepares writable monitoring directories using a shared group,
without making them world-writable. Existing application UIDs/GIDs and data paths
are retained. This change does not require upgrading or pairing initializer and
reloader image versions.

The initializer prepares `/data/prometheus`, `/data/latest-rules` and Grafana's
data directories. It preserves their owners, sets their group to the configured
`spec.podSecurityContext.fsGroup` (including zero), grants owner/group access and
setgid, and removes other-write permission. Without an explicit `fsGroup`, the
Operator adds supplemental group 2000 instead of enabling recursive volume
ownership changes. Existing files are not recursively chmodded or chowned.

Prometheus and Thanos mount only the Prometheus subdirectory, the reloader mounts
only persisted rules, and Grafana mounts its data subdirectories. Explicit
additional mounts at a data path or its parent take precedence for Prometheus,
Thanos and Grafana, preserving their volume, subPath and read-only settings.
Initializers still see the full data volume to support existing initialization
scripts. This
limits filesystem visibility between containers; it is not isolation from a
privileged container or an explicitly configured additional full-volume mount.

## Supported configurations

| Configuration | Upgrade behavior |
| --- | --- |
| Default root initializer, new or existing PVC | Automatically prepares directory permissions |
| Existing `fsGroup`, `runAsUser`, `runAsGroup` or supplemental groups | Preserves the configured identities and groups |
| Non-root initializer owns its writable directories | Prepares permissions without switching to root |
| Non-root initializer, root-owned legacy directories | Requires the one-time migration below, even if the old directories were `777` |
| Storage disallows chgrp/chmod, or policy disallows the required initializer identity | Storage administrator must prepare the directories before upgrade |
| TiDB followed by DM initialization | Both initializers prepare the same directories without deleting the other's rules |
| Independent old/new initializer and reloader images | Retains their existing initialization and persistence protocols |

Literal absolute `GF_PATHS_DATA`, `GF_PATHS_PLUGINS` and `GF_PATHS_LOGS` paths
below `/data/` get corresponding subPath mounts. Additional volume mounts take
precedence during path validation, permission preparation and mounting; covered
paths are not prepared on the main PVC. For example, `GF_PATHS_LOGS=/data` is
allowed when `/data` is explicitly mounted from another volume. Paths outside
`/data/` keep their explicitly configured storage;
the Operator does not allocate new storage for them. Relative paths, environment
expansions, parent traversal, and using `/data` itself as a Grafana
data directory must be replaced with a literal dedicated path or an appropriate
additional volume mount before upgrade. Symlinked data directories are rejected.

## One-time migration for a custom non-root initializer

Use the existing `TidbMonitor` security-context fields; no new API or image version
is needed. Record the current configuration first.

1. Temporarily override the initializer's user to root, leaving existing group
   settings unchanged:

   ```yaml
   spec:
     initializer:
       securityContext:
         runAsUser: 0
         runAsNonRoot: false
     # Include this only when spec.dm is already configured.
     dm:
       initializer:
         securityContext:
           runAsUser: 0
           runAsNonRoot: false
   ```

2. Upgrade the Operator. Check that all monitor init containers complete
   successfully. The temporary monitor Pod need not become Ready: an old
   initializer running as root can create rule files that the configured
   non-root reloader cannot edit until the next initialization.
3. Restore the original initializer security contexts, including the original
   `runAsUser` and `runAsNonRoot` values. The next Pod initialization recreates
   active rules under the original identity. Confirm that the monitor becomes
   Ready and that saving rules survives another Pod recreation.

After either configuration change, first wait for the StatefulSet's Pod template
to contain the corrected settings. If the old monitor Pod is not replaced
automatically, recreate it. This applies both to failed initialization and to
completed initialization followed by a non-Ready reloader. Restore the original
UID once initialization completes; do not wait for the temporary Pod to be Ready.
Never delete its PVC.
If admission policy or storage restrictions prevent this migration, have the
storage administrator prepare the directories for the selected group with
owner/group access, setgid and no other-write permission instead. The Operator
fails initialization rather than silently restoring `777`.

When a nonzero reloader UID is explicitly configured (directly or inherited from
the Pod), the Operator supplies `RELOADER_UID` to initializers that understand
that optional variable. Older images ignore it. An explicitly supplied
`RELOADER_UID` is preserved; it must agree with the intended reloader identity.

## Rollout, rollback and validation

The template change recreates the whole monitoring Pod, including Prometheus,
Grafana and the reloader. Expect monitoring downtime for a single replica; TiDB,
PD and TiKV Pods are not restarted. Existing PVC paths and rule-version directory
selection are unchanged.

Before and after upgrade, save a custom rule through the reloader API, verify it
is loaded by Prometheus, create a Grafana dashboard, and query an older Prometheus
sample. Recreate the monitoring Pod and repeat these checks. For DM monitoring,
save and check both TiDB and DM rules. Inspect directory ownership and modes from
the initializer's full data mount: target directories must not be world-writable.

Rolling back to an older Operator reintroduces its `chmod 777` behavior. If a
custom non-root deployment relies on the automatically added supplemental group
2000, retain that group explicitly in `spec.podSecurityContext.supplementalGroups`
before rollback, or have an administrator prepare permissions for the restored
configuration.

Validation covered real Kubernetes Pods and local persistent volumes in KIND,
published old initializer/reloader images, newer images, historical Prometheus
blocks and WAL samples, Grafana data, TiDB/DM rule editing and Pod recreation.
CSI-specific ownership handling, NFS/root-squash and arbitrary custom image
entrypoints require validation against the deployment's storage and images.
