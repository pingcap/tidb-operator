---
title: TiDB in Kubernetes Documentation
summary: Learn about TiDB in Kubernetes documentation.
aliases: ['/docs/tidb-in-kubernetes/dev/']
---

# TiDB in Kubernetes Documentation

[TiDB Operator](https://github.com/pingcap/tidb-operator) is an automatic operation system for TiDB clusters in Kubernetes. It provides a full management life-cycle for TiDB including deployment, upgrades, scaling, backup, fail-over, and configuration changes. With TiDB Operator, TiDB can run seamlessly in the Kubernetes clusters deployed on a public or private cloud.

The corresponding relationship between TiDB Operator and TiDB versions is as follows:

| TiDB Operator version | Compatible TiDB versions |
|:---|:---|
| v1.0 | v2.1, v3.0 |
| v1.1 | v3.0, v3.1, v4.0, v5.0 |
| v1.2 | v3.0 and later releases |
| dev | v3.0 and later releases, dev |

<NavColumns>
<NavColumn>
<ColumnTitle>About TiDB Operator</ColumnTitle>

- [TiDB Operator Overview](tidb-operator-overview.md)
- [What's New in v1.2](whats-new-in-v1.2.md)

</NavColumn>

<NavColumn>
<ColumnTitle>Quick Start</ColumnTitle>

- [kind](get-started.md#create-a-kubernetes-cluster-using-kind)
- [Minikube](get-started.md#create-a-kubernetes-cluster-using-minikube)
- [GKE](deploy-tidb-from-kubernetes-gke.md)

</NavColumn>

<NavColumn>
<ColumnTitle>Deploy TiDB</ColumnTitle>

- [On AWS EKS](deploy-on-aws-eks.md)
- [On GCP GKE](deploy-on-gcp-gke.md)
- [On Alibaba ACK](deploy-on-alibaba-cloud.md)
- [On Self-managed Kubernetes](deploy-on-general-kubernetes.md)
- [Deploy TiFlash](deploy-tiflash.md)
- [Deploy Monitoring Services](monitor-a-tidb-cluster.md)

</NavColumn>

<NavColumn>
<ColumnTitle>Secure</ColumnTitle>

- [Enable TLS for the MySQL Client](enable-tls-for-mysql-client.md)
- [Enable TLS between TiDB Components](enable-tls-between-components.md)
- [Run TiDB Operator and TiDB Clusters as a Non-root User](containers-run-as-non-root-user.md)

</NavColumn>

<NavColumn>
<ColumnTitle>Maintain</ColumnTitle>

- [Upgrade a TiDB Cluster](upgrade-a-tidb-cluster.md)
- [Upgrade TiDB Operator](upgrade-tidb-operator.md)
- [Scale a TiDB Cluster](scale-a-tidb-cluster.md)
- [Backup and Restore](backup-restore-overview.md)
- [Maintain Kubernetes Nodes](maintain-a-kubernetes-node.md)
- [Use Automatic Failover](use-auto-failover.md)

</NavColumn>

<NavColumn>
<ColumnTitle>Reference</ColumnTitle>

- [TiDB Scheduler](tidb-scheduler.md)
- [API Docs](https://github.com/pingcap/tidb-operator/blob/master/docs/api-references/docs.md)
- [Use tkctl](use-tkctl.md)
- [Configure TiDB Binlog Drainer](configure-tidb-binlog-drainer.md)

</NavColumn>

</NavColumns>
