# Replication Restore: Operator 侧控制流设计

> 本文是 CompactBackup sharded 子特性（已合入 master）之后、Restore 侧承接工作的正式设计。以 2026-04-15 v2 design 为骨架，在状态写权（Option B）、命名、字段默认值上做了 5 处局部修正。

## Summary

跨 region 灾备场景下，复制上游的 log compaction 会产生等量的跨 region 传输费用。为减少这份开销，下游 PITR 恢复时执行 log compaction，并通过多节点并行分片控制恢复时间。

CompactBackup sharded 子特性已提供"下游并行 compaction"能力。本次设计承接 Restore 侧：**不新增 CRD、不新增 RestoreMode**，复用现有 Restore（`mode=pitr`）+ CompactBackup CR，通过在 Restore spec 中加一个可选 `ReplicationConfig` 字段触发两阶段 BR 恢复（phase-1 snapshot restore / phase-2 log restore），并以 CompactBackup 终态作为从 phase-1 到 phase-2 的并发门控。

## Motivation

### Goals

- 在 PITR 恢复时才执行 log compaction，避免跨 region 传输 compact 后的数据
- 支持与 BR phase-1（snapshot restore + 设置断点）并行执行 compaction
- compaction 任务全部到达终态后（无论成功或失败），继续执行 BR phase-2（log restore）
- 对标准 PiTR / snapshot / volume-snapshot 用户零感知

### Non-Goals

- 不引入新 CRD、不引入新 RestoreMode
- 不修改 BR / tikv-ctl 的**内部代码**，只使用其已支持的 CLI 参数（`--replication-storage-phase`、`--replication-status-sub-prefix`、`--pitr-concurrency`）
- 不涉及上游 log compaction 的调度（本设计只覆盖下游）
- 不修改现有 PiTR、snapshot、volume-snapshot 代码路径

## Proposal

### User Story

用户分别创建 CompactBackup（`mode: sharded`）和 Restore（`mode: pitr` + `replicationConfig`），Operator 自动编排两阶段恢复：

```yaml
# Step 1: CompactBackup（本次不改，CompactBackup sharded PR 已落地）
apiVersion: pingcap.com/v1alpha1
kind: CompactBackup
metadata: { name: dr-compact, namespace: tidb-cluster }
spec:
  mode: sharded
  shardCount: 3
  concurrency: 4
  maxRetryTimes: 6
  br: { cluster: downstream-cluster, clusterNamespace: tidb-cluster }
  s3: { bucket: log-backup, region: us-west-2, prefix: /log }
  startTs: "409054741514944513"
  endTs: "18446744073709551615"
---
# Step 2: Restore with replicationConfig（本次设计新增的用户接口）
apiVersion: pingcap.com/v1alpha1
kind: Restore
metadata: { name: dr-restore, namespace: tidb-cluster }
spec:
  mode: pitr
  br: { cluster: downstream-cluster, clusterNamespace: tidb-cluster }
  s3: { bucket: log-backup, region: us-west-2, prefix: /log }
  pitrFullBackupStorageProvider:
    s3: { bucket: full-backup, region: us-west-2, prefix: /full }
  pitrRestoredTs: "409054741514944513"
  replicationConfig:
    compactBackupName: "dr-compact"    # 必填
    # waitTimeout: "30m"               # 可选，默认 0（无限等待）
```

### Risks and Mitigations

| 风险 | 缓解 |
|------|------|
| 用户拼写 `compactBackupName` 错误 | `waitTimeout` 兜底；默认 0 意味着"默认无超时"，用户可显式设 |
| CompactBackup 部分 shard 失败 | 内核保证 BR phase-2 可回退到未 compact 的原始日志（退化到慢路径，语义正确）；operator 不特殊处理 |
| BR phase-1 / phase-2 Job 失败 | Restore 进入 `Failed`，用户查 Job 日志，手动重建 |
| K8s 版本需 ≥ 1.29（Indexed Job） | CompactBackup 侧已做 server version check；Restore 侧无 Indexed Job，无此约束 |

## Design Details

### 1. CR Shape（RestoreSpec 扩展）

新增一个可选 struct，**不新增** `StatusSubPrefix` 字段（固定常量 `"ccr"`，见节 6）：

