# BR 锁 Operation 可观测性适配范围

## Summary

TiDB BR 正在为 external storage lock 增加 operation-aware metadata。每一次 BR
进程执行都会拥有一个 operation ID，并在 lock 文件或 lock conflict 日志中暴露
operation start time、restore ID、lock resource type 等 metadata。

TiDB Operator 不应该生成或替换 BR 的 operation ID。operation ID 属于真正创建
external storage lock 的 BR 执行进程。Operator 的职责是把 BR 已经提供的 operation
和 lock 身份信息透传到相关 CR 上，让运维能够判断哪个 CR 管理过某次 BR operation、
哪个 BR execution 被锁阻塞、哪个 BR execution 创建了锁，以及是否可以安全地手动删除锁
文件。

本文档记录第一阶段适配范围和关键设计决策。最终 CRD 字段名和 BR structured log
字段契约需要在执行阶段基于目标 TiDB/BR PR 的最终 commit 现场确认。

## Motivation

当 BR 进程 crash、被 kill，或者退出时无法完成 cleanup，external storage lock
可能会长期残留。后续 BR operation 被这种锁阻塞时，Kubernetes Job status 往往只会
给出类似 `BackoffLimitExceeded` 的通用失败原因，不足以让运维判断具体涉及哪个 lock
file，也无法判断之前是哪一次 BR 执行创建了锁。

BR 侧的 operation metadata 可以让 lock owner 变得可见。Operator 应该在会使用这些
lock 的流程中，把相关信息暴露到 CR 层面，避免用户第一步就必须去翻 pod log 或远端
存储。

这个能力需要形成闭环：当某个 CR 启动的 BR operation ID 已经可用时，Operator 应尽早把
它记录到该 CR 的 status 中；如果这个 operation 后续残留 lock 并成为其他任务的 blocker，
运维可以通过 blocker operation ID 反查到它来自哪个 CR。当某个 CR 自己被锁阻塞时，
Operator 也需要把 remote blocker 的信息记录到该 CR 上，帮助运维从被阻塞的一侧直接看到
需要排查的 lock owner。

### Goals

- 保持 BR 作为 `operation_id` 的权威来源。
- 明确哪些 Operator CR 需要第一轮适配 BR lock metadata。
- 第一轮聚焦已经确认会使用 BR operation-aware external storage lock 的流程，例如 log
  backup truncate 和 PiTR/log restore。
- 功能 1：BR operation 开始后，backup-manager 从 BR structured log 中读取 operation ID，
  并尽早写入对应 CR status，用于后续 blocker 反查。
- 功能 2：BR execution 失败后，backup-manager 判断失败是否来自 external storage lock 残留或冲突；
  如果是，则从诊断日志中提取 blocker 信息并写入 CR status。
- 为手动删除锁提供清晰的运维指导：只有确认 blocker operation 已经不再运行后，才删除
  对应 lock。

### Non-Goals

- Operator 不生成 BR operation ID。
- Operator 不改变 BR lock ownership 语义。
- 本范围不覆盖所有 backup 或 restore 失败诊断。
- 本范围不要求给 `BackupSchedule` 增加新的 status 字段。
- 本范围不定义最终的 lock metadata CRD schema。

## Decisions

- BR 仍然是 operation ID 的唯一权威来源，Operator 不生成、不覆盖、不合成 BR
  operation ID。
- 第一阶段确定纳入 `Backup spec.mode=log` 的 `log-truncate` 路径，以及 `Restore
  spec.mode=pitr` / replication `log-restore` 路径。其他 log subcommand 和
  `CompactBackup` 先标记为待验证，不作为第一阶段承诺。
- 数据采集点放在 backup-manager 内部。backup-manager 已经在 Backup/Restore Job Pod 内
  启动 BR 子进程并读取 stdout/stderr，因此第一阶段解析 BR 子进程原始输出，不让
  controller-manager 通过 Kubernetes Pod log API 拉日志，也不新增 `pods/log` RBAC。
