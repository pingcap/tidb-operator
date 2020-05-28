# TiDB in Kubernetes 文档

<!-- markdownlint-disable MD007 -->
<!-- markdownlint-disable MD032 -->

## 文档目录

+ 快速上手
  - [kind](deploy-tidb-from-kubernetes-kind.md)
  - [GKE](deploy-tidb-from-kubernetes-gke.md)
  - [Minikube](deploy-tidb-from-kubernetes-minikube.md)
+ 关于 TiDB Operator
  - [简介](tidb-operator-overview.md)
  - [TiDB Operator v1.1 重要注意事项](notes-tidb-operator-v1.1.md)
+ 部署
  - 部署 TiDB 集群
    - [部署到 AWS EKS](deploy-on-aws-eks.md)
    - [部署到 GCP GKE](deploy-on-gcp-gke.md)
    - [部署到阿里云 ACK](deploy-on-alibaba-cloud.md)
    + 部署到自托管的 Kubernetes
      - [集群环境要求](prerequisites.md)
      - [配置 Storage Class](configure-storage-class.md)
      - [部署 TiDB Operator](deploy-tidb-operator.md)
      - [资源及容灾配置](configure-a-tidb-cluster.md)
      - [通过 TidbCluster 配置集群](configure-cluster-using-tidbcluster.md)
      - [部署 TiDB 集群](deploy-on-general-kubernetes.md)
      - [初始化 TiDB 集群](initialize-a-cluster.md)
      - [访问 TiDB 集群](access-tidb.md)
  - [部署 TiFlash](deploy-tiflash.md)
  - [部署 TiCDC](deploy-ticdc.md)
  - [部署 TiDB Binlog](deploy-tidb-binlog.md)
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
      - [使用 Mydumper 备份 TiDB 集群数据到 GCS](backup-to-gcs.md)
      - [使用 TiDB Lightning 恢复 GCS 上的备份数据](restore-from-gcs.md)
      - [使用 Mydumper 备份 TiDB 集群数据到兼容 S3 的存储](backup-to-s3.md)
      - [使用 TiDB Lightning 恢复 S3 兼容存储上的备份数据](restore-from-s3.md)
      - [使用 BR 备份 TiDB 集群数据到兼容 S3 的存储](backup-to-aws-s3-using-br.md)
      - [使用 BR 恢复 S3 兼容存储上的备份数据](restore-from-aws-s3-using-br.md)
  - [重启 TiDB 集群](restart-a-tidb-cluster.md)
  - [维护 TiDB 集群所在节点](maintain-a-kubernetes-node.md)
  - [收集日志](collect-tidb-logs.md)
  - [集群故障自动转移](use-auto-failover.md)
  - [恢复 PD 集群](pd-recover.md)
  - [销毁 TiDB 集群](destroy-a-tidb-cluster.md)
- [导入集群数据](restore-data-using-tidb-lightning.md)
- [故障诊断](troubleshoot.md)
- [常见问题](faq.md)
+ 参考
  + 架构
    - [TiDB Scheduler 扩展调度器](tidb-scheduler.md)
    - [增强型 StatefulSet 控制器](advanced-statefulset.md)
    - [准入控制器](enable-admission-webhook.md)
  - [API 参考文档](https://github.com/pingcap/docs-tidb-operator/blob/master/zh/api-references.md)
  + 工具
    - [tkctl](use-tkctl.md)
    - [TiDB Toolkit](tidb-toolkit.md)
  + 配置
    - [tidb-drainer chart 配置](configure-tidb-binlog-drainer.md)
    - [tidb-cluster chart 配置](tidb-cluster-chart-config.md)
    - [tidb-backup chart 配置](configure-backup.md)