```go
type RestoreSpec struct {
    // ... 现有字段保留 ...

    // ReplicationConfig is the optional configuration for replication restore.
    // When Mode == pitr and this field is non-nil, the controller runs a
    // two-phase replication restore. When nil, the controller runs a standard
    // PiTR restore (existing behavior unchanged).
    // +optional
    ReplicationConfig *ReplicationConfig `json:"replicationConfig,omitempty"`
}

type ReplicationConfig struct {
    // CompactBackupName references a CompactBackup CR in the same namespace.
    CompactBackupName string `json:"compactBackupName"`

    // WaitTimeout bounds how long the controller waits for a missing
    // CompactBackup CR to appear. 0 means wait indefinitely.
    // Does NOT bound waiting for an existing CompactBackup to finish.
    // +optional
    WaitTimeout *metav1.Duration `json:"waitTimeout,omitempty"`
}
```

**触发规则**：`Spec.Mode == pitr && Spec.ReplicationConfig != nil` → replication restore；`nil` → 标准 PiTR，现有行为不变。

**与 v2 相对的修正**：
- ❌ 移除 v2 里的 `StatusSubPrefix string` 字段（固定常量，不暴露给用户）
- 其余字段与 v2 一致

### 2. State Machine

新增 2 个 Phase 值（驱动 `status.phase`）+ 2 个 Condition marker（并列不改 Phase）：

```go
// Phase 值
RestoreSnapshotRestore RestoreConditionType = "SnapshotRestore"   // phase-1 运行中
RestoreLogRestore      RestoreConditionType = "LogRestore"        // phase-2 运行中

// Condition marker
RestoreSnapshotRestored RestoreConditionType = "SnapshotRestored" // phase-1 Job 完成
RestoreCompactSettled   RestoreConditionType = "CompactSettled"   // CompactBackup 终态
```

**新增 Status 字段**：

```go
type RestoreStatus struct {
    // ... 现有字段 ...

    // ReplicationStep identifies the current phase of replication restore.
    // Values: "" (not replication), "snapshot-restore", "log-restore".
    // Set by the controller when creating each phase's Job.
    // +optional
    ReplicationStep string `json:"replicationStep,omitempty"`
}
```

**状态机**：

```
[初始] → Scheduled → SnapshotRestore → LogRestore → Complete
                            ↓              ↓
                         Failed          Failed
```

门控：`SnapshotRestored=True && CompactSettled=True` 时 controller 在同一次 reconcile 内 `SnapshotRestore → LogRestore` 并创建 phase-2 Job（**不引入中间状态** `ReadyForLogRestore`，两个 marker 同时为真即是门控凭据）。

**与 v2 相对的修正**：
- v2 `Phase1Running/Phase1Complete` → **`SnapshotRestore`**（单段，无中间态）→ `LogRestore`
- v2 `BRPhase1Complete` marker → **`SnapshotRestored`**
- v2 `ReplicationStep int32`（值 1 / 2）→ **`ReplicationStep string`**（值 `"snapshot-restore"` / `"log-restore"`）
- 命名理由：意图化（用户读 `status.phase` 能理解当前阶段在做什么），对齐 repo 的 `feedback_naming_intent` 偏好

### 3. Status Write Ownership（Option B）

**Controller 接管所有 `status.phase` 写入**，backup-manager 在 phase-1 期间状态写入被 wrapper 抑制。

| 写内容 | 写者 | 写法 |
|--------|------|------|
| `status.phase = SnapshotRestore` | Controller | `replicationHandler.Sync` 入口 |
| `SnapshotRestored` marker | Controller | 观测 phase-1 Job `JobComplete` 条件触发 |
| `CompactSettled` marker | Controller | 观测 CompactBackup `status.state ∈ {Complete, Failed}` 触发 |
| `status.phase = LogRestore` | Controller | 两个 marker 同 True 时 |
| phase-2 `Running / Complete / Failed` | backup-manager | 直写（**不 wrap**） |
| orchestration 级 `Failed` | Controller | waitTimeout 到期 / 跨 CR 不一致 / phase-1 Job Failed |

**Wrapper**（对照 `ShardedCompactStatusUpdater`）：

