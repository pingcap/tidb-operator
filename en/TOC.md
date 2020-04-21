# TiDB in Kubernetes Documentation

<!-- markdownlint-disable MD007 -->
<!-- markdownlint-disable MD032 -->

## TOC

- [About TiDB Operator](tidb-operator-overview.md)
+ Benchmark
  - [Sysbench](benchmark-sysbench.md)
+ Get Started
  - [kind](deploy-tidb-from-kubernetes-kind.md)
  - [GKE](deploy-tidb-from-kubernetes-gke.md)
  - [Minikube](deploy-tidb-from-kubernetes-minikube.md)
+ Deploy
  - [Prerequisites](prerequisites.md)
  - [TiDB Operator](deploy-tidb-operator.md)
  - [TiDB in General Kubernetes](deploy-on-general-kubernetes.md)
  - [TiDB in AWS EKS](deploy-on-aws-eks.md)
  - [TiDB in GCP GKE](deploy-on-gcp-gke.md)
  - [TiDB in Alibaba Cloud ACK](deploy-on-alibaba-cloud.md)
  - [Access TiDB in Kubernetes](access-tidb.md)
  - [Deploy TiDB Binlog](deploy-tidb-binlog.md)
+ Configure
  - [Configure Storage Class](configure-storage-class.md)
  - [Configure Resource and Disaster Recovery](configure-a-tidb-cluster.md)
  - [Initialize a Cluster](initialize-a-cluster.md)
  - [Configure a TiDB Cluster using TidbCluster](configure-cluster-using-tidbcluster.md)
  - [Configure tidb-drainer Chart](configure-tidb-binlog-drainer.md)
  - [Configure tidb-cluster Chart](tidb-cluster-chart-config.md)
  - [Configure tidb-backup Chart](configure-backup.md)
+ Monitor
  - [Monitor TiDB Using Helm](monitor-a-tidb-cluster.md)
  - [Monitor TiDB Using TidbMonitor](monitor-using-tidbmonitor.md)
+ Maintain
  - [Destroy a TiDB cluster](destroy-a-tidb-cluster.md)
  - [Restart a TiDB Cluster](restart-a-tidb-cluster.md)
  - [Maintain a Hosting Kubernetes Node](maintain-a-kubernetes-node.md)
  - [Collect TiDB Logs](collect-tidb-logs.md)
  - [Enable Automatic Failover](use-auto-failover.md)
  - [Enable Admission Controller](enable-admission-webhook.md)
+ Scale
  - [Scale](scale-a-tidb-cluster.md)
  - [Enable Auto-scaling](enable-tidb-cluster-auto-scaling.md)
+ Backup and Restore
  - [Use Helm Charts](backup-and-restore-using-helm-charts.md)
  + Use CRDs
    - [Back up Data to GCS](backup-to-gcs.md)
    - [Restore Data from GCS](restore-from-gcs.md)
    - [Back up Data to S3-Compatible Storage Using Mydumper](backup-to-s3.md)
    - [Restore Data from S3-Compatible Storage Using Loader](restore-from-s3.md)
    - [Back up Data to S3-Compatible Storage Using BR](backup-to-aws-s3-using-br.md)
    - [Restore Data from S3-Compatible Storage Using BR](restore-from-aws-s3-using-br.md)
  - [Restore Data Using TiDB Lightning](restore-data-using-tidb-lightning.md)
+ Upgrade
  - [TiDB Cluster](upgrade-a-tidb-cluster.md)
  - [TiDB Operator](upgrade-tidb-operator.md)
  - [TiDB Operator v1.1 Notice](notes-tidb-operator-v1.1.md)
+ Security
  - [Enable TLS for the MySQL Client](enable-tls-for-mysql-client.md)
  - [Enable TLS between TiDB Components](enable-tls-between-components.md)
+ Tools
  - [tkctl](use-tkctl.md)
  - [TiDB Toolkit](tidb-toolkit.md)
+ Components
  - [TiDB Scheduler](tidb-scheduler.md)
  - [Advanced StatefulSet Controller](advanced-statefulset.md)
- [Troubleshoot](troubleshoot.md)
- [FAQs](faq.md)
- [API References](api-references.md)
