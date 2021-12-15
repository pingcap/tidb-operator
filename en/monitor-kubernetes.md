---
title: Monitoring and Alerts on Kubernetes
summary: Learn the monitoring and alerts on Kubernetes.
---

# Monitoring and Alerts on Kubernetes

This document describes how to monitor a Kubernetes cluster and configure alerts for the cluster.

## Monitor the Kubernetes cluster

The TiDB monitoring system deployed with the cluster only focuses on the operation of the TiDB components themselves, and does not include the monitoring of container resources, hosts, Kubernetes components, or TiDB Operator. To monitor these components or resources, you need to deploy a monitoring system across the entire Kubernetes cluster.

### Monitor the host

Monitoring the host and its resources works in the same way as monitoring physical resources of a traditional server.

If you already have a monitoring system for your physical server in your existing infrastructure, you only need to add the host that holds Kubernetes to the existing monitoring system by conventional means; if there is no monitoring system available, or if you want to deploy a separate monitoring system to monitor the host that holds Kubernetes, then you can use any monitoring system that you are familiar with.

The newly deployed monitoring system can run on a separate server, directly on the host that holds Kubernetes, or in a Kubernetes cluster. Different deployment methods might have differences in the deployment configuration and resource utilization, but there are no major differences in usage.

Some common open source monitoring systems that can be used to monitor server resources are:

- [CollectD](https://collectd.org/)
- [Nagios](https://www.nagios.org/)
- [Prometheus](https://prometheus.io/) & [node_exporter](https://github.com/prometheus/node_exporter)
- [Zabbix](https://www.zabbix.com/)

Some cloud service providers or specialized performance monitoring service providers also have their own free or chargeable monitoring solutions that you can choose from.

It is recommended to deploy a host monitoring system in the Kubernetes cluster via [Prometheus Operator](https://github.com/coreos/prometheus-operator) based on [Node Exporter](https://github.com/prometheus/node_exporter) and Prometheus. This solution can also be compatible with and used for monitoring the Kubernetes' own components.

### Monitor Kubernetes components

For monitoring Kubernetes components, you can refer to the solutions provided in the [Kubernetes official documentation](https://kubernetes.io/docs/tasks/debug-application-cluster/resource-usage-monitoring/) or use other Kubernetes-compatible monitoring systems.

Some cloud service providers may provide their own solutions for monitoring Kubernetes components. Some specialized performance monitoring service providers have their own Kubernetes integration solutions that you can choose from.

TiDB Operator is actually a container running in Kubernetes. For this reason, you can monitor TiDB Operator by choosing any monitoring system that can monitor the status and resources of a Kubernetes container without deploying additional monitoring components.

It is recommended to deploy a host monitoring system via [Prometheus Operator](https://github.com/coreos/prometheus-operator) based on [Node Exporter](https://github.com/prometheus/node_exporter) and Prometheus. This solution can also be compatible with and used for monitoring host resources.

### Alerts in Kubernetes

If you deploy a monitoring system for Kubernetes hosts and services using Prometheus Operator, some alert rules are configured by default, and an AlertManager service is deployed. For details, see [kube-prometheus](https://github.com/coreos/kube-prometheus).

If you monitor Kubernetes hosts and services by using other tools or services, you can consult the corresponding information provided by the tool or service provider.
