# TiDB Monitor Refactor
## Summary
This document presents a design to refactor TiDB Monitor architecture to solve some issues in monitoring and do some improvements.
## Motivation
Fix the issues mentioned in https://github.com/pingcap/tidb-operator/issues/3292#issue-705145308.
### Goals
* Make TidbMonitor able to monitor multiple TidbClusters in non-TLS environment.
* Improve the monitor's Unit and E2E test.
* Smooth upgrade from deployment to StatefulSet.
* Support Thanos sidecar and more friendly integration of Prometheus operator.
### Non-Goals
## Proposal
- Tidbmonitor monitors multiple clusters across multiple namespaces in non-TLS environments.
    - Support `ClusterScoped` field in spec,it will use clusterRole and clusterRoleBinding.
- Support generating the Grafana dashboard of multiple clusters.
    - Repaint the multi-clusters dashboard,users can select the dashboard of the specified cluster.
- Smooth upgrade from deployment to StatefulSet.
    - If user create a new cluster,use statefulset deploy directly.If user upgrade operator with existing cluster exist,we need to delete deployment firstly,then change pv binding to new statefulset pvc and start statefulset.
- Support Thanos spec and optimize Thanos example.
    - Support thanos definition contain `version` and `baseImage` field.
- Optimize TiDB service, more friendly support with Prometheus operator `ServiceMonitor`.
    - User can easier create `ServiceMonitor` easier to scrape tidb metrics data.Due to `ServiceMonitor` select service endpoints,so we need to optimize tidb services.
- Refactor for PD dashboard address writing.
    - The PD Dashboard monitor address is currently TidbCluster controller updated, it should be updated by tidbmonitor controller. And PD dashboard doesn't support multiple clusters.
- Fix the combination of Kubernetes monitoring and TiDB cluster monitoring.
    - https://github.com/pingcap/tidb-operator/issues/3099
- Add more external fields for `kubectl get tm` command.
    - https://github.com/pingcap/tidb-operator/issues/2169
### Test Plan
* Deploy Tidbmonitor to monitor a TiDB cluster, StatefulSet and PVC are automatically created and monitoring works as expected.
* Deploy Tidbmonitor to monitor a TiDB cluster with old version TiDB Operator, after upgrading to the new version, StatefulSet is created, Pod is bound to the old PVC and monitoring works as expected, the old monitoring data can also be retrieved.
* Be able to monitor multiple TiDB Clusters with one TidbMonitor with TLS disabled
* Be able to monitor multiple TiDB Clusters with users own Prometheus
* Be able to aggregate the monitoring data of multiple TiDB Clusters with TLS disabled
* Be able to aggregate the monitoring data of multiple TiDB Clusters with TLS enabled
* All of the dashboards of TidbMonitor can show correctly by default especially the ones from Kubernetes monitoring
* Show reasonable and useful fields for `kubectl get tm` command
* The TiDB dashboard works as expected except for the non-supported functions in the official doc
* Grafana dashboards are shown clearly for different clusters
## Drawbacks
## Alternatives

