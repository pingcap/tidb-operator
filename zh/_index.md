---
title: TiDB in Kubernetes 用户文档
summary: 了解 TiDB in Kubernetes 的用户文档。
aliases: ['/docs-cn/tidb-in-kubernetes/dev/']
---

<!-- markdownlint-disable MD046 -->

# TiDB in Kubernetes 用户文档

[TiDB Operator](https://github.com/pingcap/tidb-operator) 是 Kubernetes 上的 TiDB 集群自动运维系统，提供包括部署、升级、扩缩容、备份恢复、配置变更的 TiDB 全生命周期管理。借助 TiDB Operator，TiDB 可以无缝运行在公有云或私有部署的 Kubernetes 集群上。

TiDB Operator 与适用的 TiDB 版本的对应关系如下：

| TiDB Operator 版本 | 适用的 TiDB 版本 |
|:---|:---|
| v1.0 | v2.1, v3.0 |
| v1.1 | v3.0, v3.1, v4.0, v5.0 |
| dev | v3.0, v3.1, v4.0, v5.0, dev |

<NavColumns>
<NavColumn>
<ColumnTitle>关于 TiDB Operator</ColumnTitle>

- [TiDB Operator 简介](tidb-operator-overview.md)
- [What's New in v1.1](whats-new-in-v1.1.md)
- [v1.1 重要注意事项](notes-tidb-operator-v1.1.md)

</NavColumn>

<NavColumn>
<ColumnTitle>快速上手</ColumnTitle>

- [kind](get-started.md#使用-kind-创建-kubernetes-集群)
- [Minikube](get-started.md#使用-minikube-创建-kubernetes-集群)
- [Google Cloud Shell](https://console.cloud.google.com/cloudshell/open?cloudshell_git_repo=https://github.com/pingcap/docs-tidb-operator&cloudshell_tutorial=zh/deploy-tidb-from-kubernetes-gke.md)

</NavColumn>

<NavColumn>
<ColumnTitle>部署集群</ColumnTitle>

- [部署到 AWS EKS](deploy-on-aws-eks.md)
- [部署到 GCP GKE](deploy-on-gcp-gke.md)
- [部署到阿里云 ACK](deploy-on-alibaba-cloud.md)
- [部署到自托管的 Kubernetes](prerequisites.md)
- [部署 TiFlash](deploy-tiflash.md)
- [部署 TiDB 集群监控](monitor-a-tidb-cluster.md)

</NavColumn>

<NavColumn>
<ColumnTitle>安全</ColumnTitle>

- [为 MySQL 客户端开启 TLS](enable-tls-for-mysql-client.md)
- [为 TiDB 组件间开启 TLS](enable-tls-between-components.md)
- [以非 root 用户运行 TiDB Operator 和 TiDB 集群](containers-run-as-non-root-user.md)

</NavColumn>

<NavColumn>
<ColumnTitle>运维</ColumnTitle>

- [升级 TiDB 集群](upgrade-a-tidb-cluster.md)
- [升级 TiDB Operator](upgrade-tidb-operator.md)
- [集群扩缩容](scale-a-tidb-cluster.md)
- [备份与恢复](backup-restore-overview.md)
- [维护 TiDB 集群所在节点](maintain-a-kubernetes-node.md)
- [集群故障自动转移](use-auto-failover.md)

</NavColumn>

<NavColumn>
<ColumnTitle>参考</ColumnTitle>

- [架构](tidb-scheduler.md)
- [API 参考文档](https://github.com/pingcap/tidb-operator/blob/master/docs/api-references/docs.md)
- [工具](use-tkctl.md)
- [配置](configure-tidb-binlog-drainer.md)

</NavColumn>

</NavColumns>