- 现有 Backup、Restore、CompactBackup CRD 不提供稳定的 BR/backup-manager log directory
  配置入口；backup-manager 自身也以 stderr/container log 为主。因此第一阶段不依赖本地
  日志文件路径，只消费 BR 子进程 stdout/stderr。
- `brOperations` 是最近 BR execution 的有界列表，第一阶段保留最近 10 条。它用于回答
  “这个 CR 启动过哪些 BR operation”，成功后不清空。
- `lockBlocker` 只保留最近一次锁冲突诊断。backup-manager 在执行过程中解析并暂存候选
  blocker 信息；只有本次 BR execution 最终以 lock conflict 失败时才写入 status。BR
  成功或新的非锁失败会清空旧 `lockBlocker`。
- 第一阶段不读取远端 lock JSON，也不假定 Operator 能定位远端 lock 文件路径。远端对象可
  作为后续增强，但当前闭环依赖 BR stdout/stderr structured log 中已经暴露的信息。
- 更稳定的长期接口可以是 BR 输出本地机器可读 report file；第一阶段先沿日志解析方向推进。

## Proposal

### In Scope

#### Backup

`Backup` 的 `log-truncate` 路径是第一阶段确定适配范围。

第一优先级是 `spec.mode=log` 且 subcommand 为 `log-truncate` 的路径。BR 现在用
`truncating.lock` external storage lock 保护 log truncate，并把它标记为 resource
type `log-truncate-exclusive`。当 truncate 因为已有锁而失败时，Backup status 后续应该
能够暴露 lock path 和 remote blocker operation metadata。

其他 log subcommand，例如 `log-start`、`log-stop`、`log-pause` 和 `log-resume`，
不作为第一阶段承诺。只有当执行阶段确认它们的 BR 失败路径也暴露 operation-aware lock
conflict 信息时，才纳入同一套适配。

#### Restore

`Restore` 的 PiTR restore 路径在适配范围内。

第一优先级是 `spec.mode=pitr`，包括 replication restore 的 phase 2
(`log-restore`)。BR 会把 operation context 传入 log restore 和 migration lock
相关路径，包括 PiTR restore 使用的 migration read lock 和 append lock。当这些流程
因为 lock conflict 失败时，Restore status 后续应该能够展示 blocker metadata。

标准 snapshot restore 和 volume-snapshot restore 不作为第一轮目标，因为它们不是这次
operation-aware external storage lock metadata 的主要消费路径。

#### CompactBackup

`CompactBackup` 暂不作为第一阶段确定范围。

当前 Operator 的 CompactBackup 主执行路径是 `tikv-ctl compact-log-backup`，BR 只用于
辅助生成 storage 参数。上游 BR operation metadata PR 的已知改动集中在 BR stream、
restore log client、PiTR collector、`operator migrate-to` 和 objstore locking 路径，
尚不能证明 CompactBackup 会创建或竞争这些 operation-aware external storage lock。

因此第一阶段不假定 CompactBackup 需要写入 `brOperations` 或 `lockBlocker`。如果后续确认
compact 路径确实会创建或竞争同一类 external storage lock，或者内核侧补齐了 compact 的
operation structured log，再把 `CompactBackup` 纳入同一套 status schema。

### Indirect Scope

#### BackupSchedule

`BackupSchedule` 间接相关，但第一轮不直接适配。

BackupSchedule 会创建和管理子 `Backup`、`CompactBackup` 资源，也可能因为保留策略触发
log truncate。lock observability 数据应该落在这些子 CR 上。BackupSchedule 可以继续
通过现有 status 字段，例如 `logBackup`、`lastBackup`、`lastCompact`，指向相关子资源。

### Out of Scope

- Snapshot `Backup`。
- Volume-snapshot `Backup`。
- Snapshot `Restore`。
- Volume-snapshot `Restore`。
- Restore prune job。
- Restore warmup job。
- Backup clean job。
- Federation 里的 VolumeBackup 和 VolumeRestore 资源。

这些路径仍然可能因为其他原因失败，但它们不是 BR operation-aware external storage
lock 的核心适配面。