```go
// pkg/controller/restore_status_updater.go
type ReplicationRestoreStatusUpdater struct {
    real RestoreConditionUpdaterInterface
}

// 仅放行 OnStart（写 Event，不改 Phase）；其余抑制
func (r *ReplicationRestoreStatusUpdater) OnStart(...) error   { return r.real.OnStart(...) }
func (r *ReplicationRestoreStatusUpdater) Update(...) error    { return nil }
func (r *ReplicationRestoreStatusUpdater) OnProgress(...) error { return nil }
func (r *ReplicationRestoreStatusUpdater) OnFinish(...) error   { return nil }
```

**激活条件**：`cmd/backup-manager/app/cmd/restore.go` 中 `--replicationPhase == 1` → 包 wrapper；`== 2` 或未设 → 不包。

**UpdateRestoreCondition 保持零改动**。v2 需要修改该函数加转化分支，Option B 通过"controller 直写 + wrapper 抑制"的组合从根上消除了转化需求。

**与 v2 相对的修正**：Option B 放弃 v2 "backup-manager 源码零改动" 的承诺，换取写权清晰和消除 v2 状态机破洞（v2 的转化层只转 `Complete` 不转 `Running`，会导致 phase-1 期间 `status.phase` 从 `Phase1Running` 闪跳到 `Running`，与 state diagram 不符）。结构上与 CompactBackup sharded 的 `ShardedCompactStatusUpdater` 对称，复用同一心智模型。

### 4. Orchestration Control Flow

#### 4.1 拦截点

`pkg/backup/restore/restore_manager.go` 的 reconcile 入口，通用校验之后、现有 mode 分支之前：

```go
if restore.Spec.Mode == v1alpha1.RestoreModePiTR && restore.Spec.ReplicationConfig != nil {
    return rm.replicationHandler.Sync(restore)
}
// 以下为现有 mode 分支（snapshot / volume-snapshot / 标准 pitr），不动
```

#### 4.2 Handler 状态机（按 `status.phase` 分派）

```
replicationHandler.Sync(restore):
    switch restore.Status.Phase:

    case "" | Scheduled:
        ① 查 CompactBackup:
           - NotFound + WaitTimeout 到期 → Phase = Failed (CompactBackupWaitTimeout)
           - NotFound + 未超时           → requeue + Warning Event
           - Found + state ∈ {Complete, Failed}
                                        → 一致性校验（节 5），不通过则 Phase = Failed
                                        → 校验通过则写 CompactSettled marker
           - Found + 未终态              → 不写 CompactSettled，正常继续
        ② 写 Phase = SnapshotRestore, ReplicationStep = "snapshot-restore"
        ③ 按 label 查 phase-1 Job；不存在则创建
        return

    case SnapshotRestore:
        ① 刷新 phase-1 Job 观测:
           - JobComplete → 写 SnapshotRestored marker
           - JobFailed   → 写 Phase = Failed (reason: SnapshotRestoreFailed)
        ② 刷新 CompactBackup 观测:
           - state ∈ {Complete, Failed} → 写 CompactSettled marker
        ③ 门控: 两个 marker 都 True
           → 写 Phase = LogRestore, ReplicationStep = "log-restore"
           → 按 label 查 phase-2 Job；不存在则创建
        return

    case LogRestore:
        ① 刷新 phase-2 Job 观测（沿用现有 PiTR Job 观测逻辑）
        ② backup-manager 直接写 Running/Complete/Failed，handler 不干预
        return

    case Complete | Failed:
        return  // 终态
```

#### 4.3 Job 识别：label 而非 name

每个 Job 打 3 个关键 label：

```
tidb.pingcap.com/restore-name       = {restore.Name}                       # 现有
app.kubernetes.io/component         = restore                              # 现有
tidb.pingcap.com/replication-step   = "snapshot-restore" | "log-restore"   # 新增
```

Job 命名作为辅助可读性（如 `{restore.Name}-snapshot-restore`），但代码**一律通过 label selector 查**，避免未来改名打破幂等。两个 Job 均以 Restore CR 为 OwnerReference。

#### 4.4 CompactBackup informer wire-up

`pkg/controller/restore/restore_controller.go`：注入 CompactBackup informer，注册 EventHandler：

```go
compactInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
    UpdateFunc: func(old, cur interface{}) { c.enqueueRestoresReferencing(cur) },
    // Add / Delete 复用同一路径
})

// 小规模假设：同 namespace Restore 数量 O(10)，线性扫描 lister 即可
func (c *Controller) enqueueRestoresReferencing(obj interface{}) {
    cb := obj.(*v1alpha1.CompactBackup)
    restores, _ := c.restoreLister.Restores(cb.Namespace).List(labels.Everything())
    for _, r := range restores {
        if r.Spec.ReplicationConfig != nil &&
            r.Spec.ReplicationConfig.CompactBackupName == cb.Name {
            c.queue.Add(key(r))
        }
    }
}
```

