---
title: TiDB Operator 简介
summary: 介绍 TiDB Operator 的整体架构及使用方式。
aliases: ['/docs-cn/tidb-in-kubernetes/dev/tidb-operator-overview/']
---

# TiDB Operator 简介

[TiDB Operator](https://github.com/pingcap/tidb-operator) 是 Kubernetes 上的 TiDB 集群自动运维系统，提供包括部署、升级、扩缩容、备份恢复、配置变更的 TiDB 全生命周期管理。借助 TiDB Operator，TiDB 可以无缝运行在公有云或私有部署的 Kubernetes 集群上。

> **注意：**
>
> 每个 Kubernetes 集群中只能部署一个 TiDB Operator。

TiDB Operator 与适用的 TiDB 版本的对应关系如下：

| TiDB Operator 版本 | 适用的 TiDB 版本 |
|:---|:---|
| v1.0 | v2.1, v3.0 |
| v1.1 | v3.0, v3.1, v4.0, v5.0 |
| dev | v3.0, v3.1, v4.0, v5.0, dev |

## 使用 TiDB Operator 管理 TiDB 集群

TiDB Operator 提供了多种方式来部署 Kubernetes 上的 TiDB 集群：

+ 测试环境：

    - [kind](get-started.md#使用-kind-创建-kubernetes-集群)
    - [Minikube](get-started.md#使用-minikube-创建-kubernetes-集群)
    - [Google Cloud Shell](https://console.cloud.google.com/cloudshell/open?cloudshell_git_repo=https://github.com/pingcap/docs-tidb-operator&cloudshell_tutorial=zh/deploy-tidb-from-kubernetes-gke.md)

+ 生产环境：

    - 公有云：参考 [AWS 部署文档](deploy-on-aws-eks.md)，[GKE 部署文档 (beta)](deploy-on-gcp-gke.md)，或[阿里云部署文档](deploy-on-alibaba-cloud.md)在对应的公有云上一键部署生产可用的 TiDB 集群并进行后续的运维管理；

    - 现有 Kubernetes 集群：首先按照[部署 TiDB Operator](deploy-tidb-operator.md)在集群中安装 TiDB Operator，再根据[在标准 Kubernetes 集群上部署 TiDB 集群](deploy-on-general-kubernetes.md)来部署你的 TiDB 集群。对于生产级 TiDB 集群，你还需要参考 [TiDB 集群环境要求](prerequisites.md)调整 Kubernetes 集群配置并根据[本地 PV 配置](configure-storage-class.md#本地-pv-配置)为你的 Kubernetes 集群配置本地 PV，以满足 TiKV 的低延迟本地存储需求。

在任何环境上部署前，都可以参考 [TiDB 集群配置](configure-a-tidb-cluster.md)来自定义 TiDB 配置。

部署完成后，你可以参考下面的文档进行 Kubernetes 上 TiDB 集群的使用和运维：

+ [部署 TiDB 集群](deploy-on-general-kubernetes.md)
+ [访问 TiDB 集群](access-tidb.md)
+ [TiDB 集群扩缩容](scale-a-tidb-cluster.md)
+ [TiDB 集群升级](upgrade-a-tidb-cluster.md#升级-tidb-版本)
+ [TiDB 集群配置变更](configure-a-tidb-cluster.md)
+ [TiDB 集群备份与恢复](backup-restore-overview.md)
+ [配置 TiDB 集群故障自动转移](use-auto-failover.md)
+ [监控 TiDB 集群](monitor-a-tidb-cluster.md)
+ [查看 TiDB 日志](view-logs.md)
+ [维护 TiDB 所在的 Kubernetes 节点](maintain-a-kubernetes-node.md)

当集群出现问题需要进行诊断时，你可以：

+ 查阅 [Kubernetes 上的 TiDB FAQ](faq.md) 寻找是否存在现成的解决办法；
+ 参考 [Kubernetes 上的 TiDB 故障诊断](tips.md)解决故障。

Kubernetes 上的 TiDB 提供了专用的命令行工具 `tkctl` 用于集群管理和辅助诊断，同时，在 Kubernetes 上，TiDB 的部分生态工具的使用方法也有所不同，你可以：

+ 参考 [`tkctl` 使用指南](use-tkctl.md) 来使用 `tkctl`；
+ 参考 [Kubernetes 上的 TiDB 相关工具使用指南](tidb-toolkit.md)来了解 TiDB 生态工具在 Kubernetes 上的使用方法。

最后，当 TiDB Operator 发布新版本时，你可以参考[升级 TiDB Operator](upgrade-tidb-operator.md) 进行版本更新。
