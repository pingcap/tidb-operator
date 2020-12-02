---
title: TiDB Operator 1.1 RC.1 Release Notes
---

# TiDB Operator 1.1 RC.1 Release Notes

Release date: April 1, 2020

TiDB Operator version: 1.1.0-rc.1

## 需要采取的行动

- 将为 tidb-server 配置 `--advertise-address` 选项。这会触发 tidb-server 组件的滚动升级。 您可以在升级 tidb-operator 之前将 `spec.paused` 设置为 `true` 以避免滚动升级的行为，并在准备好升级 tidb-server 版本时将其设置回 `false` ([#2076](https://github.com/pingcap/tidb-operator/pull/2076), [@cofyc](https://github.com/cofyc))
- 在 backup and restore spec 中添加 `tlsClient.tlsSecret` 字段。可以通过该字段指定包含证书的密钥的名称 ([#2003](https://github.com/pingcap/tidb-operator/pull/2003), [@shuijing198799](https://github.com/shuijing198799))
- 为 `Backup`, `Restore` 以及 `BakcupSchedule` 移除 `spec.br.pd`, `spec.br.ca`, `spec.br.cert`, `spec.br.key` 选项，添加 `spec.br.cluster`, `spec.br.clusterNamespace` 选项，让 BR 的配置项更加合理 ([#1836](https://github.com/pingcap/tidb-operator/pull/1836), [@shuijing198799](https://github.com/shuijing198799))

## 其他需要注意的变更

- 在 `Restore` 中使用 `tidb-lightning` 替代 `loader` ([#2068](https://github.com/pingcap/tidb-operator/pull/2068), [@Yisaer](https://github.com/Yisaer))
- 为 TiDB 组件添加 `cert-allowed-cn` 支持 ([#2061](https://github.com/pingcap/tidb-operator/pull/2061), [@weekface](https://github.com/weekface))
- 修复 PD `location-labels` 配置项的问题 ([#1941](https://github.com/pingcap/tidb-operator/pull/1941), [@aylei](https://github.com/aylei))
- 可以通过 `spec.paused` 控制 TiDB 集群暂停部署 ([#2013](https://github.com/pingcap/tidb-operator/pull/2013), [@cofyc](https://github.com/cofyc))
- 在使用 CR 部署 TiDB 集群时，TiDB 的 `max-backups` 配置项默认值设为 `3` ([#2045](https://github.com/pingcap/tidb-operator/pull/2045), [@Yisaer](https://github.com/Yisaer))
- 支持为组件配置自定义环境变量  ([#2052](https://github.com/pingcap/tidb-operator/pull/2052), [@cofyc](https://github.com/cofyc))
- 修复 `kubectl get tc` 不能正确显示镜像的问题 ([#2031](https://github.com/pingcap/tidb-operator/pull/2031), [@Yisaer](https://github.com/Yisaer))
- 在 `spec.tikv.maxFailoverCount` 及 `spec.tidb.maxFailoverCount` 未定义时，将其默认值设为 `3` ([#2015](https://github.com/pingcap/tidb-operator/pull/2015), [@Yisaer](https://github.com/Yisaer))
- 在 `maxFailoverCount` 设为 `0` 时禁用自动故障转移的功能  ([#2015](https://github.com/pingcap/tidb-operator/pull/2015), [@Yisaer](https://github.com/Yisaer))
- 支持通过 Terraform 使用TidbCluster 及 TidbMonitor CR 在 ACK 上部署 TiDB 集群  ([#2012](https://github.com/pingcap/tidb-operator/pull/2012), [@DanielZhangQD](https://github.com/DanielZhangQD))
- 将 TidbCluster 中的 PDConfig 升级到 PD v3.1.0 ([#1928](https://github.com/pingcap/tidb-operator/pull/1928), [@Yisaer](https://github.com/Yisaer))
- 支持通过 Terraform 使用 TidbCluster 及 TidbMonitor CR 在 AWS 上部署 TiDB 集群 ([#2004](https://github.com/pingcap/tidb-operator/pull/2004), [@DanielZhangQD](https://github.com/DanielZhangQD))
- 将 TidbCluster 中的 TidbConfig 升级到 TiDB v3.1.0 ([#1906](https://github.com/pingcap/tidb-operator/pull/1906), [@Yisaer](https://github.com/Yisaer))
- 允许用户在 TiDB 初始化时为 initContainers 指定资源 ([#1938](https://github.com/pingcap/tidb-operator/pull/1938), [@tfulcrand](https://github.com/tfulcrand))
- 为 Pump 及 Drainer 添加 TLS 支持 ([#1979](https://github.com/pingcap/tidb-operator/pull/1979), [@weekface](https://github.com/weekface))
- 为 auto-scaler 和 initializer 添加文档与示例 ([#1772](https://github.com/pingcap/tidb-operator/pull/1772), [@Yisaer](https://github.com/Yisaer))
    - 添加检查以保证当 TidbMonitor 的 serviceType 为 NodePort 时，NodePort 不会被改变
    - 添加 EnvVar 排序来避免控制器从同一份 TidbMonitor 规范渲染出不同的结果
    - 修复 TidbMonitor LoadBalancer IP 不被使用的问题 ([#1962](https://github.com/pingcap/tidb-operator/pull/1962), [@Yisaer](https://github.com/Yisaer))
- tidb-initializer 支持 TLS ([#1931](https://github.com/pingcap/tidb-operator/pull/1931), [@weekface](https://github.com/weekface))
    - 修复 Advanced StatefulSet 不能与 webhook 工作的问题
    - 把 Down State TiKV pod 在 webhook 中处理删除请求的响应从允许改为拒绝 ([#1963](https://github.com/pingcap/tidb-operator/pull/1963), [@Yisaer](https://github.com/Yisaer))
- 修复指定 drainerName 时 drainer 的安装错误 ([#1961](https://github.com/pingcap/tidb-operator/pull/1961), [@DanielZhangQD](https://github.com/DanielZhangQD))
- 修正一些 TiKV toml 配置文件中的配置名 ([#1887](https://github.com/pingcap/tidb-operator/pull/1887), [@aylei](https://github.com/aylei))
- 支持使用远程目录作为 tidb-lightning 的数据源 ([#1629](https://github.com/pingcap/tidb-operator/pull/1629), [@aylei](https://github.com/aylei))
- 添加 API 文档以及生成该文档的脚本 ([#1945](https://github.com/pingcap/tidb-operator/pull/1945), [@Yisaer](https://github.com/Yisaer))
- 添加 tikv-importer chart ([#1910](https://github.com/pingcap/tidb-operator/pull/1910), [@shonge](https://github.com/shonge))
- 修复当开启 TLS 时 Prometheus 的 scrape 配置问题 ([#1919](https://github.com/pingcap/tidb-operator/pull/1919), [@weekface](https://github.com/weekface))
- 为 TiDB 组件间的通信开启 TLS ([#1870](https://github.com/pingcap/tidb-operator/pull/1870), [@weekface](https://github.com/weekface))
- 修复在 TiKV 升级过程中当 `Values.admission.validation.pods` 设为 true 时的超时错误 ([#1875](https://github.com/pingcap/tidb-operator/pull/1875), [@Yisaer](https://github.com/Yisaer))
- 为 MySQL 客户端的通信开启 TLS ([#1878](https://github.com/pingcap/tidb-operator/pull/1878), [@weekface](https://github.com/weekface))
- 修复 TiDB 默认配置设置错误的问题 ([#1860](https://github.com/pingcap/tidb-operator/pull/1860), [@Yisaer](https://github.com/Yisaer))
- 如果 targetRef 没定义则使用 TidbMonitor 的 namespace 作为 targetRef ([#1834](https://github.com/pingcap/tidb-operator/pull/1834), [@Yisaer](https://github.com/Yisaer))
- 支持使用 `--advertise-address` 参数启动 tidb-server ([#1859](https://github.com/pingcap/tidb-operator/pull/1859), [@LinuxGit](https://github.com/LinuxGit))
- Backup/Restore: 支持配置 TiKV 的 GC 生命周期 ([#1835](https://github.com/pingcap/tidb-operator/pull/1835), [@LinuxGit](https://github.com/LinuxGit))
- 支持使用 OIDC 对 S3 进行访问鉴权 ([#1817](https://github.com/pingcap/tidb-operator/pull/1817), [@tirsen](https://github.com/tirsen))
    - 把之前的配置 `admission.hookEnabled.pods` 改为 `admission.validation.pods`
    - 把之前的配置 `admission.hookEnabled.statefulSets` 改为 `admission.validation.statefulSets`
    - 把之前的配置 `admission.hookEnabled.validating` 改为 `admission.validation.pingcapResources`
    - 把之前的配置 `admission.hookEnabled.defaulting` 改为 `admission.mutation.pingcapResources`
    - 把之前的配置 `admission.failurePolicy.defaulting` 改为 `admission.failurePolicy.mutation`
    - 把之前的配置 `admission.failurePolicy.*` 改为 `admission.failurePolicy.validation`([#1832](https://github.com/pingcap/tidb-operator/pull/1832), [@Yisaer](https://github.com/Yisaer))
- 默认开启 TidbCluster 的 defaulting mutation，当使用 admission webhook 时推荐开启该开关 ([#1816](https://github.com/pingcap/tidb-operator/pull/1816), [@Yisaer](https://github.com/Yisaer))
- 修复当集群开启 TLS 的情况下使用 CR 创建集群时 TiKV 启动失败的错误 ([#1808](https://github.com/pingcap/tidb-operator/pull/1808), [@weekface](https://github.com/weekface))
- 支持在备份与恢复时在远程存储中使用前缀 ([#1790](https://github.com/pingcap/tidb-operator/pull/1790), [@DanielZhangQD](https://github.com/DanielZhangQD))
