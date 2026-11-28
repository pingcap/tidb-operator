# S3 server-side encryption for BR

The `spec.s3.serverSideEncryption` object has three fields.

- `algorithm` selects the S3 encryption algorithm, for example `AES256`,
  `aws:kms`, or `aws:kms:dsse`.
- `kmsKeyId` selects the AWS KMS key for SSE-KMS. Use a key ID, ARN, or alias.
- `kmsEncryptionContext.values` supplies additional encryption context as
  string key-value pairs. Operator serializes the map to JSON and passes it as
  one BR argument. BR validates the JSON and Base64-encodes it for S3. Enter the
  original strings in the map.

## Prerequisites

Install the updated Operator CRDs and upgrade `tidb-controller-manager` and
the backup-manager image together. The `algorithm` and `kmsKeyId` fields use
existing BR flags. Use a tool image that supports the configured algorithm.

If you set `kmsEncryptionContext`, the BR image selected by `spec.toolImage`
and all participating TiKV instances must support custom SSE-KMS encryption
context. Older BR versions can reject the context flag. Older TiKV versions
can ignore the context protocol field. Upgrade both components before enabling
custom context. No released minimum version for context support has been
assigned yet. The corresponding changes are
[TiDB #71983](https://github.com/pingcap/tidb/pull/71983),
[TiKV #20193](https://github.com/tikv/tikv/pull/20193), and
[kvproto #1552](https://github.com/pingcap/kvproto/pull/1552).

Use an S3 bucket and KMS key that the backup and restore identities are
authorized to access. When the provider is `aws` or unset, BR requires
`aws:kms` or `aws:kms:dsse` with a context because S3 requires an explicit
SSE-KMS algorithm for the context header. Configure the desired algorithm and
key explicitly. Operator leaves omitted fields unset.

For cross-account SSE-KMS, specify the complete ARN of a customer-managed KMS
key. An alias can resolve to a key in the requester's account. When S3 Bucket
Keys are enabled, S3 uses the bucket ARN for the default `aws:s3:arn`
encryption-context pair and makes fewer calls to KMS. Review any key policy
conditions written for object ARNs before enabling S3 Bucket Keys. See
[Using SSE-KMS](https://docs.aws.amazon.com/AmazonS3/latest/userguide/UsingKMSEncryption.html)
and [S3 Bucket Keys](https://docs.aws.amazon.com/AmazonS3/latest/userguide/bucket-key.html).

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
  s3:
    provider: aws
    region: us-east-1
    bucket: my-backup-bucket
    prefix: snapshots/example
    serverSideEncryption:
      algorithm: aws:kms
      kmsKeyId: arn:aws:kms:us-east-1:123456789012:key/REPLACE-ME
      kmsEncryptionContext:
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
    s3:
      provider: aws
      bucket: my-backup-bucket
      region: us-east-1
      prefix: scheduled-snapshots
      serverSideEncryption:
        algorithm: aws:kms
        kmsKeyId: arn:aws:kms:us-east-1:123456789012:key/REPLACE-ME
        kmsEncryptionContext:
          values:
            tenant: tenant-a
  logBackupTemplate:
    backupMode: log
    toolImage: your-registry/br:sse-kms-context
    br:
      cluster: source-tidb
    s3:
      provider: aws
      bucket: my-backup-bucket
      region: us-east-1
      prefix: logs
      serverSideEncryption:
        algorithm: aws:kms
        kmsKeyId: arn:aws:kms:us-east-1:123456789012:key/REPLACE-ME
        kmsEncryptionContext:
          values:
            tenant: tenant-a
```

BR saves the encryption settings when it starts a log backup task. Existing
tasks keep those settings when you edit a Backup or BackupSchedule, upgrade its
binaries, or pause and resume the task. Plan for log coverage when recreating a
task to change its settings.

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
  s3:
    provider: aws
    bucket: my-backup-bucket
    region: us-east-1
    prefix: snapshots/example
    serverSideEncryption:
      algorithm: aws:kms
      kmsKeyId: arn:aws:kms:us-east-1:123456789012:key/REPLACE-ME
      kmsEncryptionContext:
        values:
          tenant: tenant-a
          purpose: backup
```

For PITR, configure the log storage under `spec.s3` and any separate full backup
storage under `spec.pitrFullBackupStorageProvider.s3`. Operator passes the log
storage settings as `--s3.*` flags. It encodes the full backup storage settings
in the `--full-backup-storage` URL as `sse`, `sse-kms-key-id`, and
`sse-kms-context` query parameters. When the full backup storage explicitly
sets `serverSideEncryption`, Operator emits all three URL parameters. An
unset field becomes an empty parameter after resolving deprecated fields and
clears the matching main storage flag for the full backup storage. When the
full backup storage has no deprecated encryption fields, `algorithm: AES256`
clears any inherited KMS key and context and `serverSideEncryption: {}` clears
all three encryption settings.

If the full backup storage omits `serverSideEncryption`, matching main storage
flags can still be inherited. A standalone legacy `s3.sse` adds no URL
parameter. Legacy `s3.sseKmsEncryptionContext` sets the context parameter and
also passes the legacy algorithm if supplied.

```yaml
spec:
  s3:
    provider: aws
    bucket: my-log-bucket
    prefix: logs
    serverSideEncryption:
      algorithm: aws:kms
      kmsKeyId: alias/log-backup
      kmsEncryptionContext:
        values:
          purpose: logs
  pitrFullBackupStorageProvider:
    s3:
      provider: aws
      bucket: my-snapshot-bucket
      prefix: snapshots
      serverSideEncryption:
        algorithm: aws:kms
        kmsKeyId: alias/snapshot-backup
        kmsEncryptionContext:
          values:
            purpose: snapshots
```

Read access still depends on the existing S3 and KMS permissions. These settings
configure BR storage. S3 reads use the existing read path without an
encryption-context request header. See the
[AWS GetObject documentation](https://docs.aws.amazon.com/AmazonS3/latest/API/API_GetObject.html).

## CompactBackup

Set `spec.s3.serverSideEncryption` on a CompactBackup to pass its algorithm,
KMS key, and context to `br operator base64ify`. Operator passes the resulting
storage configuration to `tikv-ctl compact-log-backup`. The compaction tool
images must support the configured encryption settings.
For scheduled compaction, put the same S3 configuration under
`spec.compactBackupTemplate.s3` in the BackupSchedule.

## Compatibility and validation

- The deprecated `spec.s3.sse` and `spec.s3.sseKmsEncryptionContext` fields
  still supply the algorithm and context during an upgrade. Migrate to
  `serverSideEncryption.algorithm` and
  `serverSideEncryption.kmsEncryptionContext`. Kubernetes CEL admission and
  the Operator controllers reject differing nonempty algorithms or differing
  context maps when both context fields are set. Equal algorithms and equal
  context maps are accepted, including two empty maps. A legacy algorithm
  can be combined with a structured key or context, and a legacy context can
  be combined with a structured algorithm or key.
- With no deprecated fields set, omitting `serverSideEncryption` or setting
  it to `{}` adds no flags for the main storage.
  `kmsEncryptionContext: {values: {}}` sends the explicit empty JSON object
  `{}`.
- `values` is required when `kmsEncryptionContext` or its deprecated
  counterpart is present. Every context value must be a string. Quote numeric
  strings in YAML, for example `tenant: "123"`.
- Kubernetes removes null fields before validation. `values: {tenant: null}`
  becomes `values: {}`, and `kmsEncryptionContext: null` is treated as absent.
  `kmsEncryptionContext: {values: null}` fails validation because the required
  `values` field is missing after pruning. BR rejects null values in raw JSON
  supplied through `spec.br.options`.
- For the main storage in Backup and Restore, `spec.br.options` has the highest
  priority over generated flags. Explicit `--s3.sse`, `--s3.sse-kms-key-id`,
  and `--s3.sse-kms-context` flags override the corresponding generated flags.
  Both `--flag=value` and separate flag/value arguments work. BR applies its
  existing URL option precedence if an explicit storage URL carries encryption
  settings.

## Workflows that do not use these fields

- EBS VolumeSnapshot metadata is written directly with `s3blob.WriteAll`.
  Those metadata writes carry no SSE headers from these fields. A bucket
  policy requiring explicit SSE headers or a custom KMS context can reject
  the metadata write and fail the backup.
- Dumpling and rclone exports and Lightning imports do not use these fields.
  `spec.s3.options` continues to configure rclone.
- Federation VolumeBackup, VolumeRestore, and VolumeBackupSchedule share the
  S3 schema but their metadata I/O does not use these fields.

## S3 authentication for backup and restore tasks

On a BR-based Backup or Restore, set `spec.s3.roleArn` to the IAM role for
the main S3 storage. Set `spec.s3.externalId` when the role's trust policy
requires an external ID. BR uses the role for storage access, and backup-manager
uses it for backup metadata reads, restore metadata reads, and BR backup cleanup.
The fields also work under `spec.backupTemplate.s3` and `spec.logBackupTemplate.s3`
in a BackupSchedule. Keep credentials and other sensitive values out of `externalId`.

```yaml
spec:
  br:
    sendCredToTikv: false
  s3:
    provider: aws
    region: us-east-1
    bucket: my-backup-bucket
    prefix: snapshots/example
    roleArn: arn:aws:iam::123456789012:role/backup
    externalId: customer-backup
```

The task's existing credentials are the source for AWS STS AssumeRole. The role
must have the permissions required by the enabled backup, restore and cleanup
operations. Environment credentials may include a session token. Temporary role
credentials are refreshed by the AWS SDK; source credentials must remain valid
or refreshable for the task's duration. When a role is specified, failures to
assume it or access S3 are reported without retrying under the environment
identity. Without a role, existing credential behavior is preserved.

Set `spec.br.sendCredToTikv: false` when the target role's trust policy does
not allow it to assume itself or when running a long-running log backup.
Every participating TiKV node then needs refreshable source credentials,
network access to AWS STS, and `sts:AssumeRole` permission for the target role.

The existing `spec.br.options` flags `--s3.role-arn` and `--s3.external-id`
override their matching fields independently for main storage. Both `--flag=value`
and separate flag/value arguments are accepted. Repeated flags use the last
value. `--s3.role-arn=` disables AssumeRole, and `--s3.external-id=` clears the
external ID while keeping the selected role. An external ID takes effect only
when a role is selected. Metadata and cleanup clients also apply `--s3.region`
and `--s3.endpoint` overrides. STS uses its own endpoint.

Backup-manager reads the location from the task's storage fields. BR handles
any replacement `--storage` URL and credentials or profiles embedded in it.
Configure the location and role through the task's storage fields and the
documented BR flags so that both tools can use the same settings. EBS
volume-snapshot workflows use their existing credentials to read and write cloud
snapshot metadata. Controller-owned storage operations, Dumpling, Lightning and
rclone keep their existing authentication configuration.

For PITR, `spec.s3` configures log storage. Put the full backup's role and
external ID under `spec.pitrFullBackupStorageProvider.s3`. Operator encodes
those fields in the `--full-backup-storage` URL. BR applies the full-backup
URL fields to that storage alone. Omitted fields use BR's existing defaults,
which can inherit the main S3 flags. Existing log backup tasks keep their
saved authentication settings. Verify the stored task configuration and
supported upgrade procedure before changing credentials or roles for an
existing task.

Deployments that previously used one identity for BR and another for metadata
or cleanup now use the configured role for the operations listed above. Review
that role's task permissions when upgrading. Validate with default source
credentials that cannot directly read or delete the protected test data, so an
omitted role cannot be hidden by environment permissions.

## Logging and validation scope

[Encryption context is not encrypted](https://docs.aws.amazon.com/AmazonS3/latest/userguide/UsingKMSEncryption.html#encryption-context) and is not a secret store. It remains visible in Kubernetes
objects and must not contain credentials or other sensitive information.
Operator omits the full CR from startup logs and redacts context and S3 URL credentials in its own BR
command logs and startup errors, including context supplied through options or
storage URLs. It still forwards BR/TiKV output; those binaries must independently
handle their own log redaction.

Unit tests cover the generated CRD schemas, legacy compatibility, conflicting
algorithms, context serialization, BR option precedence, scheduled backup
copying, compaction argument generation, and command-log redaction. Before
deployment, also run an Operator-managed snapshot backup and restore and a log
backup and PITR against a dedicated S3/KMS test setup. Verify restored data,
task continuation after restart, and policy rejection when the required context
is absent or wrong. These checks require a Kubernetes and AWS environment.
