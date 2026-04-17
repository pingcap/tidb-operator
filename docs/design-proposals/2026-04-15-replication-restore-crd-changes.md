# Replication Restore: CRD 字段变更

本文档列出 replication restore 功能需要对 Restore CR 和 CompactBackup CR 新增的字段，以及完整的 CR 示例。

关联文档：
- [整体设计](2026-04-15-replication-restore-v2.md)
- [状态模型](2026-04-15-replication-restore-status-model.md)

---

## Restore CR

### Spec 新增字段

以下字段添加到 `RestoreSpec`（`pkg/apis/pingcap/v1alpha1/types.go`）：

Replication restore 复用 PiTR 的现有字段（`storageProvider`、`pitrFullBackupStorageProvider`、`pitrRestoredTs`、`br` 等），不重复新增。仅新增一个可选的 `ReplicationConfig` struct：

```go
type RestoreSpec struct {
    // ... 现有字段全部保留 ...
    // storageProvider (inline)          → 复用：日志备份存储
    // pitrFullBackupStorageProvider     → 复用：全量备份存储
    // pitrRestoredTs                   → 复用：恢复目标时间点
    // br                              → 复用：集群引用

    // ReplicationConfig is the optional configuration for replication restore.
    // When mode=pitr and this field is set, the controller runs a two-phase
    // replication restore (phase-1 with compaction gate, then phase-2 log restore).
    // When nil, the controller runs a standard PiTR restore (existing behavior).
    // +optional
    ReplicationConfig *ReplicationConfig `json:"replicationConfig,omitempty"`
}

// ReplicationConfig holds the replication-specific configuration for PiTR restore.
type ReplicationConfig struct {
    // CompactBackupName is the name of the CompactBackup CR that this Restore depends on.
    // The controller waits for the referenced CompactBackup to reach a terminal state
    // (Complete or Failed) before proceeding to phase-2.
    CompactBackupName string `json:"compactBackupName"`

    // StatusSubPrefix is the sub-prefix for the replication status file.
    // Passed to BR as --replication-status-sub-prefix.
    StatusSubPrefix string `json:"statusSubPrefix"`

    // WaitTimeout is the timeout for waiting for the CompactBackup CR to exist.
    // If the timeout is reached and the CR specified by compactBackupName is not found,
    // the Restore is set to Failed.
    // This timeout does NOT apply when CompactBackup exists but is still running.
    // Default is 0, which means wait indefinitely.
    // +optional
    WaitTimeout *metav1.Duration `json:"waitTimeout,omitempty"`
}
```

**不新增 RestoreMode**。`mode: pitr` + `replicationConfig != nil` 即为 replication restore。

### Status 新增字段

以下字段添加到 `RestoreStatus`（`pkg/apis/pingcap/v1alpha1/types.go`）：

```go
type RestoreStatus struct {
    // ... 现有字段 ...

    // ReplicationStep indicates the current step of replication restore.
    // Only valid when Mode is "replication".
    // Value 1: phase-1 (BR snapshot restore + log compaction).
    //          UpdateRestoreCondition will transform Complete -> BRPhase1Complete.
    // Value 2: phase-2 (BR log restore). No transformation.
    // Set by the controller when creating each phase's Job.
    // +optional
    ReplicationStep int32 `json:"replicationStep,omitempty"`
}
```

### 新增 RestoreConditionType

```go
const (
    // Phase 值（驱动 status.phase 变化）
    RestorePhase1Running   RestoreConditionType = "Phase1Running"
    RestorePhase1Complete  RestoreConditionType = "Phase1Complete"

    // Condition 标记（不驱动 status.phase 变化）
    RestoreBRPhase1Complete RestoreConditionType = "BRPhase1Complete"
    RestoreCompactSettled   RestoreConditionType = "CompactSettled"
)
```

### 完整 Restore CR 示例

