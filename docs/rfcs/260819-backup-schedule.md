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
  - [Derive the Target from backupTemplate](#derive-the-target-from-backuptemplate)
  - [Continue Using External Scheduling](#continue-using-external-scheduling)
  - [Port the v1 Controller Directly](#port-the-v1-controller-directly)
  - [Use an Owner Reference for Generated Backups](#use-an-owner-reference-for-generated-backups)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a release*.

- [ ] (R) This design doc has been discussed and approved
- [ ] (R) Test plan is in place
  - [ ] (R) e2e tests in kind
- [ ] (R) Graduation criteria is in place if required
- [ ] (R) User-facing documentation has been created in [pingcap/docs-tidb-operator]

## Summary

TiDB Operator v2 defines a `BackupSchedule` API, but [does not currently support the resource](https://docs.pingcap.com/tidb-in-kubernetes/v2.0/v2-vs-v1/). This proposal adds a standard, immutable cluster reference together with native scheduling and count-based retention for snapshot backups. It creates deterministic `Backup` resources with distinct storage destinations, uses best-effort overlap checks, and recovers safely from retries and restarts. The existing `Backup` controller continues to execute each backup and apply its `cleanPolicy`.

## Motivation

Users can create individual `Backup` resources in v2, but recurring backups require an external scheduler. A native controller lets users configure recurring backups and retention through `BackupSchedule` instead of maintaining a separate scheduler.

### Goals

- Create deterministic recurring snapshot `Backup` resources for `spec.cluster` from `spec.backupTemplate`.
- Support pause and create at most one Backup for the newest missed scheduled time, without duplicates across retries and restarts.
- Use best-effort checks to avoid starting a snapshot Backup while another snapshot Backup targets the same TiDB cluster.
- Support count-based retention through `spec.maxBackups` without changing unrelated Backups or a generated Backup's `cleanPolicy`.
- Fail closed when unsupported `BackupSchedule` behavior is requested.

### Non-Goals

- Time-based retention through `spec.maxReservedTime`.
- Log-backup scheduling or compaction.
- Cross-namespace `BackupSchedule` targets.
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

This proposal adds the standard v2 cluster reference and otherwise supports a subset of the existing `BackupSchedule` API:

| Field | Behavior |
| --- | --- |
| `spec.cluster` | Required `core/v1alpha1.ClusterReference`. Its `name` identifies a Cluster in the `BackupSchedule` namespace and is the authoritative target for every generated Backup. |
| `spec.schedule` | Required five-field cron expression, one of `@yearly`, `@annually`, `@monthly`, `@weekly`, `@daily`, `@midnight`, or `@hourly`, or `@every <duration>`, evaluated in UTC. The duration uses Go `time.ParseDuration` syntax, such as `15m` or `1h30m`, and must be at least one minute. |
| `spec.pause` | Stops new Backup creation and retention deletion. |
| `spec.maxBackups` | A positive value enables count-based retention. Zero or absence disables it. |
| `spec.backupTemplate` | Required snapshot `BackupSpec` for each generated Backup. The controller deep-copies it, projects the target from `spec.cluster`, and appends the generated Backup name to the configured storage prefix. Schedule-level BR, storage, volume, and image-pull settings are not inherited. |
| `spec.backupTemplate.br` | Optional BR settings. If present, its `cluster` must equal `spec.cluster.name` and `clusterNamespace` must be empty. The nested value is not a second target authority. |

The new field is represented as:

```go
Cluster corev1alpha1.ClusterReference `json:"cluster"`
```

The CRD exposes `.spec.cluster.name` as a selectable field and adds a `Cluster` printer column, following the convention used by other cluster-bound v2 resources. The shared `ClusterReference` schema requires `name`, applies the project's DNS-shaped regular expression, and applies its field-scoped `self == oldSelf` transition rule. The shared schema does not declare the 253-character DNS-1123 subdomain limit, so admission enforces the shape but not the complete Kubernetes length constraint. Reconciliation also calls Kubernetes DNS-1123 subdomain validation, including the length limit, and fails closed before either loop performs a side effect.

When rendering a Backup, the controller deep-copies the template, creates `spec.br` if necessary, sets `spec.br.cluster` to `spec.cluster.name`, and leaves `spec.br.clusterNamespace` empty. A conflicting nested target makes the schedule invalid.

The template must contain the remaining BR and storage configuration needed by the generated Backup. `backupTemplate.backupMode` must be omitted or set to `snapshot`; `backupTemplate.logSubcommand` and `backupTemplate.logTruncateUntil` must be empty, and `backupTemplate.logStop` must be false. Exactly one of `s3`, `gcs`, `azblob`, or `local` must be configured so the controller can derive one per-run destination.

Six-field cron expressions and per-resource time zones are unsupported. Any other explicitly configured `BackupSchedule` spec field, including time-based retention, log backup, compaction, or the legacy schedule-level BR and storage inheritance fields, makes the specification invalid.

### Scheduling

A scheduled time is produced by evaluating `spec.schedule` in UTC and can create at most one Backup.

Calendar expressions use UTC. An `@every` expression is anchored to the durable `lastScheduleTime` cursor, including the first-observation time when no managed Backup is recovered. Every subsequent occurrence is derived from the preceding persisted occurrence rather than from reconciliation time. If an occurrence is already due but overlap or another blocker leaves the cursor unchanged, a wake deadline derived from the current time is only a safety poll and does not become a logical scheduled time.

`status.lastScheduleTime` records the latest UTC point through which scheduled times have been handled, including times that created a Backup or were intentionally skipped. It may initialize to the time of first observation. It advances only after confirmed creation or an intentional skip; errors leave it unchanged.

Scheduling follows these rules:

- On first observation, the controller selects the newest valid scheduled-time annotation among same-namespace Backups with a matching schedule UID. It persists the recovered status before considering more scheduling work. If none exists, it initializes `lastScheduleTime` to the current UTC time and does not create an immediate Backup.
- If several scheduled times are due, the controller skips all but the newest. It does not burst backfill because historical times cannot produce historical snapshots.
- A reconciliation examines at most 1000 due times. When more are due, an unblocked loop advances through bounded batches and immediately requeues before creating only the newest remaining occurrence. A paused loop advances through the same bounded batches without creating a Backup.
- While the controller observes `pause: true`, due times advance `lastScheduleTime` without creating Backups. Resuming does not backfill those times. If pause is enabled and disabled entirely while the controller is unavailable, those times follow the normal missed-time rules.
- A schedule change keeps the existing `lastScheduleTime` and applies the normal newest-only rule to the new expression.
- `spec.cluster.name` is immutable. Moving scheduling to another cluster requires a new `BackupSchedule` with a new UID and scheduling cursor.
- The controller returns `RequeueAfter` for the next scheduled time. Correctness does not depend on that in-memory timer because progress and generated Backups are persisted.

If Backup creation succeeds but the following status update fails, the next reconciliation finds the deterministic Backup and repairs status rather than creating another one.

### Backup Identity, Storage, and Ownership

Each generated Backup is named:

```text
<schedule-prefix>-<uid-hash>-<scheduled-utc>
```

The schedule prefix is the first 16 characters of the `BackupSchedule` name after replacing dots with hyphens and removing a trailing hyphen. The UID hash contains the first 16 lowercase hexadecimal characters from SHA-256 of the full `BackupSchedule` UID, and the UTC timestamp uses `yyyyMMddHHmmss`. The resulting name is at most 48 characters. For the supported snapshot mode, this reserves enough space for the existing `GetBackupJobName`, `GetCleanJobName`, and `GetVolumeBackupInitializeJobName` prefixes and suffixes while keeping every derived name within the 63-character Kubernetes limit. Log subcommands are unsupported and are rejected before a Backup is rendered.

The generated Backup receives a label containing the full `BackupSchedule` UID and annotations containing the full schedule name and scheduled time as a valid UTC timestamp. The UID label and scheduled-time annotation are authoritative; the schedule-name annotation is informational.

The controller normalizes the template storage prefix for S3, GCS, Azure Blob Storage, or local storage before appending the Backup name. It removes leading and trailing slashes, repeated slashes, and `.` path components, rejects any `..` component, and then uses slash-based `path.Join` semantics to produce `<normalized-template-prefix>/<backup-name>`. An empty template prefix produces `<backup-name>`. The result is canonical and gives every run a distinct destination.

Generated Backups do not have an owner reference to the `BackupSchedule`, so deleting or recreating a schedule does not delete or adopt them. Once a schedule has a deletion timestamp, the controller stops creating and deleting Backups for it.

The UID label and scheduled-time annotation are controller identity signals, not an authorization or security boundary. A principal that can create or update Backup resources in the namespace can forge them and can interfere with scheduling or retention. Operators must restrict Backup write permissions to trusted principals. The scheduling stage adds Backup create permission and the retention stage adds Backup delete permission to the controller's RBAC; existing chart roles can be broader.

Before creating a Backup, the controller checks the deterministic name. It accepts an existing object only when the schedule UID label, valid zero-offset RFC3339 scheduled time, deterministic name, snapshot mode, exact `spec.cluster` target, empty `clusterNamespace`, and valid canonical per-Backup destination all match the managed identity contract. Writers emit the scheduled time in whole-second UTC `Z` form; readers also accept equivalent fractional-second or `+00:00` forms and normalize them to UTC. The controller emits only the defined schedule identity keys and does not treat other metadata as provenance. A conflicting object is left unchanged and reconciliation fails.

Recovery and retention deliberately do not require equality with the schedule's current full template. A supported storage-template edit after successful creation must not erase recovery evidence or strand historical retention candidates. Each existing Backup's own validated canonical destination remains authoritative for that occurrence. Exact template equality is required only for the newly rendered object before creation. This safety model depends on trusted Backup writers as described above rather than on metadata alone.

### Overlap Prevention

Before creation, the controller checks for any nonterminal snapshot Backup targeting the same TiDB cluster, including manually created Backups and Backups from another schedule. The schedule target is the pair of the `BackupSchedule` namespace and `spec.cluster.name`; generated Backups carry that target in `spec.br.cluster` with an empty `spec.br.clusterNamespace`. Scheduling, recovery, and retention share a Backup target resolver that reads `spec.br.cluster` and uses `spec.br.clusterNamespace` when present, otherwise the Backup namespace. This mirrors Backup execution semantics. A missing or invalid BR target is safely treated as unresolved and is never dereferenced or matched to a cluster. Cross-namespace manual Backups participate in overlap checks through their effective target, but cross-namespace scheduled targets are not supported.

A snapshot Backup has finished execution only when exactly one of `Complete=True`, `Failed=True`, or `Invalid=True` is present. Contradictory terminal conditions fail closed as ambiguous. Log Backups do not block snapshot creation. If a blocking Backup exists, the controller waits without advancing `lastScheduleTime`. When the blocker finishes, the controller recalculates the schedule and considers only the newest due time.

### Count-Based Retention

Retention manages only snapshot Backups in the same namespace that have the current schedule UID label, a valid scheduled-time annotation, `spec.br.cluster` equal to `spec.cluster.name`, and an empty `spec.br.clusterNamespace`. Backups without this controller-managed metadata, Backups from another schedule, and Backups created by an older object with the same schedule name are excluded.

When `maxBackups` is positive:

- Successful Backups with `Complete=True` are ordered by scheduled time, with name as a deterministic tie-breaker. The newest `maxBackups` are retained.
- Active Backups are never deleted. Backups that already have a deletion timestamp count toward neither `maxBackups` nor the five-Backup Failed/Invalid history.
- Failed and Invalid Backups do not count against `maxBackups`. The five newest are retained as a combined troubleshooting history; older entries are eligible for deletion.
- A Backup with the matching schedule UID label but missing or malformed scheduled-time metadata causes retention to fail without deleting anything.

`maxBackups` limits successful Backup CR history, not the total number of Backup resources. Unlike v1, failures do not displace successful Backup CRs; retaining the five newest provides bounded diagnostic history without another public field.

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

The scheduling loop owns `lastScheduleTime`, `lastBackup`, `lastBackupTime`, and `SchedulingReady`. The retention loop owns only `RetentionReady`. For each status write, the loop performs one uncached read, verifies that the schedule UID and generation still match the state used to calculate the result, mutates only its owned fields, and sends one merge patch with an optimistic `resourceVersion` lock. It does not retry a conflicting patch internally. A stale UID, changed generation, or conflict returns control to the queue so the next full reconciliation reads both schedule and Backup state again before calculating another patch. This preserves status owned by the other loop and prevents a stale outcome from being replayed against newer state.

The existing fields keep their current meanings: `lastBackup` names the most recently created or confirmed Backup, and `lastBackupTime` is its creation time. Neither changes for intentionally skipped scheduled times.

The CRD enforces a nonempty schedule, `maxBackups >= 0`, a required DNS-shaped and immutable `spec.cluster.name`, an empty nested `clusterNamespace`, and equality between any nested template cluster and `spec.cluster.name`. Reconciliation enforces the complete DNS-1123 subdomain constraints, schedule parsing, supported-subset checks, and generated Backup validation before either loop acts, reusing static `BackupSpec` validation. No validating webhook is added, so semantic errors can be stored, but reconciliation reports them and creates or deletes no Backups until corrected. Reconcile-time validation also protects objects stored under an older CRD schema.

### Compatibility

The API remains `br.pingcap.com/v1alpha1`, but required `spec.cluster` changes the stored-resource contract. Existing manifests and stored v2 `BackupSchedule` objects must add `spec.cluster.name`. Adding a field to the CRD's `required` list is not covered by Kubernetes [validation ratcheting](https://kubernetes.io/docs/tasks/extend-kubernetes/custom-resources/custom-resource-definitions/#validation-ratcheting).

The migration is performed with the new controller disabled. Operators first export all stored schedules, install the updated CRD, and update each object according to the following matrix. Every row describes the required post-migration state, not an automatic conversion. A feature with no v2 equivalent must remain externally managed or be removed deliberately.

| Stored field or condition | Migration action | Required state before controller enablement |
| --- | --- | --- |
| `spec.cluster` is absent | Select the final same-namespace Cluster. Prefer the effective target already declared by `spec.backupTemplate.br` when it is valid. Add `spec.cluster.name` and reconcile the nested target in the same update. | `spec.cluster.name` is present, matches the shared admission pattern, is no longer than 253 characters, and names the intended same-namespace Cluster. |
| `spec.cluster.name` is already present | Verify it before installing or enabling the controller. | The value remains unchanged. Moving a schedule to another Cluster requires a new `BackupSchedule`. |
| `spec.schedule` | Keep a valid five-field expression or a supported descriptor. Rewrite six-field expressions, time-zone prefixes, sub-minute `@every` values, and unsupported descriptors. | The expression has a future UTC occurrence and passes controller parsing. |
| `spec.pause` | Preserve the intended boolean value. | No rewrite is required. A paused migrated schedule initializes or advances its cursor without creating a Backup. |
| `spec.maxBackups` | Keep an absent, zero, or positive value. Replace a negative value. | The value is nonnegative. Zero or absence disables count-based pruning. |
| `spec.maxReservedTime` | Choose an explicit count in `spec.maxBackups`, or keep time-based retention outside this controller, then remove the field. Do not infer a count from a duration. | Field is absent. |
| `spec.logBackupTemplate` | Move log-backup lifecycle management to a separate mechanism, then remove the field. Do not convert it into the snapshot template. | Field is absent. |
| `spec.compactSpan` or `spec.compactBackupTemplate` | Move compaction to a separate mechanism, then remove both fields. | Both fields are absent. |
| Schedule-level `spec.br` | Copy any still-required non-target BR options into `spec.backupTemplate.br`, resolve conflicts explicitly, and remove schedule-level `spec.br`. Set the template's `cluster` from `spec.cluster.name`, not from the removed field. | Schedule-level `spec.br` is absent. Template BR settings, if present, contain the final options and matching target. |
| Schedule-level `spec.s3`, `spec.gcs`, `spec.azblob`, or `spec.local` | Move the intended provider into `spec.backupTemplate` if it is not already there. Resolve duplicate or conflicting providers explicitly, then remove every schedule-level provider. | No schedule-level provider is present. Exactly one template-level provider is configured. |
| Schedule-level `spec.storageClassName` or `spec.storageSize` | Move each required value into the corresponding `spec.backupTemplate` field, resolving an existing template value explicitly, then remove or clear the schedule-level field. | `storageClassName` is absent; `storageSize` is empty or absent. |
| Schedule-level `spec.imagePullSecrets` | Move the intended list into `spec.backupTemplate.imagePullSecrets`, resolving an existing template list explicitly, then remove or clear the schedule-level field. | Schedule-level list is empty or absent. |
| `spec.backupTemplate.backupMode` | Keep an absent value or `snapshot`. Move log-backup behavior to a separate mechanism. | Value is absent or `snapshot`. |
| `spec.backupTemplate.logSubcommand`, `logTruncateUntil`, or `logStop` | Move log operations to a separate mechanism, then clear the fields. | Both strings are empty or absent and `logStop` is false or absent. |
| `spec.backupTemplate.br.cluster` | If template BR settings are present, set `cluster` equal to `spec.cluster.name`. If no BR settings are needed, the entire `br` object may be omitted and the controller creates the target projection. | Nested target is absent with the entire `br` object, or exactly equals `spec.cluster.name`. |
| `spec.backupTemplate.br.clusterNamespace` | Clear it for a same-namespace target. A cross-namespace schedule cannot be migrated in place; create a new `BackupSchedule` in the target Cluster's namespace. | Value is empty or absent. |
| Template storage providers | Select exactly one of `s3`, `gcs`, `azblob`, or `local`; retain the desired base prefix and fix invalid parent-path segments. | Exactly one provider passes static `BackupSpec` validation and has a safe, normalizable prefix. |
| All other `spec.backupTemplate` fields | Preserve the desired snapshot Backup settings and validate them as an ordinary `BackupSpec`. | The projected generated Backup passes static `BackupSpec` validation. |

The `self == oldSelf` [transition rule](https://kubernetes.io/docs/tasks/extend-kubernetes/custom-resources/custom-resource-definitions/#transition-rules) on `ClusterReference.name` is evaluated only when both the old and new values exist. Therefore, a stored object can make one absent-to-present update after the new CRD is installed. That first value must be correct: every later change is rejected at admission. If the nested BR target also needs correction, update `spec.cluster` and `spec.backupTemplate.br` atomically.

The following audit reports stored objects that still have mechanically detectable migration blockers. It intentionally does not guess target names, translate time retention to counts, parse cron, or replace full `BackupSpec` validation.

```sh
kubectl get backupschedules.br.pingcap.com --all-namespaces -o json \
  | jq -r '
      .items[] as $item
      | ($item.spec.backupTemplate // {}) as $template
      | ($item.spec.cluster.name? // "") as $cluster
      | ([
          if $cluster == "" then "missing spec.cluster.name" else empty end,
          if ($cluster | length) > 253 then "spec.cluster.name exceeds 253 characters" else empty end,
          if (($item.spec.maxBackups? // 0) < 0) then "negative spec.maxBackups" else empty end,
          if ($item.spec | has("maxReservedTime")) and $item.spec.maxReservedTime != null then "spec.maxReservedTime" else empty end,
          if ($item.spec | has("logBackupTemplate")) and $item.spec.logBackupTemplate != null then "spec.logBackupTemplate" else empty end,
          if ($item.spec | has("compactSpan")) and $item.spec.compactSpan != null then "spec.compactSpan" else empty end,
          if ($item.spec | has("compactBackupTemplate")) and $item.spec.compactBackupTemplate != null then "spec.compactBackupTemplate" else empty end,
          if ($item.spec | has("storageClassName")) and $item.spec.storageClassName != null then "schedule-level spec.storageClassName" else empty end,
          if (($item.spec.storageSize? // "") != "") then "schedule-level spec.storageSize" else empty end,
          if (($item.spec.imagePullSecrets? // []) | length) != 0 then "schedule-level spec.imagePullSecrets" else empty end,
          if ($item.spec | has("br")) and $item.spec.br != null then "schedule-level spec.br" else empty end,
          if ([$item.spec.s3?, $item.spec.gcs?, $item.spec.azblob?, $item.spec.local?] | map(select(. != null)) | length) != 0 then "schedule-level storage provider" else empty end,
          if (($template.backupMode? // "snapshot") != "snapshot") then "non-snapshot backupTemplate.backupMode" else empty end,
          if (($template.logSubcommand? // "") != "") then "backupTemplate.logSubcommand" else empty end,
          if (($template.logTruncateUntil? // "") != "") then "backupTemplate.logTruncateUntil" else empty end,
          if ($template.logStop? // false) then "backupTemplate.logStop" else empty end,
          if ([$template.s3?, $template.gcs?, $template.azblob?, $template.local?] | map(select(. != null)) | length) != 1 then "template storage provider count is not one" else empty end,
          if ($template.br? != null) and (($template.br.cluster? // "") != $cluster) then "nested cluster mismatch" else empty end,
          if (($template.br.clusterNamespace? // "") != "") then "cross-namespace nested target" else empty end
        ]) as $reasons
      | select(($reasons | length) != 0)
      | [$item.metadata.namespace, $item.metadata.name, ($reasons | join("; "))]
      | @tsv'
```

After the audit returns no rows, each migrated manifest is submitted through server-side dry-run and then applied. The controller is enabled only after all objects pass admission, the migration audit, cron parsing, and static `BackupSpec` validation:

```sh
kubectl apply --dry-run=server -f migrated-backup-schedules.yaml
kubectl apply -f migrated-backup-schedules.yaml
```

Migrated valid resources follow the first-observation behavior described under Scheduling. This avoids an immediate Backup during an operator upgrade when no matching generated Backup exists.

Existing log-backup and compaction status fields are not modified. Existing resources that request unsupported behavior remain stored but report `InvalidSpec` until corrected.

### Test Plan

Unit and controller-level tests use a controlled clock and cover:

- Cron parsing, first observation, missed scheduled times, pause, schedule changes, status recovery, and idempotency.
- Exact UTC descriptor support, durable `@every` anchoring, bounded occurrence scanning, and wake-only safety polling.
- Deterministic identity, per-run storage prefixes, required and immutable `spec.cluster`, target projection, same-namespace cluster matching, and overlap handling.
- Maximum-length coverage for generated Backup names and every derived snapshot Backup Job name.
- Prefix normalization and parent-traversal rejection for every supported storage provider.
- Supported-subset validation, nested-target equality, retention and deletion safety, and protection of unrelated Backups.
- Real API-server status tests force concurrent scheduling-owned and retention-owned writes, verify optimistic conflicts, and prove that a fresh post-conflict write preserves both loops' owned fields.
- Real API-server tests verify the one-time absent-to-present target migration, atomic nested-target repair, and later target immutability. Stored-object fixtures separately cover an absent cluster reference; schedule-level BR, storage, volume, and image-pull inheritance; time-based retention; log backup; compaction; template log operations; conflicting nested targets; and cross-namespace targets. Each fixture verifies fail-closed reconciliation without side effects until every required matrix action is complete.

An end-to-end test in kind creates a `BackupSchedule`, confirms that the generated Backup contains the projected target and is accepted by the existing Backup controller, verifies distinct destinations and retry safety, and exercises count-based retention.

### Feature Gate

No new feature gate is proposed. Users opt in by creating or migrating a valid `BackupSchedule`. The controller is enabled only after the CRD update and stored-object migration. A valid resource without a matching generated Backup initializes scheduling progress without creating an immediate Backup.

## Drawbacks

- The supported subset is smaller than the v1 `BackupSchedule` feature set.
- Without an admission webhook, some invalid resources remain stored until corrected.
- The required cluster reference requires manifest and stored-object migration. Reusing `BackupSpec` as the template type duplicates the cluster name when users configure nested BR settings.
- Retargeting requires a new `BackupSchedule`; its new UID does not adopt or prune Backups left by the old schedule.
- Cluster-wide overlap checking adds Backup watch and list work and cannot eliminate a race with other Backup creators.
- `maxBackups` does not bound all Backup resources, and deleting a `BackupSchedule` intentionally leaves its generated Backups behind.

## Alternatives

### Derive the Target from backupTemplate

The controller could continue deriving the target from `spec.backupTemplate.br.cluster`. That avoids an API migration and permits a direct template copy, but leaves the schedule target buried and mutable. The standard v2 `spec.cluster` reference gives each schedule one stable target.

### Continue Using External Scheduling

An external scheduler or Kubernetes CronJob can provide a timer but still needs custom logic for Backup creation, idempotency, overlap checks, status, and retention.

### Port the v1 Controller Directly

A direct v1 port would include log backup, compaction, time-based retention, and ownership assumptions outside this proposal. The smaller subset allows those behaviors to be reviewed independently.

### Use an Owner Reference for Generated Backups

An owner reference could garbage-collect generated Backups when a schedule is deleted. Preservation must not depend on the deleting client's propagation policy, so this proposal uses UID metadata instead.

[pingcap/docs-tidb-operator]: https://github.com/pingcap/docs-tidb-operator