## Design Details

第一轮实现讨论应从下面这些 CR 开始：

- `Backup`：log backup，尤其是 `log-truncate`。
- `Restore`：PiTR restore 和 replication log-restore。
- `CompactBackup`：待验证。只有确认 compact 路径使用同一类 operation-aware external
  storage lock 后再纳入。

第一轮明确实现两个功能：

这两个功能对应两个关键观测时间点：

1. **BR 进程启动并建立 operation context 时**：必须尽早获得本次 BR execution 的
   operation ID，并写入 owning CR 的 `brOperations`。这一步解决“这个 CR 启动过哪些
   operation”的反查问题。
2. **BR 遇到 external storage lock conflict 时**：必须解析 remote blocker 信息，并在
   本次执行最终失败时写入被阻塞 CR 的 `lockBlocker`。这一步解决“当前失败被哪个 lock
   owner 阻塞”的诊断问题。

Status 使用结构化字段承载观测结果，而不是只把信息拼进 condition message。Condition
message 可以保留面向人的摘要，但 operation 和 blocker identity 必须是机器可读字段，
方便后续根据 blocker operation ID 反查 CR。

`Backup`、`Restore`，以及未来确认纳入范围的 `CompactBackup` 应复用同一组 status
子结构，避免每个 CR 重复定义语义相同的 operation 和 blocker 字段。下面 YAML 只表达语义
结构，不承诺最终 Go/JSON 字段名：

```yaml
status:
  brOperations:
  - operationID: ...
    startedAt: ...
    command: ...
    observedAt: ...
  lockBlocker:
    lockPath: ...
    remoteOperationID: ...
    remoteStartedAt: ...
    resourceType: ...
    observedAt: ...
```

第一阶段 status schema 保持最小化，只暴露运维闭环必需的机器可读字段。`jobName`、
`podName`、`source`、raw `hint`、日志 `message`、`restoreID`、local failed operation ID
等诊断补充信息暂不进入 CRD status。Condition message 可以继续提供面向人的摘要。

### 功能 1：记录本 CR 的 BR Operation

当 BR 输出 operation started 日志后，backup-manager 读取其中的结构化字段，并写入对应 CR
status。这个动作应该尽量早发生，不需要等 Job 失败或完成。这样如果该 operation 后续创建
的 lock 残留并成为其他任务的 blocker，运维可以根据 blocker operation ID 反查到相关 CR。

第一阶段不让 controller-manager 通过 Kubernetes API 读取 Pod log，也不新增 `pods/log`
RBAC。现有 Backup/Restore 路径中的 backup-manager 进程已经在 Job Pod 内启动 BR，并读取
BR stdout/stderr；它也已经拥有更新相关 backup CR 的权限。因此 operation 和 blocker 解析应
放在 backup-manager 执行 BR 子进程的日志读取路径中：backup-manager 一边把 BR 输出写到
容器日志，一边解析结构化字段，并通过已有 status updater 写回 CR。

现有 Backup 和 Restore 路径会由 `backup-manager` 启动 BR，并读取 BR stdout/stderr，因此
可以在 backup-manager 内部解析 BR structured log。这样不依赖 controller-manager 的
`pods/log` 权限，也不会因为 Pod log API 权限或日志保留策略影响主闭环。

Backup 和 Restore 的 BR stdout、stderr 都需要进入同一个 parser。BR structured log 可能
出现在任一输出流中；如果只解析 stdout 或只解析 stderr，可能漏掉 operation 或 blocker
字段。第一阶段应避免为不同输出流维护两套解析逻辑。

stdout 和 stderr 都应实时逐行解析，而不是等待某一路输出结束后整块处理。现有 Backup 和
Restore 路径中，stdout 已经逐行读取，stderr 当前更接近执行结束后整块读取；第一阶段需要
把两路输出都接入实时 parser，确保 `BR operation started` 即使出现在 stderr 中，也能尽早
写入 `brOperations`。

