---
title: TiDB Operator 1.1 Beta.1 Release Notes
---

# TiDB Operator 1.1 Beta.1 Release Notes

发布日期：2020 年 1 月 8 日

TiDB Operator 版本：1.1.0-beta.1

## 需要操作的变更

- 所有 [charts](https://github.com/pingcap/tidb-operator/tree/master/charts) 支持配置 `timezone` ([#1122](https://github.com/pingcap/tidb-operator/pull/1122), [@weekface](https://github.com/weekface))

    对于 `tidb-cluster` chart, 之前已经有 `timezone` 配置项（默认值：`UTC`）。如果用户没有修改成不同值（如：`Asia/Shangehai`），所有 Pods 不会被重建。

    如果用户改成其它值（如：`Asia/Shanghai`），所有相关的 Pods（添加 `TZ` 环境变量）会被重建（滚动升级）。

    对于其它 charts，之前在对应的 `values.yaml` 里没有 `timezone` 配置项。这个 PR 添加了对 `timezone` 配置项的支持。不管用户使用旧的 `values.yaml` 还是新的 `values.yaml`，所有相关的 Pods（添加了 `TZ` 环境变量）都不会被重建（滚动升级）。

    相关的 Pods 有：`pump`，`drainer`，`discovery`，`monitor`，`scheduled backup`，`tidb-initializer`，`tikv-importer`。

    TiDB Operator 维护的全部镜像的 `timezone` 都是 `UTC`。如果你使用自己的镜像，请确保你的镜像的 `timezone` 是 `UTC`。

## 其他重要变更

- 支持使用 [Backup & Restore (BR)](https://github.com/pingcap/br) 备份到 S3 ([#1280](https://github.com/pingcap/tidb-operator/pull/1280), [@DanielZhangQD](https://github.com/DanielZhangQD))
- 为 `TidbCluster` 添加基础默认设置及验证 ([#1429](https://github.com/pingcap/tidb-operator/pull/1429), [@aylei](https://github.com/aylei))
- 支持使用增强型 StatefulSet 进行扩缩容 ([#1361](https://github.com/pingcap/tidb-operator/pull/1361), [@cofyc](https://github.com/cofyc))
- 支持使用 `TidbInitializer` 自定义资源初始化 TiDB 集群 ([#1403](https://github.com/pingcap/tidb-operator/pull/1403), [@DanielZhangQD](https://github.com/DanielZhangQD))
- 优化 PD、TiKV、TiDB 的配置结构 ([#1411](https://github.com/pingcap/tidb-operator/pull/1411), [@aylei](https://github.com/aylei))
- 设置 `tidbcluster` 拥有的资源的实例 label 键的默认名称为集群名 ([#1419](https://github.com/pingcap/tidb-operator/pull/1419), [@aylei](https://github.com/aylei))
- `TidbCluster` 自定义资源支持管理 Pump 集群 ([#1269](https://github.com/pingcap/tidb-operator/pull/1269), [@aylei](https://github.com/aylei))
- 修复 tikv-importer 默认配置中的错误 ([#1415](https://github.com/pingcap/tidb-operator/pull/1415), [@aylei](https://github.com/aylei))
- 支持在资源配置中配置临时存储 ([#1398](https://github.com/pingcap/tidb-operator/pull/1398), [@aylei](https://github.com/aylei))
- 添加不使用 Helm 运维 TiDB 集群的 e2e 测试用例 ([#1396](https://github.com/pingcap/tidb-operator/pull/1396), [@aylei](https://github.com/aylei))
- 发布 Terraform Aliyun ACK 版本，指定默认版本为 `1.14.8-aliyun.1` ([#1284](https://github.com/pingcap/tidb-operator/pull/1284), [@shonge](https://github.com/shonge))
- 优化 scheduler 的报错信息 ([#1373](https://github.com/pingcap/tidb-operator/pull/1373), [@weekface](https://github.com/weekface))
- 把 `system:kube-scheduler` 集群 role 绑定到 `tidb-scheduler` 服务账号 ([#1355](https://github.com/pingcap/tidb-operator/pull/1355), [@shonge](https://github.com/shonge))
- 添加新的 `TidbInitializer` 自定义资源类型 ([#1391](https://github.com/pingcap/tidb-operator/pull/1391), [@aylei](https://github.com/aylei))
- 升级默认备份镜像为 `pingcap/tidb-cloud-backup:20191217`，优化 `-r` 选项 ([#1360](https://github.com/pingcap/tidb-operator/pull/1360), [@aylei](https://github.com/aylei))
- 修复最新 EKS AMI 的 Docker ulimit 配置的错误 ([#1349](https://github.com/pingcap/tidb-operator/pull/1349), [@aylei](https://github.com/aylei))
- 支持同步 Pump 状态到 TiDB 集群 ([#1292](https://github.com/pingcap/tidb-operator/pull/1292), [@shonge](https://github.com/shonge))
- `tidb-controller-manager` 支持自动创建并调和 `tidb-discovery-service` ([#1322](https://github.com/pingcap/tidb-operator/pull/1322), [@aylei](https://github.com/aylei))
- 扩大备份恢复的适用范围，提升安全性 ([#1276](https://github.com/pingcap/tidb-operator/pull/1276), [@onlymellb](https://github.com/onlymellb))
- `TidbCluster` 自定义资源中添加 PD 和 TiKV 配置 ([#1330](https://github.com/pingcap/tidb-operator/pull/1330), [@aylei](https://github.com/aylei))
- `TidbCluster` 自定义资源中添加 TiDB 配置 ([#1291](https://github.com/pingcap/tidb-operator/pull/1291), [@aylei](https://github.com/aylei))
- 添加 TiKV 配置 schema ([#1306](https://github.com/pingcap/tidb-operator/pull/1306), [@aylei](https://github.com/aylei))
- TiDB `host:port` 开启后再初始化 TiDB 集群，加快初始化速度 ([#1296](https://github.com/pingcap/tidb-operator/pull/1296), [@cofyc](https://github.com/cofyc))
- 移除 DinD 相关脚本 ([#1283](https://github.com/pingcap/tidb-operator/pull/1283), [@shonge](https://github.com/shonge))
- 支持从 AWS 和 GCP 的元数据获取验证证书 ([#1248](https://github.com/pingcap/tidb-operator/pull/1248), [@gregwebs](https://github.com/gregwebs))
- 增加 `tidb-controller-manager` 的权限来操作 `configmap` ([#1275](https://github.com/pingcap/tidb-operator/pull/1275), [@aylei](https://github.com/aylei))
- 通过 `tidb-controller-manager` 管理 TiDB 服务 ([#1242](https://github.com/pingcap/tidb-operator/pull/1242), [@aylei](https://github.com/aylei))
- 支持为组件设置 cluster-level 配置 ([#1193](https://github.com/pingcap/tidb-operator/pull/1193), [@aylei](https://github.com/aylei))
- 从当前时间来获取时间字符串，代替从 Pod Name 获取 ([#1229](https://github.com/pingcap/tidb-operator/pull/1229), [@weekface](https://github.com/weekface))
- 升级 TiDB 时，TiDB Operator 将不再注销 ddl owner，因为 TiDB 将在关闭时自动转移 ddl owner ([#1239](https://github.com/pingcap/tidb-operator/pull/1239), [@aylei](https://github.com/aylei))
- 修复 Google terraform 模块的 `use_ip_aliases` 错误 ([#1206](https://github.com/pingcap/tidb-operator/pull/1206), [@tennix](https://github.com/tennix))
- 升级 TiDB 默认版本为 v3.0.5 ([#1179](https://github.com/pingcap/tidb-operator/pull/1179), [@shonge](https://github.com/shonge))
- 升级 Docker image 的基本系统到最新稳定版 ([#1178](https://github.com/pingcap/tidb-operator/pull/1178), [@AstroProfundis](https://github.com/AstroProfundis))
- `tkctl get TiKV` 支持为每个 TiKV Pod 显示储存状态 ([#916](https://github.com/pingcap/tidb-operator/pull/916), [@Yisaer](https://github.com/Yisaer))
- 添加一个选项实现跨命名空间的监控 ([#907](https://github.com/pingcap/tidb-operator/pull/907), [@gregwebs](https://github.com/gregwebs))
- 在 `tkctl get TiKV` 中添加 `STOREID` 列，为每个 TiKV Pod 显示 store ID ([#842](https://github.com/pingcap/tidb-operator/pull/842), [@Yisaer](https://github.com/Yisaer))
- 用户可以在 chart 中通过 `values.tidb.permitHost` 指定被许可的主机 ([#779](https://github.com/pingcap/tidb-operator/pull/779), [@shonge](https://github.com/shonge))
- 为 kubelet 添加 zone 标签和资源预留配置 ([#871](https://github.com/pingcap/tidb-operator/pull/871), [@aylei](https://github.com/aylei))
- 修复了 apply 语法可能会导致 kubeconfig 被破坏的问题 ([#861](https://github.com/pingcap/tidb-operator/pull/861), [@cofyc](https://github.com/cofyc))
- 支持 TiKV 组件的灰度发布 ([#869](https://github.com/pingcap/tidb-operator/pull/869), [@onlymellb](https://github.com/onlymellb))
- 最新的 chart 兼容旧的 controller manager ([#856](https://github.com/pingcap/tidb-operator/pull/856), [@onlymellb](https://github.com/onlymellb))
- 添加对 TiDB 集群中 TLS 加密连接的基础支持 ([#750](https://github.com/pingcap/tidb-operator/pull/750), [@AstroProfundis](https://github.com/AstroProfundis))
- 支持为 tidb-operator chart 配置 nodeSelector、亲和力和容忍度 ([#855](https://github.com/pingcap/tidb-operator/pull/855), [@shonge](https://github.com/shonge))
- 支持为 TiDB 集群的所有容器配置资源请求和限制 ([#853](https://github.com/pingcap/tidb-operator/pull/853), [@aylei](https://github.com/aylei))
- 支持使用 `Kind` (Kubernetes IN Docker) 建立测试环境 ([#791](https://github.com/pingcap/tidb-operator/pull/791), [@xiaojingchen](https://github.com/xiaojingchen))
- 支持使用 `tidb-lightning` chart 恢复特定的数据源 ([#827](https://github.com/pingcap/tidb-operator/pull/827), [@tennix](https://github.com/tennix))
- 添加 `tikvGCLifeTime` 选项 ([#835](https://github.com/pingcap/tidb-operator/pull/835), [@weekface](https://github.com/weekface))
- 更新默认的备份镜像到 `pingcap/tidb-cloud-backup:20190828` ([#846](https://github.com/pingcap/tidb-operator/pull/846), [@aylei](https://github.com/aylei))
- 修复 Pump/Drainer 的数据目录避免潜在的数据丢失 ([#826](https://github.com/pingcap/tidb-operator/pull/826), [@aylei](https://github.com/aylei))
- 修复了 `tkctl` 使用 `-oyaml` 或 `-ojson` 标志不输出任何内容的问题，支持查看特定 Pod 或 PV 的详细信息，改进 `tkctl get` 命令的输出 ([#822](https://github.com/pingcap/tidb-operator/pull/822), [@onlymellb](https://github.com/onlymellb))
- 为 mydumper 添加推荐选项：`-t 16 -F 64 --skip-tz-utc` ([#828](https://github.com/pingcap/tidb-operator/pull/828), [@weekface](https://github.com/weekface))
- 在 `deploy /gcp` 中支持单可用区和多可用区集群 ([#809](https://github.com/pingcap/tidb-operator/pull/809), [@cofyc](https://github.com/cofyc))
- 修复当默认备份名被使用时备份失败的问题 ([#836](https://github.com/pingcap/tidb-operator/pull/836), [@DanielZhangQD](https://github.com/DanielZhangQD))
- 增加对 TiDB Lightning 的支持 ([#817](https://github.com/pingcap/tidb-operator/pull/817), [@tennix](https://github.com/tennix))
- 支持从指定的定时备份目录还原 TiDB 集群 ([#804](https://github.com/pingcap/tidb-operator/pull/804), [@onlymellb](https://github.com/onlymellb))
- 修复了一个 `tkctl` 日志中的异常 ([#797](https://github.com/pingcap/tidb-operator/pull/797), [@onlymellb](https://github.com/onlymellb))
- 在 PD/TiKV/TiDB 规范中添加 `hostNetwork` 字段，以便可以在主机网络中运行 TiDB 组件 ([#774](https://github.com/pingcap/tidb-operator/pull/774), [@cofyc](https://github.com/cofyc))
- 当 mdadm 和 RAID 在 GKE 上可用时，使用 mdadm 和 RAID 而不是 LVM ([#789](https://github.com/pingcap/tidb-operator/pull/789), [@gregwebs](https://github.com/gregwebs))
- 支持通过增加 PVC 存储大小来动态扩展云存储 PV ([#772](https://github.com/pingcap/tidb-operator/pull/772), [@tennix](https://github.com/tennix))
- 支持为 PD/TiDB/TiKV 节点池配置节点镜像类型 ([#776](https://github.com/pingcap/tidb-operator/pull/776), [@cofyc](https://github.com/cofyc))
- 添加脚本以删除 GKE 的未使用磁盘 ([#771](https://github.com/pingcap/tidb-operator/pull/771), [@gregwebs](https://github.com/gregwebs))
- 对 Pump 和 Drainer 增加 `binlog.pump.config` and `binlog.drainer.config` 配置项 ([#693](https://github.com/pingcap/tidb-operator/pull/693), [@weekface](https://github.com/weekface))
- 当 Pump 变为 “offline” 状态时，阻止 Pump 进程退出 ([#769](https://github.com/pingcap/tidb-operator/pull/769), [@weekface](https://github.com/weekface))
- 引入新的 `tidb-drainer` helm chart 以管理多个 Drainer ([#744](https://github.com/pingcap/tidb-operator/pull/744), [@aylei](https://github.com/aylei))
- 添加 backup-manager 工具以支持备份、还原和清除备份数据 ([#694](https://github.com/pingcap/tidb-operator/pull/694), [@onlymellb](https://github.com/onlymellb))
- 在 Pump/Drainer 的配置中添加 `affinity` 选项 ([#741](https://github.com/pingcap/tidb-operator/pull/741), [@weekface](https://github.com/weekface))
- 修复 TiKV failover 后某些情况下的 TiKV 缩容失败 ([#726](https://github.com/pingcap/tidb-operator/pull/726), [@onlymellb](https://github.com/onlymellb))
- 修复 UpdateService 的错误处理 ([#718](https://github.com/pingcap/tidb-operator/pull/718), [@DanielZhangQD](https://github.com/DanielZhangQD))
- 将 e2e 运行时间从 60m 减少到 20m ([#713](https://github.com/pingcap/tidb-operator/pull/713), [@weekface](https://github.com/weekface))
- 添加 `AdvancedStatefulset` 功能以使用增强型 StatefulSet 代替 Kubernetes 内置的 StatefulSet ([#1108](https://github.com/pingcap/tidb-operator/pull/1108), [@cofyc](https://github.com/cofyc))
- 支持为 TiDB 集群自动生成证书 ([#782](https://github.com/pingcap/tidb-operator/pull/782), [@AstroProfundis](https://github.com/AstroProfundis))
- 支持备份到 GCS ([#1127](https://github.com/pingcap/tidb-operator/pull/1127), [@onlymellb](https://github.com/onlymellb))
- 支持为 TiDB 配置 `net.ipv4.tcp_keepalive_time` 和 `net.core.somaxconn`，以及为 TiKV 配置 `net.core.somaxconn` ([#1107](https://github.com/pingcap/tidb-operator/pull/1107), [@DanielZhangQD](https://github.com/DanielZhangQD))
- 为聚合的 apiserver 添加基本的 e2e 测试 ([#1109](https://github.com/pingcap/tidb-operator/pull/1109), [@aylei](https://github.com/aylei))
- 添加 `enablePVReclaim` 选项，在 tidb-operator 缩容 TiKV 或 PD 时回收 PV ([#1037](https://github.com/pingcap/tidb-operator/pull/1037), [@onlymellb](https://github.com/onlymellb))
- 统一所有兼容 S3 的存储以支持备份和还原 ([#1088](https://github.com/pingcap/tidb-operator/pull/1088), [@onlymellb](https://github.com/onlymellb))
- 将 `podSecuriyContext` 的默认值设置为 nil ([#1079](https://github.com/pingcap/tidb-operator/pull/1079), [@aylei](https://github.com/aylei))
- 在 `tidb-operator` chart 中添加 `tidb-apiserver` ([#1083](https://github.com/pingcap/tidb-operator/pull/1083), [@aylei](https://github.com/aylei))
- 添加新组件 TiDB aggregated apiserver ([#1048](https://github.com/pingcap/tidb-operator/pull/1048), [@aylei](https://github.com/aylei))
- 修复了当发行版名称是 `un-wanted` 时 `tkctl version` 不起作用的问题 ([#1065](https://github.com/pingcap/tidb-operator/pull/1065), [@aylei](https://github.com/aylei))
- 支持暂停备份计划 ([#1047](https://github.com/pingcap/tidb-operator/pull/1047), [@onlymellb](https://github.com/onlymellb))
- 修复了在 Terraform 输出中 TiDB Loadbalancer 为空的问题 ([#1045](https://github.com/pingcap/tidb-operator/pull/1045), [@DanielZhangQD](https://github.com/DanielZhangQD))
- 修复了 AWS terraform 脚本中 `create_tidb_cluster_release` 变量不起作用的问题 ([#1062](https://github.com/pingcap/tidb-operator/pull/1062), [@aylei](https://github.com/aylei))
- 在稳定性测试中，默认情况下启用 `ConfigMapRollout` ([#1036](https://github.com/pingcap/tidb-operator/pull/1036), [@aylei](https://github.com/aylei))
- 迁移到使用 `app/v1` 并且不再支持 1.9 版本之前的 Kubernetes ([#1012](https://github.com/pingcap/tidb-operator/pull/1012), [@Yisaer](https://github.com/Yisaer))
- 暂停 AWS TiKV 自动缩放组的 `ReplaceUnhealthy` 流程 ([#1014](https://github.com/pingcap/tidb-operator/pull/1014), [@aylei](https://github.com/aylei))
- 将 `tidb-monitor-reloader` 镜像更改为 `pingcap/tidb-monitor-reloader:v1.0.1` ([#898](https://github.com/pingcap/tidb-operator/pull/898), [@qiffang](https://github.com/qiffang))
- 添加一些 sysctl 内核参数设置以进行调优 ([#1016](https://github.com/pingcap/tidb-operator/pull/1016), [@tennix](https://github.com/tennix))
- 备份计划支持设置最长保留时间 ([#979](https://github.com/pingcap/tidb-operator/pull/979), [@onlymellb](https://github.com/onlymellb))
- 将默认的 TiDB 版本升级到 v3.0.4 ([#837](https://github.com/pingcap/tidb-operator/pull/837), [@shonge](https://github.com/shonge))
- 在阿里云上修复 TiDB Operator 的定制值文件 ([#971](https://github.com/pingcap/tidb-operator/pull/971), [@DanielZhangQD](https://github.com/DanielZhangQD))
- 在 TiKV 中添加 `maxFailoverCount` 限制 ([#965](https://github.com/pingcap/tidb-operator/pull/965), [@weekface](https://github.com/weekface))
- 支持在 AWS Terraform 脚本中为 tidb-operator 设置自定义配置 ([#946](https://github.com/pingcap/tidb-operator/pull/946), [@aylei](https://github.com/aylei))
- 在 TiKV 容量不是 GiB 的倍数时，将其转换为 MiB ([#942](https://github.com/pingcap/tidb-operator/pull/942), [@cofyc](https://github.com/cofyc))
- 修复 Drainer 的配置错误 ([#939](https://github.com/pingcap/tidb-operator/pull/939), [@weekface](https://github.com/weekface))
- 支持使用自定义的 `values.yaml` 部署 TiDB Operator 和 TiDB 集群 ([#959](https://github.com/pingcap/tidb-operator/pull/959), [@DanielZhangQD](https://github.com/DanielZhangQD))
- 支持为 PD、TiKV 和 TiDB Pod 指定 `SecurityContext`，并为 AWS 启用 `tcp keepalive` ([#915](https://github.com/pingcap/tidb-operator/pull/915), [@aylei](https://github.com/aylei))
