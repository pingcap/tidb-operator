---
title: 优雅重启 TidbCluster 组件
category: how-to
---

## 优雅重启 TidbCluster 组件

在使用 TiDB 集群的过程中，如果你发现某个集群节点存在内存泄漏等问题需要进行重启。我们可以通过优雅重启指令来将某个 TiDB 集群节点优雅下线然后再进行重新启动。

> **警告：**
>
> 在生产环境未经过优雅重启，手动删除某个 TiDB 集群 Pod 节点是一件极其危险的事情，虽然 Stateful 控制器会将 Pod 节点再次拉起，但这依旧会对 TiDB 集群造成伤害。
>

## 开启相关设置

开启优雅下线功能，需要打开 Operator 相关设置。默认情况下相关配置是关闭的，你需要手动开启:

1. 修改 Operator 的 `values.yaml`

    开启 Operator Webhook 特性:

    ```yaml
    admissionWebhook:
      create: true
      mutation:
        pods: true
    ```

2. 安装/更新 operator

    修改完 `values.yaml` 文件中上述配置项以后进行 TiDB-Operator 部署或者更新。安装与更新 Operator 请参考[在 Kubernetes 上部署 TiDB Operator](deploy-tidb-operator.md)。 


## 使用 annotate 标记目标节点

我们通过 `kubectl annotate` 的方式来标记目前 TiDB 节点组件，当 `annotate` 标记完成以后，operator 会自动进行节点的优雅下线并进行重启。你可以通过以下方式来进行标记:

    {{< copyable "shell-regular" >}}

    ```sh
    kubectl annotate <pod-name> -n <namespace> tidb.pingcap.com/pod-defer-deleting=true
    ```