#### 4.5 重入恢复

Controller 重启后：读 `status.phase` 决定分支 → label selector 查现有 Job → 创建前再查一次幂等。无任何内存状态。

### 5. Cross-CR Binding

**查找**：按 `spec.replicationConfig.compactBackupName` 在同 namespace 下 `Lister.Get`。

**一致性校验**（作为写入 `CompactSettled` marker 的前置条件——handler 发现 CompactBackup 已到终态后、先校验成功才写 `CompactSettled`；校验不通过直接 `Failed`。`CompactSettled=True` 隐含了"一致性已通过"，不需要额外 marker）：

| 校验项 | 来源字段 | 比较方式 |
|--------|----------|----------|
| 目标集群 | `spec.br.cluster`、`spec.br.clusterNamespace` | 字符串等值 |
| 日志备份存储位置 | `spec.s3/gcs/azblob` | 比较位置字段（bucket+prefix+region；gcs: bucket+prefix；azblob: container+prefix）；**忽略** `secretName` |

不一致 → `Phase = Failed`, reason `CompactBackupMismatch`，message 标注不一致的字段名。

**Late binding**：
- CompactBackup 不存在 + 未超时 → requeue + Warning Event
- CompactBackup 不存在 + `WaitTimeout > 0 && now - Restore.CreationTimestamp > WaitTimeout` → `Phase = Failed`, reason `CompactBackupWaitTimeout`
- CompactBackup 存在但未终态 → 持续等，不受 `WaitTimeout` 影响（compaction 耗时由业务决定）

**Delete 语义**：
- Restore 删除 → OwnerRef 级联删除两个 BR Job；CompactBackup 不受影响
- CompactBackup 删除：
  - 已写过 `CompactSettled` marker → handler 不再查询，删除无影响
  - 未 settle → 下次 reconcile NotFound → 走 late-binding；若超过 `WaitTimeout` → `Failed`
- 多 Restore 引用同一 CompactBackup → 允许

### 6. BR CLI Surface（backup-manager 端）

**新增 1 个 CLI flag**：

```
--replicationPhase int   # 1 或 2；0/未设 = 标准 PiTR
```

**硬编码常量**（`cmd/backup-manager/app/restore/restore.go`）：

```go
const (
    replicationStatusSubPrefix = "ccr"
    replicationPiTRConcurrency = 1024
)
```

**BR 命令追加**（仅 `opts.ReplicationPhase > 0` 时）：

```
--replication-storage-phase=<1|2>
--replication-status-sub-prefix=ccr
--pitr-concurrency=1024
```

**Controller 侧写入**：创建 BR Job 时将 `--replicationPhase={1|2}` 作为 Pod args 传给 backup-manager。`replicationConfig.compactBackupName` 等字段**不传给** backup-manager——它不需要知道 CompactBackup 的存在。

**常量放 backup-manager 侧的原因**：这两个值是 BR 调用细节，放到 types.go 会把实现细节泄漏到 API。

### 7. Module Layout

| 文件 | 性质 | 改动 |
|------|------|------|
| `pkg/apis/pingcap/v1alpha1/types.go` | 改 | 加 `ReplicationConfig` struct、`RestoreSpec.ReplicationConfig`、`RestoreStatus.ReplicationStep`、4 个 `RestoreConditionType` 常量 |
| `pkg/apis/pingcap/v1alpha1/zz_generated.deepcopy.go` | 生成 | `make generate` |
| `pkg/apis/pingcap/v1alpha1/openapi_generated.go` | 生成 | 同上 |
| `manifests/crd/v1/pingcap.com_restores.yaml`、`manifests/crd.yaml` | 生成 | `make manifests` |
| `pkg/apis/label/label.go` | 改 | 加 `ReplicationStepLabelKey` 和两个值常量 |
| `pkg/backup/restore/replication_handler.go` | **新增** | Handler 实现（节 4）|
| `pkg/backup/restore/restore_manager.go` | 改 | 拦截点；注入 `replicationHandler` 依赖 |
| `pkg/controller/restore_status_updater.go` | 改 | 新增 `ReplicationRestoreStatusUpdater` wrapper |
| `pkg/controller/restore/restore_controller.go` | 改 | 注入 CompactBackup informer + EventHandler |
| `cmd/backup-manager/app/cmd/restore.go` | 改 | 加 `--replicationPhase` flag；`== 1` 时 wrap updater |
| `cmd/backup-manager/app/restore/restore.go` | 改 | PiTR 分支追加 3 个 BR flag（仅 `ReplicationPhase > 0`）；定义两个常量 |
| `cmd/backup-manager/app/restore/manager.go` | 改（若需要） | `Options.ReplicationPhase int` 字段 |

