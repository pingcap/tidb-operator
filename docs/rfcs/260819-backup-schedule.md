# Native BackupSchedule Support for Snapshot Backups

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Supported API](#supported-api)
  - [Scheduling](#scheduling)
  - [Backup Identity, Storage, and Ownership](#backup-identity-storage-and-ownership)
  - [Overlap Prevention](#overlap-prevention)
  - [Count-Based Retention](#count-based-retention)
  - [Status and Validation](#status-and-validation)
  - [Compatibility](#compatibility)
  - [Test Plan](#test-plan)
  - [Feature Gate](#feature-gate)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a release*.

- [ ] (R) This design doc has been discussed and approved
- [ ] (R) Test plan is in place
  - [ ] (R) e2e tests in kind
- [ ] (R) Graduation criteria is in place if required
- [ ] (R) User-facing documentation has been created in [pingcap/docs-tidb-operator]

## Summary

TiDB Operator v2 defines a `BackupSchedule` API, but [does not currently support the resource](https://docs.pingcap.com/tidb-in-kubernetes/v2.0/v2-vs-v1/). This proposal adds native scheduling and count-based retention for snapshot backups. It creates deterministic `Backup` resources with distinct storage destinations, uses best-effort overlap checks, and recovers safely from retries and restarts. The existing `Backup` controller continues to execute each backup and apply its `cleanPolicy`.

## Motivation

Users can create individual `Backup` resources in v2, but recurring backups require an external scheduler. A native controller lets users configure recurring backups and retention through `BackupSchedule` instead of maintaining a separate scheduler.

### Goals

- Create deterministic recurring snapshot `Backup` resources from `spec.backupTemplate`.
- Support pause and create at most one Backup for the newest missed scheduled time, without duplicates across retries and restarts.
- Use best-effort checks to avoid starting a snapshot Backup while another snapshot Backup targets the same TiDB cluster.
- Support count-based retention through `spec.maxBackups` without changing unrelated Backups or a generated Backup's `cleanPolicy`.
- Fail closed when unsupported `BackupSchedule` behavior is requested.

### Non-Goals

- Time-based retention through `spec.maxReservedTime`.
- Log-backup scheduling or compaction.
- Automated restore verification.
- Full behavioral parity with every v1 `BackupSchedule` feature.

## Proposal

A new `BackupSchedule` controller watches `BackupSchedule` and `Backup` resources. Scheduling and retention run in separate reconciliation loops:

- Scheduling creates or recovers at most one due `Backup`.
- Retention deletes generated `Backup` resources that exceed the configured limits.

The new controller manages `Backup` resources and `BackupSchedule` status but does not execute backups. The existing `Backup` controller resolves the target cluster and credentials, manages the BR Job and Backup status, and cleans stored data according to `cleanPolicy`.

Both loops validate the `BackupSchedule` before creating or deleting a Backup and retry independently.

### Risks and Mitigations

- If Backup creation succeeds but the following status update fails, a retry could create a duplicate. Deterministic names and identifying metadata let the controller recognize the existing Backup and repair status.
- Incorrect association could delete an unrelated Backup. Retention selects by the full `BackupSchedule` UID, requires valid scheduled-time metadata, and uses Backup UID and `resourceVersion` preconditions on deletion.
- BR [does not support concurrent backup tasks on one cluster](https://docs.pingcap.com/tidb/stable/backup-and-restore-overview/). The controller checks for nonterminal snapshot Backups targeting the same cluster immediately before creation. This is best-effort coordination, not a distributed lock.
- Deleting an expired `Backup` can remove stored data when its `cleanPolicy` requests deletion. The schedule controller preserves the configured policy and never deletes storage directly.

## Design Details

### Supported API

This proposal supports a subset of the existing `BackupSchedule` API:

| Field | Behavior |
| --- | --- |
| `spec.schedule` | Required five-field cron expression or supported v1 descriptor such as `@daily` or `@every <duration>`, evaluated in UTC. `@every` durations must be positive and at least one minute. |
| `spec.pause` | Stops new Backup creation and retention deletion. |
| `spec.maxBackups` | A positive value enables count-based retention. Zero or absence disables it. |
| `spec.backupTemplate` | Required snapshot `BackupSpec` copied into each generated Backup. |

The template must contain the complete BR and storage configuration needed by the generated Backup. `backupTemplate.backupMode` must be omitted or set to `snapshot`; `backupTemplate.logSubcommand` and `backupTemplate.logTruncateUntil` must be empty, and `backupTemplate.logStop` must be false. Exactly one of `s3`, `gcs`, `azblob`, or `local` must be configured so the controller can derive one per-run destination.

Six-field cron expressions and per-resource time zones are unsupported. Any other explicitly configured `BackupSchedule` spec field, including time-based retention, log backup, compaction, or schedule-level inheritance, makes the specification invalid.

### Scheduling

A scheduled time is produced by evaluating `spec.schedule` in UTC and can create at most one Backup.

`status.lastScheduleTime` records the latest UTC point through which scheduled times have been handled, including times that created a Backup or were intentionally skipped. It may initialize to the time of first observation. It advances only after confirmed creation or an intentional skip; errors leave it unchanged.

Scheduling follows these rules:

- On first observation, the controller selects the newest valid scheduled-time annotation among same-namespace Backups with a matching schedule UID. It persists the recovered status before considering more scheduling work. If none exists, it initializes `lastScheduleTime` to the current UTC time and does not create an immediate Backup.
- If several scheduled times are due, the controller skips all but the newest. It does not burst backfill because historical times cannot produce historical snapshots.
- While the controller observes `pause: true`, due times advance `lastScheduleTime` without creating Backups. Resuming does not backfill those times. If pause is enabled and disabled entirely while the controller is unavailable, those times follow the normal missed-time rules.
- A schedule change keeps the existing `lastScheduleTime` and applies the normal newest-only rule to the new expression.
- The controller returns `RequeueAfter` for the next scheduled time. Correctness does not depend on that in-memory timer because progress and generated Backups are persisted.

If Backup creation succeeds but the following status update fails, the next reconciliation finds the deterministic Backup and repairs status rather than creating another one.

### Backup Identity, Storage, and Ownership

Each generated Backup is named:

```text
<schedule-prefix>-<uid-hash>-<scheduled-utc>
```

The schedule prefix is the first 16 characters of the `BackupSchedule` name after replacing dots with hyphens and removing a trailing hyphen. The UID hash contains the first 16 lowercase hexadecimal characters from SHA-256 of the full `BackupSchedule` UID, and the UTC timestamp uses `yyyyMMddHHmmss`. The resulting name is at most 48 characters so names derived by the existing Backup controller remain within Kubernetes limits.

The generated Backup receives a label containing the full `BackupSchedule` UID and annotations containing the full schedule name and scheduled time as a valid UTC timestamp. The UID label and scheduled-time annotation are authoritative; the schedule-name annotation is informational.

The controller uses `<template-prefix>/<backup-name>` as the generated storage prefix, giving every run a distinct destination.

Generated Backups do not have an owner reference to the `BackupSchedule`, so deleting or recreating a schedule does not delete or adopt them. Once a schedule has a deletion timestamp, the controller stops creating and deleting Backups for it.

Before creating a Backup, the controller checks the deterministic name. It accepts an existing object only when its UID label and scheduled-time annotation match. A conflicting object is left unchanged and reconciliation fails.

### Overlap Prevention

Before creation, the controller checks for any nonterminal snapshot Backup targeting the same TiDB cluster, including manually created Backups and Backups from another schedule. The target is identified by `spec.br.cluster` and the effective cluster namespace, which is `spec.br.clusterNamespace` when set and the Backup namespace otherwise.

A snapshot Backup has finished execution when it has `Complete=True`, `Failed=True`, or `Invalid=True`. Log Backups do not block snapshot creation. If a blocking Backup exists, the controller waits without advancing `lastScheduleTime`. When the blocker finishes, the controller recalculates the schedule and considers only the newest due time.

### Count-Based Retention

Retention manages only snapshot Backups in the same namespace that have the current schedule UID label and a valid scheduled-time annotation. Backups without this controller-managed metadata, Backups from another schedule, and Backups created by an older object with the same schedule name are excluded.

When `maxBackups` is positive:

- Successful Backups with `Complete=True` are ordered by scheduled time, with name as a deterministic tie-breaker. The newest `maxBackups` are retained.
- Active Backups are never deleted. Backups that already have a deletion timestamp count toward neither `maxBackups` nor the five-Backup Failed/Invalid history.
- Failed and Invalid Backups do not count against `maxBackups`. The five newest are retained as a combined troubleshooting history; older entries are eligible for deletion.
- A Backup with the matching schedule UID label but missing or malformed scheduled-time metadata causes retention to fail without deleting anything.

`maxBackups` limits successful recovery points, not the total number of Backup resources. Unlike v1, failures do not displace successful recovery points; retaining the five newest provides bounded diagnostic history without another public field.

When `maxBackups` is absent or zero, nothing is pruned. While `pause` is true, no new deletion requests are sent, although pausing cannot cancel an accepted request. Pausing retention intentionally differs from v1.

Expired Backups are deleted from oldest to newest through the Kubernetes API using UID and `resourceVersion` preconditions. On failure, the controller stops and recalculates the current set during the next reconciliation; earlier successful deletions are not rolled back. The controller does not alter `cleanPolicy` or delete stored objects directly. The existing Backup controller applies the policy configured on each Backup.

### Status and Validation

This proposal adds two optional fields to `BackupScheduleStatus`:

```go
LastScheduleTime *metav1.Time `json:"lastScheduleTime,omitempty"`

// +listType=map
// +listMapKey=type
Conditions []metav1.Condition `json:"conditions,omitempty"`
```

The controller reports two conditions:

| Condition | Responsibility |
| --- | --- |
| `SchedulingReady` | Scheduling, creation, overlap waiting, and status recovery. |
| `RetentionReady` | Retention calculation and deletion requests. |

Each condition records the evaluated `metadata.generation`. `True` with reason `Reconciled` means its loop reached a safe result, including an intentional wait or no-op. `False` with reason `InvalidSpec` means the current specification is unsupported or invalid and no Backups were created or deleted. `False` with reason `ReconcilerError` means reconciliation could not safely complete.

The scheduling loop patches `lastScheduleTime`, `lastBackup`, `lastBackupTime`, and `SchedulingReady`. The retention loop patches `RetentionReady`. Both use optimistic concurrency and preserve status owned by the other loop.

The existing fields keep their current meanings: `lastBackup` names the most recently created or confirmed Backup, and `lastBackupTime` is its creation time. Neither changes for intentionally skipped scheduled times.

The CRD enforces a nonempty schedule and `maxBackups >= 0`. Reconciliation handles schedule parsing, supported-subset checks, and generated Backup validation before either loop acts, reusing static `BackupSpec` validation. No validating webhook is added, so semantic errors can be stored, but reconciliation reports them and creates or deletes no Backups until corrected. Reconcile-time validation also protects objects stored under an older CRD schema.

### Compatibility

The API remains `br.pingcap.com/v1alpha1`. Both new status fields are optional, so no new API version, conversion webhook, or stored-object migration is required. The updated CRD must be installed before enabling the controller.

Existing valid resources follow the first-observation behavior described under Scheduling. This avoids an immediate Backup during an operator upgrade when no matching generated Backup exists.

Existing log-backup and compaction status fields are not modified. Existing resources that request unsupported behavior remain stored but report `InvalidSpec` until corrected.

### Test Plan

Unit and controller-level tests use a controlled clock and cover:

- Cron parsing, first observation, missed scheduled times, pause, schedule changes, status recovery, and idempotency.
- Deterministic identity, per-run storage prefixes, cluster matching, and overlap handling.
- Supported-subset validation, retention and deletion safety, protection of unrelated Backups, and upgrade compatibility.

An end-to-end test in kind creates a `BackupSchedule`, confirms that the existing Backup controller accepts the generated Backup, verifies distinct destinations and retry safety, and exercises count-based retention.

### Feature Gate

No new feature gate is proposed. Users opt in by creating a `BackupSchedule`. An existing resource without a matching generated Backup initializes scheduling progress without creating an immediate Backup.

## Drawbacks

- The supported subset is smaller than the v1 `BackupSchedule` feature set.
- Without an admission webhook, some invalid resources remain stored until corrected.
- Cluster-wide overlap checking adds Backup watch and list work and cannot eliminate a race with other Backup creators.
- `maxBackups` does not bound all Backup resources, and deleting a `BackupSchedule` intentionally leaves its generated Backups behind.

## Alternatives

### Continue Using External Scheduling

An external scheduler or Kubernetes CronJob can provide a timer but still needs custom logic for Backup creation, idempotency, overlap checks, status, and retention.

### Port the v1 Controller Directly

A direct v1 port would include log backup, compaction, time-based retention, and ownership assumptions outside this proposal. The smaller subset allows those behaviors to be reviewed independently.

### Use an Owner Reference for Generated Backups

An owner reference could garbage-collect generated Backups when a schedule is deleted. Preservation must not depend on the deleting client's propagation policy, so this proposal uses UID metadata instead.

[pingcap/docs-tidb-operator]: https://github.com/pingcap/docs-tidb-operator
