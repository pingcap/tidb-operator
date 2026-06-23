# BR 锁 Operation 可观测性适配范围

## Summary

TiDB BR 正在为 external storage lock 增加 operation-aware metadata。每一次 BR
进程执行都会拥有一个 operation ID，并在 lock 文件或 lock conflict 日志中暴露
operation start time、restore ID、lock resource type 等 metadata。

TiDB Operator 不应该生成或替换 BR 的 operation ID。operation ID 属于真正创建
external storage lock 的 BR 执行进程。Operator 的职责是把 BR 已经提供的 operation
身份信息透传到相关 CR 上，让运维能够判断哪个 CR 管理过某次 BR operation，并在后续
需要排查残留 lock 时，可以通过 operation ID 反查可能的 owner。

本文档记录第一阶段适配范围和关键设计决策。最终 CRD 字段名和 BR structured log
字段契约需要在执行阶段基于目标 TiDB/BR PR 的最终 commit 现场确认。

## Motivation

当 BR 进程 crash、被 kill，或者退出时无法完成 cleanup，external storage lock
可能会长期残留。后续 BR operation 被这种锁阻塞时，Kubernetes Job status 往往只会
给出类似 `BackoffLimitExceeded` 的通用失败原因，不足以让运维判断具体涉及哪个 lock
file，也无法判断之前是哪一次 BR 执行创建了锁。

BR 侧的 operation metadata 可以让 lock owner 变得可见。Operator 第一阶段只把“本 CR
启动过哪些 BR operation”暴露到 CR 层面，避免用户第一步就必须去翻 pod log 才知道
operation ID。remote lock conflict 的具体 blocker 信息仍以 BR 日志为准，等 BR 提供更稳定
的机器可读结果后再考虑进入 status。

### Goals

- 保持 BR 作为 `operation_id` 的权威来源。
- 明确哪些 Operator CR 需要第一轮适配 BR lock metadata。
- 第一轮聚焦已经确认会使用 BR operation-aware external storage lock 的流程，例如 log
  backup truncate 和 PiTR/log restore。
- 功能 1：BR operation 开始后，backup-manager 从 BR structured log 中读取 operation ID，
  并尽早写入对应 CR status，用于后续 operation owner 反查。
- 不把 remote blocker 或 lock conflict 诊断冻结到 CRD status。当前 BR retry 日志和最终
  错误之间缺少稳定的结构化因果关系，单靠日志判断最终失败是否由某个 remote blocker 导致
  容易误导运维。

### Non-Goals

- Operator 不生成 BR operation ID。
- Operator 不改变 BR lock ownership 语义。
- 本范围不覆盖所有 backup 或 restore 失败诊断。
- 本范围不要求给 `BackupSchedule` 增加新的 status 字段。
- 本范围不定义最终的 lock metadata CRD schema。
- 本范围不新增 `lockBlocker` 之类的 remote blocker status 字段。

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
- 第一阶段不读取远端 lock JSON，也不假定 Operator 能定位远端 lock 文件路径。远端对象可
  作为后续增强，但当前闭环依赖 BR stdout/stderr structured log 中已经暴露的信息。
- 更稳定的长期接口可以是 BR 输出本地机器可读 report file；第一阶段先沿日志解析方向推进。

## Proposal

### In Scope

#### Backup

`Backup` 的 `log-truncate` 路径是第一阶段确定适配范围。

第一优先级是 `spec.mode=log` 且 subcommand 为 `log-truncate` 的路径。BR 现在用
`truncating.lock` external storage lock 保护 log truncate，并把它标记为 resource
type `log-truncate-exclusive`。第一阶段 Backup status 只记录本 CR 启动的 BR operation；
truncate 因已有锁失败时，remote blocker 细节仍从 BR 日志确认。

其他 log subcommand，例如 `log-start`、`log-stop`、`log-pause` 和 `log-resume`，
不作为第一阶段承诺。只有当执行阶段确认它们的 BR 失败路径也暴露 operation-aware lock
conflict 信息时，才纳入同一套适配。

#### Restore

`Restore` 的 PiTR restore 路径在适配范围内。

第一优先级是 `spec.mode=pitr`，包括 replication restore 的 phase 2
(`log-restore`)。BR 会把 operation context 传入 log restore 和 migration lock
相关路径，包括 PiTR restore 使用的 migration read lock 和 append lock。当这些流程
因为 lock conflict 失败时，第一阶段 Restore status 仍只记录本 CR 的 BR operation。

标准 snapshot restore 和 volume-snapshot restore 不作为第一轮目标，因为它们不是这次
operation-aware external storage lock metadata 的主要消费路径。

#### CompactBackup

`CompactBackup` 暂不作为第一阶段确定范围。

当前 Operator 的 CompactBackup 主执行路径是 `tikv-ctl compact-log-backup`，BR 只用于
辅助生成 storage 参数。上游 BR operation metadata PR 的已知改动集中在 BR stream、
restore log client、PiTR collector、`operator migrate-to` 和 objstore locking 路径，
尚不能证明 CompactBackup 会创建或竞争这些 operation-aware external storage lock。