Parser 第一阶段只面向 backup-manager 从 BR 子进程 stdout/stderr 读取到的原始 structured
log，不解析 Kubernetes Pod log 中被 klog 包装后的文本。这样可以避免依赖 klog 前缀、
时间戳和格式化细节。第一阶段不从 backup-manager 构造的失败 message 中 best-effort 提取
operation metadata，避免把非契约化文本格式引入 CRD status 语义。

更长期、更稳定的接口可以是 BR 输出本地机器可读 report file，例如由 backup-manager 传入
`--operation-report-file`，BR 以 JSONL 形式写入 operation started 和 lock conflict 事件。
但这需要 BR/内核侧新增正式接口，不作为第一阶段前提。第一阶段先基于现有 BR 原始
structured log 前进，未来如果 report file 可用，backup-manager 可以优先消费 report file，
再把日志解析保留为兼容 fallback。

具体 parser 字段契约不在本文档中提前锁死。实现阶段应基于目标 TiDB/BR PR 的最终 commit
现场确认 structured log 字段名和事件格式。本文档只要求两类语义可被结构化解析：operation
started 事件能够提供 operation identity；lock conflict 事件能够提供 remote blocker
identity。

解析结果由 backup-manager 直接写回 CR status，而不是交给 controller-manager 再写。
backup-manager 已经拥有更新相关 backup CR 的权限，且现有 restore progress、compact
progress 也采用“执行进程解析子进程输出并更新 status”的模式。这样可以把数据采集点保持在
BR 子进程旁边，避免 controller-manager 依赖 Pod log API。CompactBackup 是否纳入本功能
仍以后续验证为准。

Status 更新应由观测值变化触发，而不是每读一行日志就更新。解析到新的 operation ID 时才
追加 `brOperations`；重复观测到同一个 operation ID 时应去重并刷新已有记录的观测时间。
解析到新的 lock conflict 诊断时，先更新本次 execution 的候选 blocker；只有本次 BR
execution 最终以 lock conflict 失败时，才把候选 blocker 写入 `lockBlocker`。

Observability status 更新失败不应改变 BR 主流程结果。记录 operation status 失败时，
backup-manager 只记录自身日志并继续执行 BR；记录 lockBlocker 失败时，BR 原有失败结果
保持不变，不因为诊断 status 写入失败而替换或放大失败原因。

所有第一轮 in-scope 路径都必须能够记录 operation。Operator 不应为缺失日志猜测或合成
operation ID。如果某条路径无法证明会输出兼容的 operation structured log，则不应被列入
第一阶段确定范围。

由于 Kubernetes Job retry、Pod 重建或容器重启都可能重新启动 BR 进程，同一个 CR 可能对应
多个 BR operation。第一阶段使用有上限的 operation 列表，而不是单个最近值。列表按
observed time 倒序保存，优先保证最近若干次 BR execution 可反查。

operation 列表中的单条记录应保持轻量，建议字段包括：

- operation ID
- operation started time
- command
- observed time

第一阶段不要求解析 `BR operation restore ID resolved`，也不要求补充本 CR operation
的 restore ID。restore ID 对 PiTR lineage 有帮助，但不是“从 blocker operation ID
反查 CR”的必要条件，可以作为后续增强。

列表需要设置固定上限，避免 status 膨胀。第一阶段建议保留最近 10 条 operation。若同一个
operation ID 被重复观测到，应更新已有记录的观测时间，而不是追加重复项。

多阶段 Restore，例如 replication restore 的 snapshot phase 和 log-restore phase，也复用
同一个 `brOperations` 列表。不同 BR execution 通过 operation ID 和 command 字段区分，
不为每个 phase 单独定义一套 status 字段。

### 功能 2：失败时写入 Lock Blocker 诊断

当 BR Job 失败时，backup-manager 在 BR 子进程 stdout/stderr 中判断失败是否来自 external
storage lock conflict 或 stale lock blocker。如果是，则从 lock conflict 日志中的结构化字段
提取 remote blocker metadata，并写入对应 CR status 的诊断信息。

