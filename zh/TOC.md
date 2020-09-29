# TiDB in Kubernetes 文档

<!-- markdownlint-disable MD007 -->
<!-- markdownlint-disable MD032 -->

## 文档目录

+ 关于 TiDB Operator
  - [简介](tidb-operator-overview.md)
  - [What's New in v1.1](whats-new-in-v1.1.md)
  - [TiDB Operator v1.1 重要注意事项](notes-tidb-operator-v1.1.md)
+ [快速上手](get-started.md)
+ 部署
  - 部署 TiDB 集群
    - [部署到 AWS EKS](deploy-on-aws-eks.md)
    - [部署到 GCP GKE](deploy-on-gcp-gke.md)
    - [部署到阿里云 ACK](deploy-on-alibaba-cloud.md)
    + 部署到自托管的 Kubernetes
      - [集群环境要求](prerequisites.md)
      - [配置 Storage Class](configure-storage-class.md)
      - [部署 TiDB Operator](deploy-tidb-operator.md)
      - [配置 TiDB 集群](configure-a-tidb-cluster.md)
      - [部署 TiDB 集群](deploy-on-general-kubernetes.md)    
      - [初始化 TiDB 集群](initialize-a-cluster.md)
      - [访问 TiDB 集群](access-tidb.md)
  - [部署异构集群](deploy-heterogeneous-tidb-cluster.md)
  - [部署 TiFlash](deploy-tiflash.md)
  - [部署 TiCDC](deploy-ticdc.md)
  - [部署 TiDB Binlog](deploy-tidb-binlog.md)
  - [部署 TiDB 企业版集群](deploy-tidb-enterprise-edition.md)
  + 部署 TiDB 集群监控
    - [监控 Kubernetes 和 TiDB 集群](monitor-a-tidb-cluster.md)
    - [使用 TidbMonitor 监控 TiDB 集群](monitor-using-tidbmonitor.md)
    - [访问 TiDB Dashboard](access-dashboard.md)
+ 安全
  - [为 MySQL 客户端开启 TLS](enable-tls-for-mysql-client.md)
  - [为 TiDB 组件间开启 TLS](enable-tls-between-components.md)
+ 运维
  - [升级 TiDB 集群](upgrade-a-tidb-cluster.md)
  - [升级 TiDB Operator](upgrade-tidb-operator.md)
  + TiDB 集群伸缩
    - [手动扩缩容](scale-a-tidb-cluster.md)
    - [自动弹性伸缩](enable-tidb-cluster-auto-scaling.md)
  + 备份与恢复
    - [基于 Helm Charts 的备份恢复](backup-and-restore-using-helm-charts.md)
    + 基于 CRD 的备份恢复
      - [使用 Dumpling 备份 TiDB 集群数据到 GCS](backup-to-gcs.md)
      - [使用 TiDB Lightning 恢复 GCS 上的备份数据](restore-from-gcs.md)
      - [使用 Dumpling 备份 TiDB 集群数据到兼容 S3 的存储](backup-to-s3.md)
      - [使用 TiDB Lightning 恢复 S3 兼容存储上的备份数据](restore-from-s3.md)
      - [使用 BR 备份 TiDB 集群数据到 GCS](backup-to-gcs-using-br.md)
      - [使用 BR 恢复 GCS 上的备份数据](restore-from-gcs-using-br.md)
      - [使用 BR 备份 TiDB 集群数据到兼容 S3 的存储](backup-to-aws-s3-using-br.md)
      - [使用 BR 恢复 S3 兼容存储上的备份数据](restore-from-aws-s3-using-br.md)
  - [重启 TiDB 集群](restart-a-tidb-cluster.md)
  - [维护 TiDB 集群所在节点](maintain-a-kubernetes-node.md)
  - [查看日志](view-logs.md)
  - [集群故障自动转移](use-auto-failover.md)  
  - [销毁 TiDB 集群](destroy-a-tidb-cluster.md)
+ 灾难恢复
  - [恢复 PD 集群](pd-recover.md)
  - [恢复误删的集群](recover-deleted-cluster.md)
- [导入集群数据](restore-data-using-tidb-lightning.md)
+ 故障诊断
  - [使用技巧](tips.md)
  - [部署错误](deploy-failures.md)
  - [集群异常](exceptions.md)
  - [网络问题](network-issues.md)
- [常见问题](faq.md)
+ 参考
  + 架构
    - [TiDB Operator 架构](architecture.md)
    - [TiDB Scheduler 扩展调度器](tidb-scheduler.md)
    - [增强型 StatefulSet 控制器](advanced-statefulset.md)
    - [准入控制器](enable-admission-webhook.md)
  - [Sysbench 性能测试](benchmark-sysbench.md)
  - [API 参考文档](https://github.com/pingcap/tidb-operator/blob/master/docs/api-references/docs.md)
  - [Cheat Sheet](cheat-sheet.md)
  + 工具
    - [tkctl](use-tkctl.md)
    - [TiDB Toolkit](tidb-toolkit.md)
  + 配置
    - [tidb-drainer chart 配置](configure-tidb-binlog-drainer.md)
    - [tidb-cluster chart 配置](tidb-cluster-chart-config.md)
    - [tidb-backup chart 配置](configure-backup.md)
  - [日志收集](logs-collection.md)
+ 版本发布历史
  + v1.1
    - [1.1.5](releases/release-1.1.5.md)
    - [1.1.4](releases/release-1.1.4.md)
    - [1.1.3](releases/release-1.1.3.md)
    - [1.1.2](releases/release-1.1.2.md)
    - [1.1.1](releases/release-1.1.1.md)
    - [1.1 GA](releases/release-1.1-ga.md)
    - [1.1.0-rc.4](releases/release-1.1.0-rc.4.md)
    - [1.1.0-rc.3](releases/release-1.1.0-rc.3.md)
    - [1.1.0-rc.2](releases/release-1.1.0-rc.2.md)
    - [1.1.0-rc.1](releases/release-1.1.0-rc.1.md)
    - [1.1.0-beta.2](releases/release-1.1.0-beta.2.md)
    - [1.1.0-beta.1](releases/release-1.1.0-beta.1.md)
  + v1.0
    - [1.0.7](releases/release-1.0.7.md)
    - [1.0.6](releases/release-1.0.6.md)
    - [1.0.5](releases/release-1.0.5.md)
    - [1.0.4](releases/release-1.0.4.md)
    - [1.0.3](releases/release-1.0.3.md)
    - [1.0.2](releases/release-1.0.2.md)
    - [1.0.1](releases/release-1.0.1.md)
    - [1.0 GA](releases/release-1.0-ga.md)
    - [1.0.0-rc.1](releases/release-1.0.0-rc.1.md)
    - [1.0.0-beta.3](releases/release-1.0.0-beta.3.md)
    - [1.0.0-beta.2](releases/release-1.0.0-beta.2.md)
    - [1.0.0-beta.1-p2](releases/release-1.0.0-beta.1-p2.md)
    - [1.0.0-beta.1-p1](releases/release-1.0.0-beta.1-p1.md)
    - [1.0.0-beta.1](releases/release-1.0.0-beta.1.md)
    - [1.0.0-beta.0](releases/release-1.0.0-beta.0.md)
  + v0
    - [0.4.0](releases/release-0.4.0.md)
    - [0.3.1](releases/release-0.3.1.md)
    - [0.3.0](releases/release-0.3.0.md)
    - [0.2.1](releases/release-0.2.1.md)
    - [0.2.0](releases/release-0.2.0.md)
    - [0.1.0](releases/release-0.1.0.md)
