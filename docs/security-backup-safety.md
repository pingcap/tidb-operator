# Backup cleanup safety and tracker stability

<!-- toc -->
- [Existing backups](#existing-backups)
<!-- /toc -->

This change preserves existing cross-namespace Backup and Restore references. It does not add namespace consent annotations, caller authorization, or Cluster UID bindings. Those authorization questions are tracked separately.

Invalid persisted log Backups without BR configuration return errors instead of crashing Operator startup or checkpoint refresh. Tracker map reads share the writer lock. Group lookups and checkpoint clients use the configured target Cluster namespace. Existing Restore Jobs can finish reporting status when the Cluster is unavailable. Stop Jobs mount TLS credentials at the cluster client path.

New Backups record a credential-free `status.storageIdentity` before writing data. Cleanup and log truncation check that fingerprint against the effective destination, including endpoint/account and Local volume mounts, after the last status refresh. The original path and fingerprint cannot be overwritten by a stale worker. A changed storage destination requires a new Backup; restore an accidentally changed spec before cleaning the original data.

The cleaner refuses bucket roots, noncanonical remote paths, Local mount roots, and directory traversal. Local cleanup uses directory handles to prevent symlink escape and removes only entries within the opened backup directory. Contained aliases to a proper subdirectory remain supported.

Ordinary BR tuning options, short flags, negative values, and PD configuration remain supported. Backup options cannot replace managed storage targets through `--storage`, `--full-backup-storage`, S3 endpoint/region/provider, GCS endpoint, or Azure endpoint/account; use storage-provider configuration instead. Generated storage arguments are appended after raw options so short storage flags cannot change the recorded destination. A standalone `--` in Backup options is rejected because it would disable these managed flags. Restore options retain their previous behavior.

## Existing backups

Existing log Backups do not need a new Cluster binding to continue tracking, pause, resume, or stop. Backups with a recorded path but no storage fingerprint can retry the original path without manufacturing a historical fingerprint from the current spec. Upgrading does not restart or terminate existing Jobs.

Automatic cleanup and log truncation of a historical Backup without its original storage identity remain blocked: its original endpoint or mounted volume cannot be proven from a mutable spec. An administrator must verify and clean the original destination separately. This restriction applies to destructive operations, rather than normal log-task management. Backups intentionally written to a storage root also require administrator cleanup. Never populate the fingerprint from the current spec without historical evidence.

Install the updated Backup CRD before the new backup-manager image so the API server retains `status.storageIdentity`. Keep status writes restricted to trusted controllers and administrators. GCS fingerprints include the runtime emulator/universe endpoint; Azure fingerprints include the effective account. Credential rotation alone does not change the destination identity.
