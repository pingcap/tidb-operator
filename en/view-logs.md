---
title: View TiDB Logs in Kubernetes
summary: Learn how to view TiDB slow logs and application logs in Kubernetes.
aliases: ['/docs/tidb-in-kubernetes/dev/view-logs/']
---

# View TiDB Logs in Kubernetes

This document introduces the methods to view logs of TiDB components and TiDB slow log.

## View logs of TiDB components

The TiDB components deployed by TiDB Operator output the logs in the `stdout` and `stderr` of the container by default. You can view the log of a single Pod by running the following command:

{{< copyable "shell-regular" >}}

```bash
kubectl logs -n ${namespace} ${pod_name}
```

If the Pod has multiple containers, you can also view the logs of a container in this Pod:

{{< copyable "shell-regular" >}}

```bash
kubectl logs -n ${namespace} ${pod_name} -c ${container_name}
```

For more methods to view Pod logs, run `kubectl logs --help`.

## View slow query logs of TiDB components

For TiDB 3.0 or later versions, TiDB separates slow query logs from application logs. You can view slow query logs from the sidecar container named `slowlog`:

{{< copyable "shell-regular" >}}

```bash
kubectl logs -n ${namespace} ${pod_name} -c slowlog
```

> **Note:**
>
> The format of TiDB slow query logs is the same as that of MySQL slow query logs. However, due to the characteristics of TiDB itself, some of the specific fields might be different. For this reason, the tool for parsing MySQL slow query logs may not be fully compatible with TiDB slow query logs.
