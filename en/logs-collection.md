---
title: TiDB Log Collection in Kubernetes
summary: Learn the methods of collecting logs of TiDB and its related components.
aliases: ['/docs/tidb-in-kubernetes/dev/collect-tidb-logs/']
---

# TiDB Log Collection in Kubernetes

The system and application logs can be useful for troubleshooting issues and automating operations. This article briefly introduces the methods of collecting logs of TiDB and its related components.

## Collect logs of TiDB and Kubernetes components

The TiDB components deployed by TiDB Operator output the logs in the `stdout` and `stderr` of the container by default. For Kubernetes, these logs are stored in the host's `/var/log/containers` directory, and the file name contains information such as the Pod name and the container name. For this reason, you can collect the logs of the application in the container directly on the host.

If you already have a system for collecting logs in your existing infrastructure, you only need to add the `/var/log/containers/*.log` file on the host in which Kubernetes is located in the collection scope by common methods; if there is no available log collection system, or you want to deploy a separate system for collecting relevant logs, you are free to use any system or solution that you are familiar with.

The Kubernetes official documentation provides [Stackdriver](https://kubernetes.io/docs/tasks/debug-application-cluster/logging-stackdriver/) as a log collection method.

Common open source tools that can be used to collect Kubernetes logs are:

- [Fluentd](https://www.fluentd.org/)
- [Fluent-bit](https://fluentbit.io/)
- [Filebeat](https://www.elastic.co/products/beats/filebeat)
- [Logstash](https://www.elastic.co/products/logstash)

Collected Logs can usually be aggregated and stored on a specific server or in a dedicated storage and analysis system such as ElasticSearch.

Some cloud service providers or specialized performance monitoring service providers also have their own free or charged log collection options that you can choose from.

## Collect system logs

System logs can be collected on Kubernetes hosts in the usual way. If you already have a system for collecting logs in your existing infrastructure, you only need to add the relevant servers and log files in the collection scope by conventional means; if there is no available log collection system, or you want to deploy a separate set of systems for collecting relevant logs, you are free to use any system or solution that you are familiar with.

All of the common log collection tools mentioned above support collecting system logs. Some cloud service providers or specialized performance monitoring service providers also have their own free or charged log collection options that you can choose from.
