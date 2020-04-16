---
title: Restart a TiDB Cluster in Kubernetes
summary: Learn how to restart a TiDB cluster in the Kubernetes cluster.
category: how-to
---

# Restart a TiDB Cluster in Kubernetes

If you find that the memory leak occurs in a Pod of the TiDB cluster during use, you need to restart the cluster. This document describes how to logoff a Pod and restart a TiDB cluster gracefully.

> **Warning:**
>
> It is not recommended to manually remove a Pod in the TiDB cluster without graceful restart in a production environment, because this might lead to some request failures of accessing the TiDB cluster though the `StatefulSet` controller pulls the Pod up again.

## Enable the configurations

To activate the graceful logoff feature, you need to enable some related configurations in TiDB Operator. These configurations are disabled by default. Take the following steps to manually turn them on.

1. Edit the `values.yaml` file.

    Enable the `Operator Webhook` feature:

    ```yaml
    admissionWebhook:
      create: true
    ```

    For more information about `Operator Webhook`, see [Enable Admission Controller in TiDB Operator](enable-admission-webhook.md).

2. Install or update TiDB Operator.

    To install or update TiDB Operator, see [Deploy TiDB Operator in Kubernetes](deploy-tidb-operator.md).

## Use annotate to mark the target Pod

You can use `kubectl annotate` to mark the target Pod component of the TiDB cluster. After marking, the TiDB Operator automatically performs graceful logoff of the Pod and restarts the target Pod. To mark the target Pod, run the following command:

{{< copyable "shell-regular" >}}

```sh
kubectl annotate ${pod_name} -n ${namespace} tidb.pingcap.com/pod-defer-deleting=true
```
