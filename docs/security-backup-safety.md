# Backup safety and cross-namespace consent

<!-- toc -->
<!-- /toc -->

Backup and Restore references resolve to their own namespace by default. A cross-namespace `spec.br.clusterNamespace` now requires the target Cluster administrator to list the source namespace in the Cluster annotation `br.pingcap.com/allowed-backup-namespaces`, using comma-separated namespace names. Matching is exact; `*` is not supported.

This permission covers both backup reads and destructive restores. Only list namespaces whose CR writers are trusted to manage the target database. The source namespace administrator still needs to supply suitable storage/TLS credentials and network access. Removing the annotation prevents subsequent Operator resolution, including tracker refresh; it does not kill an already running BR Job or revoke data-plane credentials. Stop those Jobs and rotate credentials separately when revoking active access.

The BR cleaner now refuses to open storage when the live bucket/prefix no longer matches `status.backupPath`. It also refuses bucket roots, noncanonical remote paths, and local mount roots or parent-directory traversal. Local BR cleanup remains supported for a relative subdirectory. Snapshot and log backup starts cannot replace an existing different recorded path. Create a new Backup when changing the destination; inspect storage and recover an accidentally changed CR instead of widening its deletion scope. These checks rely on the Kubernetes status subresource remaining writable only by trusted controllers/administrators.

A log Backup without `spec.br.cluster` returns an error rather than panicking during startup or checkpoint refresh. Invalid persisted CRs are skipped without preventing other trackers from starting. Tracker map reads now share the writer lock.

These changes do not delete existing storage objects, rotate credentials or terminate running Jobs during the source update. Cleanup of backups intentionally written to a storage root must be performed explicitly by their storage administrator.

The original storage destination is recorded as a credential-free `status.storageIdentity` before data is written. Cleanup and restart also compare this fingerprint, including the S3 endpoint, Azure effective account (including Secret-provided accounts), and local volume source. Credential rotation does not change the destination identity. Old Backups with a recorded path but no identity cannot establish their original endpoint from the current mutable spec: automatic cleanup/restart refuses them. A storage administrator must verify the original destination and perform cleanup manually; do not populate the fingerprint from the current spec without historical evidence.

When cross-namespace consent is revoked during log-backup deletion, the Backup records `StopLogBackupBlocked` and retains its finalizer. Restore target consent, wait for the Operator to stop the remote task, and then let normal cleanup finish. If consent cannot be restored, an administrator must stop and verify the remote BR task and clean the original data before considering manual finalizer removal. Removing a finalizer alone does not stop BR. After consent is granted, ordinary Backup reconciliation re-enrolls an already-started log backup for progress tracking without restarting the Operator.
