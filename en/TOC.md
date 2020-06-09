# TiDB in Kubernetes Documentation

<!-- markdownlint-disable MD007 -->
<!-- markdownlint-disable MD032 -->

## TOC

+ Introduction
  - [Overview](tidb-operator-overview.md)
  - [TiDB Operator v1.1 Notice](notes-tidb-operator-v1.1.md)
+ Get Started
  - [kind](deploy-tidb-from-kubernetes-kind.md)
  - [GKE](deploy-tidb-from-kubernetes-gke.md)
  - [Minikube](deploy-tidb-from-kubernetes-minikube.md)
+ Deploy
  - Deploy TiDB Cluster
    - [On AWS EKS](deploy-on-aws-eks.md)
    - [On GCP GKE](deploy-on-gcp-gke.md)
    - [On Alibaba Cloud ACK](deploy-on-alibaba-cloud.md)
    + On Self-managed Kubernetes
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
      - [Back up Data to GCS Using Mydumper](backup-to-gcs.md)
      - [Restore Data from GCS Using TiDB Lightning](restore-from-gcs.md)
      - [Back up Data to S3-Compatible Storage Using Mydumper](backup-to-s3.md)
      - [Restore Data from S3-Compatible Storage Using TiDB Lightning](restore-from-s3.md)
      - [Back up Data to S3-Compatible Storage Using BR](backup-to-aws-s3-using-br.md)
      - [Restore Data from S3-Compatible Storage Using BR](restore-from-aws-s3-using-br.md)
  - [Restart a TiDB Cluster](restart-a-tidb-cluster.md)
  - [Maintain a Kubernetes Node](maintain-a-kubernetes-node.md)
  - [Collect TiDB Logs](collect-tidb-logs.md)
  - [Enable Automatic Failover](use-auto-failover.md)
  - [Recover the PD Cluster](pd-recover.md)
  - [Destroy a TiDB Cluster](destroy-a-tidb-cluster.md)
- [Import Data](restore-data-using-tidb-lightning.md)
- [Troubleshoot](troubleshoot.md)
- [FAQs](faq.md)
+ Reference
  + Architecture
    - [TiDB Scheduler](tidb-scheduler.md)
    - [Advanced StatefulSet Controller](advanced-statefulset.md)
    - [Admission Controller](enable-admission-webhook.md)
  - [Sysbench Performance Test](benchmark-sysbench.md)
  - [API References](https://github.com/pingcap/docs-tidb-operator/blob/master/zh/api-references.md)
  + Tools
    - [tkctl](use-tkctl.md)
    - [TiDB Toolkit](tidb-toolkit.md)
  + Configure
    - [Configure tidb-drainer Chart](configure-tidb-binlog-drainer.md)
    - [Configure tidb-cluster Chart](tidb-cluster-chart-config.md)
    - [Configure tidb-backup Chart](configure-backup.md)
