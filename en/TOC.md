# TiDB in Kubernetes Documentation

<!-- markdownlint-disable MD007 -->
<!-- markdownlint-disable MD032 -->

## TOC

+ Introduction
  - [Overview](tidb-operator-overview.md)
  - [What's New in v1.1](whats-new-in-v1.1.md)
  - [TiDB Operator v1.1 Notice](notes-tidb-operator-v1.1.md)
- [Get Started](get-started.md)
+ Deploy
  - Deploy TiDB Cluster
    - [On AWS EKS](deploy-on-aws-eks.md)
    - [On GCP GKE](deploy-on-gcp-gke.md)
    - [On Alibaba Cloud ACK](deploy-on-alibaba-cloud.md)
    + In Self-managed Kubernetes
      - [Prerequisites](prerequisites.md)
      - [Configure Storage Class](configure-storage-class.md)
      - [Deploy TiDB Operator](deploy-tidb-operator.md)
      - [Configure TiDB Cluster](configure-a-tidb-cluster.md)
      - [Deploy TiDB Cluster](deploy-on-general-kubernetes.md)
      - [Initialize TiDB Cluster](initialize-a-cluster.md)
      - [Access TiDB Cluster](access-tidb.md)
  - [Deploy TiFlash](deploy-tiflash.md)
  - [Deploy TiCDC](deploy-ticdc.md)
  - [Deploy TiDB Binlog](deploy-tidb-binlog.md)
  - [Deploy TiDB Enterprise Edition](deploy-tidb-enterprise-edition.md)
  + Deploy Monitoring
    - [Monitor Kubernetes and TiDB Cluster](monitor-a-tidb-cluster.md)
    - [Monitor TiDB Cluster Using TidbMonitor](monitor-using-tidbmonitor.md)
    - [Access TiDB Dashboard](access-dashboard.md)
+ Secure
  - [Enable TLS for the MySQL Client](enable-tls-for-mysql-client.md)
  - [Enable TLS between TiDB Components](enable-tls-between-components.md)
+ Operate
  - [Upgrade TiDB Cluster](upgrade-a-tidb-cluster.md)
  - [Upgrade TiDB Operator](upgrade-tidb-operator.md)
  + Scale TiDB Cluster
    - [Manually Scale](scale-a-tidb-cluster.md)
    - [Automatically Scale](enable-tidb-cluster-auto-scaling.md)
  + Backup and Restore
    - [Use Helm Charts](backup-and-restore-using-helm-charts.md)
    + Use CRDs
      - [Back up Data to GCS Using Dumpling](backup-to-gcs.md)
      - [Restore Data from GCS Using TiDB Lightning](restore-from-gcs.md)
      - [Back up Data to S3-Compatible Storage Using Dumpling](backup-to-s3.md)
      - [Restore Data from S3-Compatible Storage Using TiDB Lightning](restore-from-s3.md)
      - [Back up Data to GCS Using BR](backup-to-gcs-using-br.md)
      - [Restore Data from GCS Using BR](restore-from-gcs-using-br.md)
      - [Back up Data to S3-Compatible Storage Using BR](backup-to-aws-s3-using-br.md)
      - [Restore Data from S3-Compatible Storage Using BR](restore-from-aws-s3-using-br.md)
  - [Restart a TiDB Cluster](restart-a-tidb-cluster.md)
  - [Maintain a Kubernetes Node](maintain-a-kubernetes-node.md)
  - [View TiDB Logs](view-logs.md)
  - [Enable Automatic Failover](use-auto-failover.md)
  - [Destroy a TiDB Cluster](destroy-a-tidb-cluster.md)
+ Disaster Recovery
  - [Recover PD Cluster](pd-recover.md)
  - [Recover Deleted Cluster](recover-deleted-cluster.md)
- [Import Data](restore-data-using-tidb-lightning.md)
+ Troubleshoot
  - [Troubleshooting Tips](tips.md)
  - [Deployment Failures](deploy-failures.md)
  - [Cluster Exceptions](exceptions.md)
  - [Network Issues](network-issues.md)
- [FAQs](faq.md)
+ Reference
  + Architecture
    - [TiDB Operator](architecture.md)
    - [TiDB Scheduler](tidb-scheduler.md)
    - [Advanced StatefulSet Controller](advanced-statefulset.md)
    - [Admission Controller](enable-admission-webhook.md)
  - [Sysbench Performance Test](benchmark-sysbench.md)
  - [API References](https://github.com/pingcap/tidb-operator/blob/master/docs/api-references/docs.md)
  - [Cheat Sheet](cheat-sheet.md)
  + Tools
    - [tkctl](use-tkctl.md)
    - [TiDB Toolkit](tidb-toolkit.md)
  + Configure
    - [Configure tidb-drainer Chart](configure-tidb-binlog-drainer.md)
    - [Configure tidb-cluster Chart](tidb-cluster-chart-config.md)
    - [Configure tidb-backup Chart](configure-backup.md)
  - [Log Collection](logs-collection.md)
+ Release Notes
  + v1.1
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