```yaml
apiVersion: pingcap.com/v1alpha1
kind: Restore
metadata:
  name: dr-restore
  namespace: tidb-cluster
spec:
  # 复用 PiTR 模式
  mode: pitr

  # 目标 TiDB 集群（复用现有字段）
  br:
    cluster: downstream-cluster
    clusterNamespace: tidb-cluster

  # 日志备份存储（复用现有字段）
  storageProvider:
    s3:
      provider: aws
      bucket: log-backup-us-west-2
      region: us-west-2
      prefix: /log-backup/downstream

  # 全量备份存储（复用现有 PiTR 字段）
  pitrFullBackupStorageProvider:
    s3:
      provider: aws
      bucket: full-backup-us-west-2
      region: us-west-2
      prefix: /full-backup/2026-04-15

  # PiTR 恢复目标时间点（复用现有字段）
  pitrRestoredTs: "409054741514944513"

  # ========== 新增：replication 配置 ==========
  # 传了此字段 → 走 replication restore（phase-1 + compaction 等待 + phase-2）
  # 不传 → 走标准 PiTR restore（现有行为不变）
  replicationConfig:
    compactBackupName: "dr-compact"              # 引用的 CompactBackup CR
    statusSubPrefix: "downstream-checkpoint"     # replication status 子路径前缀
    waitTimeout: "10m"                           # 等待 CompactBackup 出现的超时

  # 通用配置
  toolImage: pingcap/br:v8.5.0
  serviceAccount: tidb-backup-manager
  tolerations:
    - key: dedicated
      operator: Equal
      value: br
      effect: NoSchedule
```

**运行时 Status 示例（阶段 1 进行中）**：

```yaml
status:
  phase: Phase1Running
  replicationStep: 1
  timeStarted: "2026-04-15T10:00:00Z"
  conditions:
    - type: Phase1Running
      status: "True"
      lastTransitionTime: "2026-04-15T10:00:00Z"
```

**运行时 Status 示例（门控通过，阶段 2 进行中）**：

```yaml
status:
  phase: Running
  replicationStep: 2
  timeStarted: "2026-04-15T10:00:00Z"
  conditions:
    - type: Phase1Running
      status: "True"
      lastTransitionTime: "2026-04-15T10:00:00Z"
    - type: BRPhase1Complete
      status: "True"
      lastTransitionTime: "2026-04-15T10:05:00Z"
    - type: CompactSettled
      status: "True"
      lastTransitionTime: "2026-04-15T10:20:00Z"
      reason: ShardsPartialFailed
      message: "2 completed, 1 failed"
    - type: Phase1Complete
      status: "True"
      lastTransitionTime: "2026-04-15T10:20:01Z"
    - type: Running
      status: "True"
      lastTransitionTime: "2026-04-15T10:20:05Z"
```

**运行时 Status 示例（全部完成）**：

```yaml
status:
  phase: Complete
  replicationStep: 2
  timeStarted: "2026-04-15T10:00:00Z"
  timeCompleted: "2026-04-15T10:35:00Z"
  conditions:
    - type: Phase1Running
      status: "True"
      lastTransitionTime: "2026-04-15T10:00:00Z"
    - type: BRPhase1Complete
      status: "True"
      lastTransitionTime: "2026-04-15T10:05:00Z"
    - type: CompactSettled
      status: "True"
      lastTransitionTime: "2026-04-15T10:20:00Z"
      reason: AllShardsComplete
    - type: Phase1Complete
      status: "True"
      lastTransitionTime: "2026-04-15T10:20:01Z"
    - type: Running
      status: "True"
      lastTransitionTime: "2026-04-15T10:20:05Z"
    - type: Complete
      status: "True"
      lastTransitionTime: "2026-04-15T10:35:00Z"
```

---

## CompactBackup CR

### Spec 新增字段

以下字段添加到 `CompactSpec`（`pkg/apis/pingcap/v1alpha1/types.go`）：

```go
type CompactSpec struct {
    // ... 现有字段 ...

    // Mode is the compaction mode.
    // When "sharded", the controller creates an Indexed Job with ShardCount parallel Pods.
    // When empty or unset, the controller creates a regular single-Pod Job (existing behavior).
    // +optional
    // +kubebuilder:validation:Enum:="";sharded
    Mode CompactMode `json:"mode,omitempty"`

    // ShardCount is the number of parallel compaction shards.
    // Required when Mode is "sharded".
    // Each Pod receives JOB_COMPLETION_INDEX as shard index, and runs
    // tikv-ctl compact-log-backup --shard <index>/<shardCount>.
    // Requires Kubernetes >= 1.29 for backoffLimitPerIndex support.
    // +optional
    ShardCount *int32 `json:"shardCount,omitempty"`
}

type CompactMode string

const (
    CompactModeDefault CompactMode = ""
    CompactModeSharded CompactMode = "sharded"
)
```

