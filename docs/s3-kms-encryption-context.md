# S3 SSE-KMS encryption context for BR

`spec.s3.sseKmsEncryptionContext.values` supplies additional encryption context
as string key-value pairs. Operator serializes the map to JSON and passes it as
one BR argument. BR validates the JSON and performs the Base64 encoding for S3.
Do not Base64-encode values or add shell quoting to them.

## Prerequisites

Install the updated Operator CRDs and use the corresponding backup-manager image.
The BR image selected by `spec.toolImage` and all participating TiKV instances
must support custom SSE-KMS encryption context. This change does not upgrade
those components or add support to an old binary. No released minimum version
has been assigned yet; the corresponding changes are
[TiDB #71983](https://github.com/pingcap/tidb/pull/71983),
[TiKV #20193](https://github.com/tikv/tikv/pull/20193), and
[kvproto #1552](https://github.com/pingcap/kvproto/pull/1552).
An old BR can reject the new flag; an old TiKV does not consume the new protocol
field. Upgrade all participating components before enabling this option.

Use an S3 bucket and KMS key that the backup/restore identities are authorized to
access. Configure `sse: aws:kms` explicitly for writes with context; Operator
does not infer the SSE mode or a KMS key ID from the context. The key ID remains
available through `spec.br.options`.

## Backup

Replace the example bucket, key ARN, image and cluster names with your own.
The existing S3 authentication configuration still applies.

```yaml
apiVersion: pingcap.com/v1alpha1
kind: Backup
metadata:
  name: kms-context-backup
spec:
  backupMode: snapshot
  toolImage: your-registry/br:sse-kms-context
  br:
    cluster: source-tidb
    options:
      - --s3.sse-kms-key-id=arn:aws:kms:us-east-1:123456789012:key/REPLACE-ME
  s3:
    provider: aws
    region: us-east-1
    bucket: my-backup-bucket
    prefix: snapshots/example
    sse: aws:kms
    sseKmsEncryptionContext:
      values:
        tenant: tenant-a
        purpose: backup
```

## BackupSchedule and log backup

Put the same S3 configuration under `spec.backupTemplate.s3` for scheduled
snapshots and under `spec.logBackupTemplate.s3` for log backup. Each template has
its own BR options and tool image; configure both when using both modes.

```yaml
apiVersion: pingcap.com/v1alpha1
kind: BackupSchedule
metadata:
  name: kms-context-schedule
spec:
  schedule: "0 2 * * *"
  backupTemplate:
    backupMode: snapshot
    toolImage: your-registry/br:sse-kms-context
    br:
      cluster: source-tidb
      options:
        - --s3.sse-kms-key-id=arn:aws:kms:us-east-1:123456789012:key/REPLACE-ME
    s3:
      provider: aws
      bucket: my-backup-bucket
      region: us-east-1
      prefix: scheduled-snapshots
      sse: aws:kms
      sseKmsEncryptionContext:
        values:
          tenant: tenant-a
  logBackupTemplate:
    backupMode: log
    toolImage: your-registry/br:sse-kms-context
    br:
      cluster: source-tidb
      options:
        - --s3.sse-kms-key-id=arn:aws:kms:us-east-1:123456789012:key/REPLACE-ME
    s3:
      provider: aws
      bucket: my-backup-bucket
      region: us-east-1
      prefix: logs
      sse: aws:kms
      sseKmsEncryptionContext:
        values:
          tenant: tenant-a
```

Context is saved by BR when the log backup task is created. Editing an existing
Backup or BackupSchedule, upgrading its binaries, or pausing/resuming a task
does not replace that task's saved context. Plan any transition for an existing
task separately; do not assume uninterrupted log coverage after recreating it.

## Restore and PITR

Restore accepts the same `spec.s3` field and BR options:

```yaml
apiVersion: pingcap.com/v1alpha1
kind: Restore
metadata:
  name: kms-context-restore
spec:
  restoreMode: snapshot
  toolImage: your-registry/br:sse-kms-context
  br:
    cluster: target-tidb
    options:
      - --s3.sse-kms-key-id=arn:aws:kms:us-east-1:123456789012:key/REPLACE-ME
  s3:
    provider: aws
    bucket: my-backup-bucket
    region: us-east-1
    prefix: snapshots/example
    sse: aws:kms
    sseKmsEncryptionContext:
      values:
        tenant: tenant-a
        purpose: backup
```

For PITR, `spec.s3` describes the log storage. If a separate full backup storage
is configured, its context belongs under
`spec.pitrFullBackupStorageProvider.s3.sseKmsEncryptionContext.values`.
Operator URL-encodes this JSON in `--full-backup-storage`, so it does not overwrite
the main storage's context. This does not add separate CLI flags for that URL's
other S3 settings.

These settings configure BR storage; they do not add an encryption-context
header to S3 reads. Read access still depends on the existing S3/KMS permissions. See the
[AWS GetObject documentation](https://docs.aws.amazon.com/AmazonS3/latest/API/API_GetObject.html).

## Validation and precedence

- Omitted `sseKmsEncryptionContext`: no new parameter, preserving existing behavior.
- `sseKmsEncryptionContext: {values: {}}`: explicit empty JSON object (`{}`).
- `values` is required when the field is present. Numbers, booleans, nested
  objects, arrays and null are not accepted as string values. Quote numeric
  strings in YAML, for example `tenant: "123"`.
- `spec.br.options` retains its existing highest priority over generated CLI
  flags. A later `--s3.sse-kms-context=...` (or two-element flag/value pair)
  overrides the generated main-storage argument. An explicit storage URL can
  also carry context; BR's URL option precedence continues to apply. Prefer one
  configuration source for each storage to avoid ambiguity.
- `spec.s3.options` contains rclone options and is not the BR passthrough.
- This field applies to BR-based operations. It does not enable context in
  Dumpling/rclone, Lightning, or Operator's own cloud storage clients. The shared
  S3 schema also appears in compaction and federation CRDs; the schema alone
  does not guarantee support by every underlying tool.

## Logging and validation scope

[Encryption context is not encrypted](https://docs.aws.amazon.com/AmazonS3/latest/userguide/UsingKMSEncryption.html#encryption-context) and is not a secret store. It remains visible in Kubernetes
objects and must not contain credentials or other sensitive information.
Operator omits the full CR from startup logs and redacts context and S3 URL credentials in its own BR
command logs and startup errors, including context supplied through options or
storage URLs. It still forwards BR/TiKV output; those binaries must independently
handle their own log redaction.

Unit tests validate the generated CRD schemas, string escaping, explicit empty
context, omitted configuration, option precedence, scheduled backup copying and
command-log redaction. Before deployment, also run an Operator-managed snapshot
backup/restore and log backup/PITR against a dedicated S3/KMS test setup. Verify
restored data, task continuation after restart, and policy rejection when the
required context is absent or wrong. Passing local argument/schema tests is not
an Operator Kubernetes/AWS E2E result.