因此第一阶段不假定 CompactBackup 需要写入 `brOperations`。如果后续确认 compact 路径确实
会创建或竞争同一类 external storage lock，或者内核侧补齐了 compact 的 operation structured
log，再把 `CompactBackup` 纳入同一套 status schema。

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

第一轮只实现一个关键观测点：

1. **BR 进程启动并建立 operation context 时**：必须尽早获得本次 BR execution 的
   operation ID，并写入 owning CR 的 `brOperations`。这一步解决“这个 CR 启动过哪些
   operation”的反查问题。

Status 使用结构化字段承载观测结果，而不是只把信息拼进 condition message。Condition
message 可以保留面向人的摘要，但 operation identity 必须是机器可读字段，方便后续根据
operation ID 反查 CR。

`Backup`、`Restore`，以及未来确认纳入范围的 `CompactBackup` 应复用同一组 status
子结构，避免每个 CR 重复定义语义相同的 operation 字段。下面 YAML 只表达语义结构，不承诺
最终 Go/JSON 字段名：

```yaml
status:
  brOperations:
  - operationID: ...
    startedAt: ...
    command: ...
    observedAt: ...
```

第一阶段 status schema 保持最小化，只暴露运维闭环必需的机器可读字段。`jobName`、
`podName`、`source`、raw `hint`、日志 `message`、`restoreID`、local failed operation ID
等诊断补充信息暂不进入 CRD status。Condition message 可以继续提供面向人的摘要。

### 功能 1：记录本 CR 的 BR Operation

当 BR 输出 operation started 日志后，backup-manager 读取其中的结构化字段，并写入对应 CR
status。这个动作应该尽量早发生，不需要等 Job 失败或完成。这样如果该 operation 后续创建
的 lock 残留并成为其他任务的 blocker，运维可以根据 operation ID 反查到相关 CR。

第一阶段不让 controller-manager 通过 Kubernetes API 读取 Pod log，也不新增 `pods/log`
RBAC。现有 Backup/Restore 路径中的 backup-manager 进程已经在 Job Pod 内启动 BR，并读取
BR stdout/stderr；它也已经拥有更新相关 backup CR 的权限。因此 operation 解析应
放在 backup-manager 执行 BR 子进程的日志读取路径中：backup-manager 一边把 BR 输出写到
容器日志，一边解析结构化字段，并通过已有 status updater 写回 CR。

现有 Backup 和 Restore 路径会由 `backup-manager` 启动 BR，并读取 BR stdout/stderr，因此
可以在 backup-manager 内部解析 BR structured log。这样不依赖 controller-manager 的
`pods/log` 权限，也不会因为 Pod log API 权限或日志保留策略影响主闭环。

Backup 和 Restore 的 BR stdout、stderr 都需要进入同一个 parser。BR structured log 可能
出现在任一输出流中；如果只解析 stdout 或只解析 stderr，可能漏掉 operation 字段。第一阶段
应避免为不同输出流维护两套解析逻辑。

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
现场确认 structured log 字段名和事件格式。第一阶段只要求 operation started 事件能够提供
operation identity。lock conflict 日志可以作为人类排障线索，但不作为 CRD status 契约。

解析结果由 backup-manager 直接写回 CR status，而不是交给 controller-manager 再写。
backup-manager 已经拥有更新相关 backup CR 的权限，且现有 restore progress、compact
progress 也采用“执行进程解析子进程输出并更新 status”的模式。这样可以把数据采集点保持在
BR 子进程旁边，避免 controller-manager 依赖 Pod log API。CompactBackup 是否纳入本功能
仍以后续验证为准。

Status 更新应由观测值变化触发，而不是每读一行日志就更新。解析到新的 operation ID 时才
追加 `brOperations`；重复观测到同一个 operation ID 时应去重并刷新已有记录的观测时间。

Observability status 更新失败不应改变 BR 主流程结果。记录 operation status 失败时，
backup-manager 只记录自身日志并继续执行 BR；BR 原有失败结果保持不变，不因为诊断 status
写入失败而替换或放大失败原因。

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
的 restore ID。restore ID 对 PiTR lineage 有帮助，但不是“从 operation ID 反查 CR”的
必要条件，可以作为后续增强。

列表需要设置固定上限，避免 status 膨胀。第一阶段建议保留最近 10 条 operation。若同一个
operation ID 被重复观测到，应更新已有记录的观测时间，而不是追加重复项。

多阶段 Restore，例如 replication restore 的 snapshot phase 和 log-restore phase，也复用
同一个 `brOperations` 列表。不同 BR execution 通过 operation ID 和 command 字段区分，
不为每个 phase 单独定义一套 status 字段。

### 不记录 Lock Blocker 诊断

第一阶段不把 remote blocker 信息写入 CR status。原因是 BR 当前会输出 retry 过程中的
lock conflict 采样日志，也会在 60 次重试耗尽后返回带原始 locked error 的最终错误，但
Operator 侧单靠日志流无法稳定区分“曾经遇到过 transient blocker”和“最终失败就是因为这个
blocker”。如果把采样日志暂存后在任意 BR 失败时写入 status，可能把无关 blocker 标记成最终
失败原因。

