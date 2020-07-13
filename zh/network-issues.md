---
title: Kubernetes 上的 TiDB 集群常见网络问题
summary: 介绍 Kubernetes 上 TiDB 集群的常见网络问题以及诊断解决方案。
---

# Kubernetes 上的 TiDB 集群常见网络问题

本文介绍了 Kubernetes 上 TiDB 集群常见网络问题以及诊断解决方案。

## Pod 之间网络不通

针对 TiDB 集群而言，绝大部分 Pod 间的访问均通过 Pod 的域名（使用 Headless Service 分配）进行，例外的情况是 TiDB Operator 在收集集群信息或下发控制指令时，会通过 PD Service 的 `service-name` 访问 PD 集群。

当通过日志或监控确认 Pod 间存在网络连通性问题，或根据故障情况推断出 Pod 间网络连接可能不正常时，可以按照下面的流程进行诊断，逐步缩小问题范围：

1. 确认 Service 和 Headless Service 的 Endpoints 是否正常：

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl -n ${namespace} get endpoints ${cluster_name}-pd
    kubectl -n ${namespace} get endpoints ${cluster_name}-tidb
    kubectl -n ${namespace} get endpoints ${cluster_name}-pd-peer
    kubectl -n ${namespace} get endpoints ${cluster_name}-tikv-peer
    kubectl -n ${namespace} get endpoints ${cluster_name}-tidb-peer
    ```

    以上命令展示的 `ENDPOINTS` 字段中，应当是由逗号分隔的 `cluster_ip:port` 列表。假如字段为空或不正确，请检查 Pod 的健康状态以及 `kube-controller-manager` 是否正常工作。

2. 进入 Pod 的 Network Namespace 诊断网络问题：

    {{< copyable "shell-regular" >}}

    ```
    tkctl debug -n ${namespace} ${pod_name}
    ```

    远端 shell 启动后，使用 `dig` 命令诊断 DNS 解析，假如 DNS 解析异常，请参照[诊断 Kubernetes DNS 解析](https://kubernetes.io/docs/tasks/administer-cluster/dns-debugging-resolution/)进行故障排除：

    {{< copyable "shell-regular" >}}

    ```shell
    dig ${HOSTNAME}
    ```

    使用 `ping` 命令诊断到目的 IP 的三层网络是否连通（目的 IP 为使用 `dig` 解析出的 Pod IP）:

    {{< copyable "shell-regular" >}}

    ```shell
    ping ${TARGET_IP}
    ```

    假如 ping 检查失败，请参照[诊断 Kubernetes 网络](https://www.praqma.com/stories/debugging-kubernetes-networking/)进行故障排除。

    假如 ping 检查正常，继续使用 `telnet` 检查目标端口是否打开：

    {{< copyable "shell-regular" >}}

    ```shell
    telnet ${TARGET_IP} ${TARGET_PORT}
    ```

    假如 `telnet` 检查失败，则需要验证 Pod 的对应端口是否正确暴露以及应用的端口是否配置正确：

    {{< copyable "shell-regular" >}}

    ```shell
    # 检查端口是否一致
    kubectl -n ${namespace} get po ${pod_name} -ojson | jq '.spec.containers[].ports[].containerPort'

    # 检查应用是否被正确配置服务于指定端口上
    # PD, 未配置时默认为 2379 端口
    kubectl -n ${namespace} -it exec ${pod_name} -- cat /etc/pd/pd.toml | grep client-urls
    # TiKV, 未配置时默认为 20160 端口
    kubectl -n ${namespace} -it exec ${pod_name} -- cat /etc/tikv/tikv.toml | grep addr
    # TiDB, 未配置时默认为 4000 端口
    kubectl -n ${namespace} -it exec ${pod_name} -- cat /etc/tidb/tidb.toml | grep port
    ```

## 无法访问 TiDB 服务

TiDB 服务访问不了时，首先确认 TiDB 服务是否部署成功，确认方法如下：

查看该集群的所有组件是否全部都启动了，状态是否为 Running。

{{< copyable "shell-regular" >}}

```shell
kubectl get po -n ${namespace}
```

查看 TiDB 服务是否正确生成了 Endpoint 对象。

{{< copyable "shell-regular" >}}

```shell
kubectl get endpoints -n ${namespaces} ${cluster_name}-tidb
```

检查 TiDB 组件的日志，看日志是否有报错。

{{< copyable "shell-regular" >}}

```shell
kubectl logs -f ${pod_name} -n ${namespace} -c tidb
```

如果确定集群部署成功，则进行网络检查：

1. 如果你是通过 `NodePort` 方式访问不了 TiDB 服务，请在 node 上尝试使用 clusterIP 访问 TiDB 服务，假如 clusterIP 的方式能访问，基本判断 Kubernetes 集群内的网络是正常的，问题可能出在下面两个方面：

    * 客户端到 node 节点的网络不通。
    * 查看 TiDB service 的 `externalTrafficPolicy` 属性是否为 Local。如果是 Local 则客户端必须通过 TiDB Pod 所在 node 的 IP 来访问。

2. 如果 clusterIP 方式也访问不了 TiDB 服务，尝试用 TiDB服务后端的 `<PodIP>:4000` 连接看是否可以访问，如果通过 PodIP 可以访问 TiDB 服务，可以确认问题出在 clusterIP 到 PodIP 之间的连接上，排查项如下：

    * 检查各个 node 上的 kube-proxy 是否正常运行：

        {{< copyable "shell-regular" >}}

        ```shell
        kubectl get po -n kube-system -l k8s-app=kube-proxy
        ```

    * 检查 node 上的 iptables 规则中 TiDB 服务的规则是否正确：

        {{< copyable "shell-regular" >}}

        ```shell
        iptables-save -t nat |grep ${clusterIP}
        ```

    * 检查对应的 endpoint 是否正确：

        {{< copyable "shell-regular" >}}

        ```shell
        kubectl get endpoints -n ${namespaces} ${cluster_name}-tidb
        ```

3. 如果通过 PodIP 访问不了 TiDB 服务，问题出在 Pod 层面的网络上，排查项如下：

    * 检查 node 上的相关 route 规则是否正确
    * 检查网络插件服务是否正常
    * 参考上面的 [Pod 之间网络不通](#pod-之间网络不通)章节
