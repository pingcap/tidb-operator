---
title: 访问 Kubernetes 上的 TiDB 集群
summary: 介绍如何访问 Kubernetes 上的 TiDB 集群。
aliases: ['/docs-cn/tidb-in-kubernetes/dev/access-tidb/']
---

# 访问 TiDB 集群

Service 可以根据场景配置不同的类型，比如 `ClusterIP`、`NodePort`、`LoadBalancer` 等，对于不同的类型可以有不同的访问方式。

可以通过如下命令获取 TiDB Service 信息：

{{< copyable "shell-regular" >}}

```bash
kubectl get svc ${serviceName} -n ${namespace}
```

示例：

```
# kubectl get svc basic-tidb -n default
NAME         TYPE       CLUSTER-IP     EXTERNAL-IP   PORT(S)                          AGE
basic-tidb   NodePort   10.233.6.240   <none>        4000:32498/TCP,10080:30171/TCP   61d
```

上述示例描述了 `default` namespace 下 `basic-tidb` 服务的信息，类型为 `NodePort`，ClusterIP 为 `10.233.6.240`，ServicePort 为 `4000` 和 `10080`，对应的 NodePort 分别为 `32498` 和 `30171`。

> **注意：**
>
> [MySQL 8.0 默认认证插件](https://dev.mysql.com/doc/refman/8.0/en/server-system-variables.html#sysvar_default_authentication_plugin)从 `mysql_native_password` 更新为 `caching_sha2_password`，因此如果使用 MySQL 8.0 客户端访问 TiDB 服务（TiDB 版本 < v4.0.7），并且用户账户有配置密码，需要显示指定 `--default-auth=mysql_native_password` 参数。

## ClusterIP

`ClusterIP` 是通过集群的内部 IP 暴露服务，选择该类型的服务时，只能在集群内部访问，可以通过如下方式访问：

* ClusterIP + ServicePort
* Service 域名 (`${serviceName}.${namespace}`) + ServicePort

## NodePort

在没有 LoadBalancer 时，可选择通过 NodePort 暴露。NodePort 是通过节点的 IP 和静态端口暴露服务。通过请求 `NodeIP + NodePort`，可以从集群的外部访问一个 NodePort 服务。

查看 Service 分配的 Node Port，可通过获取 TiDB 的 Service 对象来获知：

{{< copyable "shell-regular" >}}

```bash
kubectl -n ${namespace} get svc ${cluster_name}-tidb -ojsonpath="{.spec.ports[?(@.name=='mysql-client')].nodePort}{'\n'}"
```

查看可通过哪些节点的 IP 访问 TiDB 服务，有两种情况：

- `externalTrafficPolicy` 为 `Cluster` 时，所有节点 IP 均可
- `externalTrafficPolicy` 为 `Local` 时，可通过以下命令获取指定集群的 TiDB 实例所在的节点

    {{< copyable "shell-regular" >}}

    ```bash
    kubectl -n ${namespace} get pods -l "app.kubernetes.io/component=tidb,app.kubernetes.io/instance=${cluster_name}" -ojsonpath="{range .items[*]}{.spec.nodeName}{'\n'}{end}"
    ```

## LoadBalancer

若运行在有 LoadBalancer 的环境，比如 GCP/AWS 平台，建议使用云平台的 LoadBalancer 特性。

参考 [EKS](deploy-on-aws-eks.md#安装-mysql-客户端并连接)、[GKE](deploy-on-gcp-gke.md#安装-mysql-客户端并连接) 和 [ACK](deploy-on-alibaba-cloud.md#连接数据库) 文档，通过 LoadBalancer 访问 TiDB 服务。

访问 [Kubernetes Service 文档](https://kubernetes.io/docs/concepts/services-networking/service/)，了解更多 Service 特性以及云平台 Load Balancer 支持。