因此 remote blocker、lock path、remote operation started time、resource type 等字段暂时
只保留在 BR 日志中。后续如果 BR 提供正式的机器可读 report file 或明确的 terminal lock
failure 事件，再评估是否新增单独的 blocker status 字段。

第一轮不假定 Operator 可以直接读取远端 lock 文件。远端路径如何定位、不同 storage
provider 的 credential 如何复用，以及是否应该提供机器可读的 lock report，都需要 BR
或内核侧继续配合。Operator 侧先只依赖 backup-manager 在执行 BR 子进程时已经能看到的
stdout 和 stderr。

第一轮数据来源如下：

- BR structured log：解析 BR stdout/stderr 中的 `BR operation started` 字段。lock
  conflict 日志字段只作为人工排障依据，不转换为 CRD status。

直接读取远端 lock JSON 暂不作为第一轮依赖。锁文件写入本身是一个不确定事件：BR 只有在
执行到加锁路径时才会写入；正常结束时锁可能很快被删除；如果在写锁前失败，则不会有锁
文件。因此 lock 文件适合后续作为 blocker 诊断的增强来源，但不适合作为记录本 CR
operation identity 的主机制。

实现上只解析结构化日志字段。若没有结构化字段，Operator 不合成 operation ID，也不猜测
blocker identity。

第一轮不把 BR lock conflict 日志字段转换为 CRD status。若日志中没有 lock path 或 remote
blocker metadata，则 Operator 不应猜测远端路径。

## Execution Notes

本节记录执行阶段基于 `pingcap/tidb#69231` 当前 head 的现场确认结果。

- 目标 TiDB PR head：`e9b7fff5ee3d464fad0bb0204b1ce01173189f25`。
- BR root flag 当前默认：`--log-format=text`，`--log-file` 默认为时间戳文件；Job
  侧已有 `BR_LOG_TO_TERM` 会让 BR log 打到 stdout/stderr。第一阶段沿用既有终端日志
  读取方式，不覆盖用户指定的 log format。Parser 需要从 BR 默认 text log 中的
  bracket fields 解析 operation started 字段；JSON log 仅作为兼容输入。
- `github.com/pingcap/log` JSON encoder 的 message key 是 `message`；默认 text log
  中 message 形如 `["BR operation started"]`，字段形如 `[operation_id=...]`。
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
  `operation_started_at=2026-06-15T12:00:00Z restore_id=123 detail="..."`。Operator
  第一阶段不解析这些 remote blocker hint 字段进入 status，也不能依赖远端 lock JSON。

Operation parser fixture 形态应基于上述字段，而不是自行发明字段。例如默认 text log：

```text
[2026/06/17 10:00:00.000 +00:00] [INFO] [context.go:122] ["BR operation started"] [operation_id=11111111-1111-1111-1111-111111111111] [operation_started_at=2026-06-17T10:00:00Z] [host=br-pod] [pid=123] [command=log-restore]
```

```text
[2026/06/17 10:01:00.000 +00:00] [WARN] [stream_metas.go:1497] ["Failed to acquire log truncate lock"] [error=locked] [path=truncating.lock] [local_owner_id=local-op] [local_lock_type=log-truncate-exclusive] [local_hint=operation_started_at=2026-06-17T10:00:00Z] [remote_blocker_count=1] [remote_blocker_0_path=truncating.lock] [remote_blocker_0_owner_id=remote-op] [remote_blocker_0_lock_type=log-truncate-exclusive] [remote_blocker_0_hint="operation_started_at=2026-06-17T09:00:00Z restore_id=123"]
```

兼容的 JSON log 示例：

```json
{"level":"info","time":"2026/06/17 10:00:00.000 +00:00","caller":"operation/context.go:122","message":"BR operation started","operation_id":"11111111-1111-1111-1111-111111111111","operation_started_at":"2026-06-17T10:00:00Z","host":"br-pod","pid":123,"command":"log-restore"}
```

```json
{"level":"warn","time":"2026/06/17 10:01:00.000 +00:00","caller":"stream/stream_metas.go:1497","message":"Failed to acquire log truncate lock","error":"locked","path":"truncating.lock","local_owner_id":"local-op","local_lock_type":"log-truncate-exclusive","local_hint":"operation_started_at=2026-06-17T10:00:00Z","remote_blocker_count":1,"remote_blocker_0_path":"truncating.lock","remote_blocker_0_owner_id":"remote-op","remote_blocker_0_lock_type":"log-truncate-exclusive","remote_blocker_0_hint":"operation_started_at=2026-06-17T09:00:00Z restore_id=123"}
```

### Test Plan

详细测试计划会在执行计划阶段细化。至少需要覆盖：

- Backup log truncate 能记录自身 BR operation；若出现 `truncating.lock` 冲突，remote
  blocker 字段只保留在日志中。
- Restore PiTR log restore 能记录自身 BR operation；若出现 migration metadata lock
  冲突，remote blocker 字段只保留在日志中。
- 若后续确认 CompactBackup 使用 operation-aware external storage lock，再补充
  CompactBackup operation 记录测试。
- BR operation metadata 缺失时的兼容性，例如旧 BR 镜像。

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
