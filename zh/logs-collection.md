---
title: 日志收集
summary: 介绍收集 TiDB 及相关组件日志的方法。
aliases: ['/docs-cn/tidb-in-kubernetes/dev/logs-collection/','/docs-cn/tidb-in-kubernetes/dev/collect-tidb-logs/']
---

# 日志收集

系统与程序的运行日志对排查问题以及实现一些自动化操作可能非常有用。本文将简要说明收集 TiDB 及相关组件日志的方法。

## TiDB 与 Kubernetes 组件运行日志

通过 TiDB Operator 部署的 TiDB 各组件默认将日志输出在容器的 `stdout` 和 `stderr` 中。对于 Kubernetes 而言，这些日志会被存放在宿主机的 `/var/log/containers` 目录下，并且文件名中包含了 Pod 和容器名称等信息。因此，可以直接在宿主机上收集容器中应用的日志。

如果在你的现有基础设施中已经有用于收集日志的系统，只需要通过常规方法将 Kubernetes 所在的宿主机上的 `/var/log/containers/*.log` 文件加入采集范围即可；如果没有可用的日志收集系统，或者希望部署一套独立的系统用于收集相关日志，也可以使用你熟悉的任意日志收集系统或方案。

常见的可用于收集 Kubernetes 日志的开源工具有：

- [Fluentd](https://www.fluentd.org/)
- [Fluent-bit](https://fluentbit.io/)
- [Filebeat](https://www.elastic.co/products/beats/filebeat)
- [Logstash](https://www.elastic.co/products/logstash)

收集到的日志通常可以汇总存储在某一特定的服务器上，或存放到 ElasticSearch 等专用的存储、分析系统当中。

一些云服务商或专门的性能监控服务提供商也有各自的免费或收费的日志收集方案可以选择。

## 系统日志

系统日志可以通过常规方法在 Kubernetes 宿主机上收集，如果在你的现有基础设施中已经有用于收集日志的系统，只需要通过常规方法将相关服务器和日志文件添加到收集范围即可；如果没有可用的日志收集系统，或者希望部署一套独立的系统用于收集相关日志，也可以使用你熟悉的任意日志收集系统或方案。

上文提到的几种常见日志收集工具均支持对系统日志的收集，一些云服务商或专门的性能监控服务提供商也有各自的免费或收费的日志收集方案可以选择。