第一阶段 `lockBlocker` 只表示最近一次 lock conflict 诊断，不保留 blocker 历史。新的
lock conflict 诊断会覆盖旧的 `lockBlocker`。这样 status 始终回答“当前或最近一次失败
被哪个 blocker 阻塞”，避免历史诊断误导运维。

`lockBlocker` 应在成功完成或新的非 lock-conflict 失败出现时清空，因为它不再代表当前
诊断状态。`brOperations` 不随成功清空，继续保留最近 10 条 operation，用于后续反查。

`lockBlocker` 的写入和清理由 backup-manager 在本次 BR execution 结束时负责。Backup 和
Restore 路径中的 BR 是 backup-manager 通过子进程启动的；backup-manager 读取 stdout/stderr
期间记录本次 execution 是否观测到 lock conflict，并在 `cmd.Wait()` 返回后判断本次
execution 的最终结果。若本次成功，清空旧 `lockBlocker`；若本次失败且观测到 lock
conflict，写入新的 `lockBlocker`；若本次失败但没有观测到 lock conflict，清空旧
`lockBlocker`，避免旧诊断继续误导运维。

对运维最重要的是 remote blocker 信息：

- lock path
- remote operation ID
- remote operation started time
- resource type
- observed time

local failed operation ID 和 restore ID 可以从原始 BR 日志中辅助人工排查，但第一阶段不把
它们冻结为 CRD status 字段。Status 文案应明确指导：只有确认 blocker operation 对应的
Job、Pod 或 BR 进程已经不再运行后，才考虑手动删除对应 lock。

第一轮不假定 Operator 可以直接读取远端 lock 文件。远端路径如何定位、不同 storage
provider 的 credential 如何复用，以及是否应该提供机器可读的 lock report，都需要 BR
或内核侧继续配合。Operator 侧先只依赖 backup-manager 在执行 BR 子进程时已经能看到的
stdout 和 stderr。

第一轮数据来源如下：

- BR structured log：解析 BR stdout/stderr 中的 `BR operation started` 以及 lock
  conflict 日志里的结构化字段。

直接读取远端 lock JSON 暂不作为第一轮依赖。锁文件写入本身是一个不确定事件：BR 只有在
执行到加锁路径时才会写入；正常结束时锁可能很快被删除；如果在写锁前失败，则不会有锁
文件。因此 lock 文件适合后续作为 blocker 诊断的增强来源，但不适合作为记录本 CR
operation identity 的主机制。

实现上只解析结构化日志字段。若没有结构化字段，Operator 不合成 operation ID，也不猜测
blocker identity。

第一轮这些字段主要来自 BR lock conflict 日志，而不是 Operator 主动读取远端对象。若日志
中没有 lock path 或 remote blocker metadata，则 Operator 不应猜测远端路径。

## Execution Notes

本节记录执行阶段基于 `pingcap/tidb#69231` 当前 head 的现场确认结果。

- 目标 TiDB PR head：`955eb2b94928a5920fcca0a53308dbf2ba957e7e`。
- BR root flag 当前默认：`--log-format=text`，`--log-file` 默认为时间戳文件；Job
  侧已有 `BR_LOG_TO_TERM` 会让 BR log 打到 stdout/stderr。第一阶段 parser 若按 JSON
  structured log 实现，backup-manager 必须在纳入范围的 BR 命令上显式追加
  `--log-format=json`，不能依赖 BR 默认 text log。
- `github.com/pingcap/log` JSON encoder 的 message key 是 `message`。
- operation started marker：`BR operation started`。
- operation started fields：`operation_id`、`operation_started_at`、`command`，另有
  `host`、`pid` 可作为诊断补充但不是 CR status 必需字段。
- restore ID resolved marker：`BR operation restore ID resolved`，字段为
  `operation_id`、`operation_started_at`、`restore_id`。第一阶段不要求解析该事件。
- lock conflict marker 不是单一文案，而是使用 `pkg/objstore.LockConflictLogFields`
  产出的字段族。已确认的调用点包括 `Failed to acquire log truncate lock` 和
  `Encountered lock, will retry`。