**Handler 放 `pkg/backup/restore/` 的原因**：该目录承担"reconcile 一次 Restore CR 的业务编排"职责；`replication_handler` 内容属于业务编排，与 `restore_manager.go` 同包同层更合理；拦截点是**同包函数调用**，不需要跨包 interface。

**wrapper 放 `pkg/controller/` 的原因**：对称于 `pkg/controller/compact_status_updater.go` 里的 `ShardedCompactStatusUpdater`，评审时建立"同一模式的第二次应用"论点。

### 8. Backward Compatibility

| 兼容面 | 保证 |
|--------|------|
| CRD 旧对象 | `ReplicationConfig`、`ReplicationStep` 均 optional，零值等价旧行为 |
| 标准 PiTR 代码路径 | `replicationConfig == nil` → controller 不走新分支；`--replicationPhase` 未设 → backup-manager 不包 wrapper 不追加 BR flag |
| snapshot / volume-snapshot 模式 | 拦截点条件 `Mode == pitr`，其他模式完全不受影响 |
| `UpdateRestoreCondition` | **零改动**（Option B 的核心收益） |

**打破 v2 的一条承诺**：v2 承诺 "backup-manager 源码零改动"。Option B 下 backup-manager 需加 1 个 CLI flag + 1 个条件 wrap 装配，但：
- 用户视角行为零变化（未设 flag 即旧行为）
- 换来的是消除 v2 的状态机破洞（phase-1 期间 `status.phase` 抖动）

## Test Plan

### 单元测试

| 对象 | 测试点 |
|------|--------|
| `replicationHandler.Sync` | `""/Scheduled → SnapshotRestore`；`SnapshotRestore → LogRestore`（两 marker 同 True）；`SnapshotRestore → Failed`（phase-1 Job Failed / waitTimeout / 一致性校验失败）；`LogRestore` 走现有 PiTR 观测 |
| 门控判定 | 两 marker 都 True 才放行；到达顺序无关；单个 marker True 继续等 |
| 跨 CR 一致性校验 | cluster / clusterNamespace 不一致 → Failed；storage 位置不一致 → Failed；忽略 `secretName` 差异；校验只跑一次 |
| `ReplicationRestoreStatusUpdater` | 仅放行 `OnStart`；`Update/OnProgress/OnFinish` 一律 no-op；wrapper 未激活时行为等同直连 |
| Label-based Job 查找 | 同一 replication-step 的 Job 已存在时 `Sync()` 不重复创建（幂等）|
| Re-entrancy | 给定 `(status.phase, conditions, 现存 Job)` 各组合，handler 下一步推导确定 |

### 集成测试场景

| 场景 | 预期 |
|------|------|
| phase-1 Job 成功 + CompactBackup Complete | `LogRestore` Job 启动 → `Complete` |
| phase-1 Job 成功 + CompactBackup Failed（部分/全部 shard 失败）| `LogRestore` Job 仍启动 → `Complete`（BR 回退未 compact 日志）|
| phase-1 Job 成功 + CompactBackup Running | 停留 `SnapshotRestore`，等两 marker |
| phase-1 Job 失败 | Restore `Failed`（不等 CompactBackup）|
| phase-2 Job 失败 | Restore `Failed` |
| `compactBackupName` 拼写错误 + `waitTimeout=30m` 到期 | Restore `Failed`, reason `CompactBackupWaitTimeout` |
| 跨 CR 存储/集群配置不一致 | Restore `Failed`, reason `CompactBackupMismatch` |
| phase-1 期间 `status.phase` 不抖动（恒为 `SnapshotRestore`）| 验证 wrapper 消除 v2 的状态机破洞 |
| Controller 重启恢复 | kill controller Pod 后 Restore 从 `SnapshotRestore` 中断态继续走完 |

