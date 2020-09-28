# TiDB Monitor Refactor

## Summary

This document presents a design to refactor tidb monitor,solve some problems in monitoring uniformly.

## Motivation

Currently, it is not friendly that tidbmonitor monitor multiple cluster under the same namespaces or different namespaces.In a multi-cluster TLS environment, the cert of tidbmonitor is bound to the first cluster of the cluster,we should eliminate this binding relationship.
Secondly,tidbmonitor is also a statefulset application,we should switch to statefulset deploy from deployment crd,this process should smooth upgrade.
Also, many users will integrate the company’s existing monitoring system, we should support thanos sidecar spec refer to the Prometheus operator, and optimize metric service and provide googe yaml example like `ServiceMonitor` ,`thanos` etc.
Others are some small optimization points,providing more field for `kubectl get tidbmonitor`  command, supporting multiple cluster grafana dashborad,optimizing pd dashboard address writing logic,fulling e2e and unit testing etc.

### Goals

* Make TidbMonitor able to monitor multiple TidbClusters at non-TLS environment.
* Improve the monitor's Unit and E2E test.
* Smooth upgrade deployment to statefulset.
* Support thanos sidecar and more friendly integration of Prometheus operator.

### Non-Goals


## Proposal


* Check tidbmonitor monitor multiple clusters cross multiple ns in tls and non-tls environments.
* Support multiple cluster grafana dashboard.
	
* Smooth upgrade deployment to statefulset.
* Support thanos spec and optimize thanos example.
* Optimze tidb service,more friendly support prometheus operator `ServiceMonitor`.
* Refactor for pd dashboard address writing.


* Fix combination of Kubernetes monitoring and TiDB cluster monitoring.
* Add more external field for  `kubectl get tm` command.


### Test Plan

* Deploy tidbmonitor monitor multiple clusters cross multiple ns in tls and non-tls environments.
* Deploy tidbmonitor monitor multiple clusters cross same ns in tls and non-tls environments.
* Deploy tidbmonitor monitor smoothly upgrade to statefulset.
* Deploy tidbmointor with thanos sidecar,checking thanos metric data.
* Use prometheus operator `ServiceMonitor` monitor tidb.

## Drawbacks

* Make monitoring easier to use.

## Alternatives
