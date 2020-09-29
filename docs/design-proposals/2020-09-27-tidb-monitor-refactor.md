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
* Improve the robustness of tidbmonitor,for example multiple cluster,code level.
* Meet most of the needs of users and make monitoring easier to use.
## Proposal
- Tidbmonitor monitors multiple clusters across multiple ns in non-TLS environments.
- Support generating the Grafana dashboard of multiple clusters.
- Smooth upgrade from deployment to StatefulSet.
- Support Thanos spec and optimize Thanos example.
- Optimize TiDB service, more friendly support with Prometheus operator `ServiceMonitor`.
- Refactor for PD dashboard address writing.
- Fix the combination of Kubernetes monitoring and TiDB cluster monitoring.
- Add more external fields for `kubectl get tm` command.
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