- lock conflict 字段：
  - `path`
  - `local_owner_id`
  - `local_lock_type`
  - `local_hint`
  - `remote_blocker_count`
  - `remote_blocker_0_path`
  - `remote_blocker_0_owner_id`
  - `remote_blocker_0_lock_type`
  - `remote_blocker_0_hint`
  - 以及 `_1`、`_2` 后缀的最多 3 个 blocker sample。
- 如果没有 blocker sample，但 `ErrLocked.Meta` 可用，字段会退化为 `remote_owner_id`、
  `remote_lock_type`、`remote_hint`。
- `operation_started_at` 和 `restore_id` 在 lock metadata 中嵌入 `hint` 字符串，例如
  `operation_started_at=2026-06-15T12:00:00Z restore_id=123 detail="..."`；Operator
  第一阶段只从 hint 中解析 `operation_started_at`，作为 `remoteStartedAt`；`restore_id`
  暂不进入 CRD status。Operator 不能依赖远端 lock JSON。

Parser fixture 形态应基于上述字段，而不是自行发明字段。例如：

```json
{"level":"info","time":"2026/06/17 10:00:00.000 +00:00","caller":"operation/context.go:122","message":"BR operation started","operation_id":"11111111-1111-1111-1111-111111111111","operation_started_at":"2026-06-17T10:00:00Z","host":"br-pod","pid":123,"command":"log-restore"}
```

```json
{"level":"warn","time":"2026/06/17 10:01:00.000 +00:00","caller":"stream/stream_metas.go:1497","message":"Failed to acquire log truncate lock","error":"locked","path":"truncating.lock","local_owner_id":"local-op","local_lock_type":"log-truncate-exclusive","local_hint":"operation_started_at=2026-06-17T10:00:00Z","remote_blocker_count":1,"remote_blocker_0_path":"truncating.lock","remote_blocker_0_owner_id":"remote-op","remote_blocker_0_lock_type":"log-truncate-exclusive","remote_blocker_0_hint":"operation_started_at=2026-06-17T09:00:00Z restore_id=123"}
```

### Test Plan

详细测试计划会在执行计划阶段细化。至少需要覆盖：

- Backup log truncate 被已有 `truncating.lock` 阻塞。
- Restore PiTR log restore 被 migration metadata lock 阻塞。
- 若后续确认 CompactBackup 使用 operation-aware external storage lock，再补充
  CompactBackup 被 migration read、write 或 append lock 阻塞的测试。
- BR lock metadata 缺失时的兼容性，例如旧 BR 镜像或旧 lock 文件。

## Drawbacks

第一轮保持聚焦意味着部分 CR 失败仍然只会展示通用 Job 失败信息。这是可以接受的，因为
本工作的目标是有针对性的 lock observability，而不是完整替代 BR 失败诊断。

## Alternatives

### Operator 生成 Operation ID

Operator 可以生成一个 ID 并传给 BR。这个方案不适合作为 BR operation ID 本身，因为
operation 表示一次 BR 进程执行。单个 Operator CR 可能创建多个 Job 和多次 retry，如果
使用 CR 级别 ID，会混淆不同 BR 执行并削弱 lock ownership 语义。

Operator 后续仍然可以增加单独的 correlation ID 或 attempt ID，但它不应该替代 BR 的
`operation_id`。

### 适配所有 Backup 和 Restore CR

Operator 可以给所有 backup 和 restore 路径增加 lock-observability status。这个范围过
大，会把不相关的失败模式混进来。第一轮应该保持在 log backup truncate 和 PiTR/log
restore 这些上游 BR 改动已经提供明确 lock metadata 的路径上。

### 只依赖 Pod Log

Operator 可以保持 CR status 不变，让运维去检查 pod log。这个方案会损失主要的运维收益。
CR status 应该携带足够的 lock 身份信息来指导下一步排查，即使 pod log 仍然对完整诊断有用。