## PR Breakdown

按"单 spec / 多 PR"策略，切成 2 个 PR：

### PR 1: Operator 侧完整实现（CRD + 逻辑 + label + 单测）

- `pkg/apis/pingcap/v1alpha1/types.go` — `ReplicationConfig` struct、`RestoreSpec.ReplicationConfig`、`RestoreStatus.ReplicationStep`、4 个 `RestoreConditionType` 常量
- 生成文件刷新：`zz_generated.deepcopy.go`、`openapi_generated.go`、`manifests/crd/v1/pingcap.com_restores.yaml`、`manifests/crd.yaml`
- `pkg/apis/label/label.go` — `ReplicationStepLabelKey` 和两个值常量
- `pkg/controller/restore_status_updater.go` — 新增 `ReplicationRestoreStatusUpdater` wrapper
- `pkg/backup/restore/replication_handler.go` — 新增 Handler（状态机分派、双 Job 管理、门控判定、跨 CR 查找/校验）
- `pkg/backup/restore/restore_manager.go` — 拦截点（通用校验之后、mode 分支之前）
- `pkg/controller/restore/restore_controller.go` — 注入 CompactBackup informer + EventHandler
- 配套单测

**目标**：operator 端全部逻辑就绪。此时 backup-manager 未改，端到端流程还跑不通（BR 命令没追加新 flag），但 operator 侧的类型、状态机、wrapper、handler、informer 全部可以在单测层面完整验证。把类型和消费者放在同一 PR 里评审，避免"字段放哪里用？"这种空 PR 的问题。

### PR 2: backup-manager CLI + BR 参数

- `cmd/backup-manager/app/cmd/restore.go` — 加 `--replicationPhase` flag；`== 1` 时 wrap status updater
- `cmd/backup-manager/app/restore/restore.go` — PiTR 分支追加 3 个 BR flag（仅 `ReplicationPhase > 0`）；定义 `replicationStatusSubPrefix = "ccr"` 和 `replicationPiTRConcurrency = 1024` 常量
- `cmd/backup-manager/app/restore/manager.go` — `Options.ReplicationPhase int` 字段（若需要）
- 集成测试

**目标**：端到端可用。PR 2 必须独立，因为：
1. 依赖内核 BR 版本包含 `--replication-storage-phase` 等 flag，发布节奏受内核约束
2. 集成测试需要真实内核 BR 环境，和 operator 侧单测的运行条件不同
3. 即便 PR 1 合了、PR 2 暂时不合，operator 创建的 `replicationConfig` Restore 也能走到 `SnapshotRestore` phase（只是 BR Job 没拼对参数），用户能观察到清晰的停滞位置

## Drawbacks

- Restore controller 引入 replication 分支逻辑和 CompactBackup informer；controller 包依赖面扩大
- 用户需分别创建两个 CR（CompactBackup + Restore），操作步骤多于单 CRD 方案
- `compactBackupName` 是跨 CR 的名字引用，拼写错误需要 `waitTimeout` 兜底

## Alternatives

见 v2 design 的 Alternatives 章节（被否决的 `ReplicationRestore` CRD、per-shard CompactBackup CR、新 RestoreMode `replication`、label 反向发现等方案）。本次设计未改变这些结论。

**本次新增的替代选择及结论**：

- **Option A（v2 转化层路径）**：保持 backup-manager 源码零改动，`UpdateRestoreCondition` 加转化分支。**被否决**，因为 v2 的转化逻辑未覆盖 `Running` 写入，会导致 phase-1 期间 `status.phase` 抖动；补洞会让单函数变复杂。
- **选定 Option B**：controller 直写 + wrapper 抑制，`UpdateRestoreCondition` 零改动，对称 CompactBackup sharded 已有模式。

## Open Questions

- `compactBackupName` 跨 namespace 引用是否需要支持？当前设计限同 namespace（与 BR 引用 TiDBCluster 的模式一致）。若将来有跨 namespace 需求，可扩展为 `{Namespace, Name}`。
- `statusSubPrefix` 常量 `"ccr"` 是否需要可配置？当前硬编码；若未来发现不同 replication 流需要隔离，再按需扩展 CRD。