### Status 新增字段

以下字段添加到 `CompactStatus`（`pkg/apis/pingcap/v1alpha1/types.go`）：

```go
type CompactStatus struct {
    // ... 现有字段 ...

    // CompletedIndexes holds the completed indexes when shardCount > 0.
    // Copied from Job.status.completedIndexes.
    // Format: "0,2-4" or "".
    // +optional
    CompletedIndexes string `json:"completedIndexes,omitempty"`

    // FailedIndexes holds the failed indexes when shardCount > 0.
    // Copied from Job.status.failedIndexes.
    // +optional
    FailedIndexes string `json:"failedIndexes,omitempty"`
}
```

### 完整 CompactBackup CR 示例

```yaml
apiVersion: pingcap.com/v1alpha1
kind: CompactBackup
metadata:
  name: dr-compact                # Restore 通过 spec.replicationConfig.compactBackupName 引用此名字
  namespace: tidb-cluster
spec:
  # 目标集群引用（用于推导 TiKV 镜像版本）
  br:
    cluster: downstream-cluster
    clusterNamespace: tidb-cluster

  # 日志备份存储（tikv-ctl 从此存储读取日志备份数据做 compaction）
  storageProvider:
    s3:
      provider: aws
      bucket: log-backup-us-west-2
      region: us-west-2
      prefix: /log-backup/downstream

  # compact 时间范围
  # startTs: 全量备份的 BackupTS（用户手动从 full backup 的 backupmeta.EndVersion 获取）
  startTs: "409054741514944513"
  # endTs: compact 到最新（u64::MAX）
  endTs: "18446744073709551615"

  # 分片配置
  mode: sharded             # 显式声明分片模式
  shardCount: 3             # 并行 Pod 数
  concurrency: 4          # 每个 shard 内部并发度（tikv-ctl -N 参数）
  maxRetryTimes: 6        # 每个 shard 的重试上限（backoffLimitPerIndex）

  # 通用配置
  toolImage: pingcap/br:v8.5.0
  tikvImage: pingcap/tikv:v8.5.0
  serviceAccount: tidb-backup-manager
  tolerations:
    - key: dedicated
      operator: Equal
      value: br
      effect: NoSchedule
```

**运行时 Status 示例（运行中）**：

```yaml
status:
  state: Running
  progress: "[READ_META(3/10),COMPACT_WORK(1024/4096)]"
```

**运行时 Status 示例（全部成功）**：

```yaml
status:
  state: Complete
  completedIndexes: "0-2"
  failedIndexes: ""
```

**运行时 Status 示例（部分 shard 失败）**：

```yaml
status:
  state: Failed
  message: "1 of 3 shards failed after exhausting retries"
  completedIndexes: "0,2"
  failedIndexes: "1"
  backoffRetryStatus:
    - retryNum: 6
      detectFailedAt: "2026-04-15T10:18:00Z"
      retryReason: "tikv-ctl exit code 1: storage timeout"
```

---

## 字段一致性校验

Restore controller 在门控放行前校验 Restore 和 CompactBackup 的关键字段一致：

| 校验项 | Restore 字段 | CompactBackup 字段 | 不一致时 |
|--------|-------------|-------------------|---------|
| 日志备份存储 | `spec.storageProvider` | `spec.storageProvider` | Failed, reason: CompactBackupMismatch |
| 目标集群 | `spec.br.cluster` | `spec.br.cluster` | Failed, reason: CompactBackupMismatch |

---

## 现有字段兼容性

| CRD | 新增字段 | 默认值 | 对现有行为的影响 |
|-----|----------|--------|-----------------|
| Restore | `replicationConfig` | nil | 无影响，nil 时走标准 PiTR 流程 |
| Restore | `status.replicationStep` | 0 | 无影响，转化层仅在 step=1 时生效 |
| CompactBackup | `mode` | 空（默认单 Job）| 无影响，空值走现有逻辑 |
| CompactBackup | `shardCount` | nil | 无影响，仅 mode=sharded 时使用 |
| CompactBackup | `status.completedIndexes` | 空 | 无影响，仅 shardCount > 0 时填充 |
| CompactBackup | `status.failedIndexes` | 空 | 无影响，仅 shardCount > 0 时填充 |
