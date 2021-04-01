---
title: Kubernetes 上的集群初始化配置
summary: 介绍如何初始化配置 Kubernetes 上的 TiDB 集群。
aliases: ['/docs-cn/tidb-in-kubernetes/dev/initialize-a-cluster/']
---

# Kubernetes 上的集群初始化配置

本文介绍如何对 Kubernetes 上的集群进行初始化配置完成初始化账号和密码设置，以及批量自动执行 SQL 语句对数据库进行初始化。

> **注意：**
>
> * 如果 TiDB 集群创建完以后手动修改过 `root` 用户的密码，初始化会失败。
> * 以下功能只在 TiDB 集群创建后第一次执行起作用，执行完以后再修改不会生效。

## 配置 TidbInitializer

请参考 TidbInitializer [示例](https://github.com/pingcap/tidb-operator/blob/master/manifests/initializer/tidb-initializer.yaml)和 [API 文档](https://github.com/pingcap/tidb-operator/blob/master/docs/api-references/docs.md)（示例和 API 文档请切换到当前使用的 TiDB Operator 版本）以及下面的步骤，完成 TidbInitializer CR，保存到文件 `${cluster_name}/tidb-initializer.yaml`。

### 初始化账号和密码设置

集群创建时默认会创建 `root` 账号，但是密码为空，这会带来一些安全性问题。可以通过如下步骤为 `root` 账号设置初始密码：

通过下面命令创建 [Secret](https://kubernetes.io/docs/concepts/configuration/secret/) 指定 root 账号密码：

{{< copyable "shell-regular" >}}

```shell
kubectl create secret generic tidb-secret --from-literal=root=${root_password} --namespace=${namespace}
```

如果希望能自动创建其它用户，可以在上面命令里面再加上其他用户的 username 和 password，例如：

{{< copyable "shell-regular" >}}

```shell
kubectl create secret generic tidb-secret --from-literal=root=${root_password} --from-literal=developer=${developer_password} --namespace=${namespace}
```

该命令会创建 `root` 和 `developer` 两个用户的密码，存到 `tidb-secret` 的 Secret 里面。并且创建的普通用户 `developer` 默认只有 `USAGE` 权限，其他权限请在 `initSql` 中设置。

在 `${cluster_name}/tidb-initializer.yaml` 中设置 `passwordSecret: tidb-secret`。

## 设置允许访问 TiDB 的主机

在 `${cluster_name}/tidb-initializer.yaml` 中设置 `permitHost: ${mysql_client_host_name}` 配置项来设置允许访问 TiDB 的主机 **host_name**。如果不设置，则允许所有主机访问。详情请参考 [MySQL GRANT host name](https://dev.mysql.com/doc/refman/5.7/en/grant.html)。

## 批量执行初始化 SQL 语句

集群在初始化过程还可以自动执行 `initSql` 中的 SQL 语句用于初始化，该功能可以用于默认给集群创建一些 database 或者 table，并且执行一些用户权限管理类的操作。例如如下设置会在集群创建完成后自动创建名为 `app` 的 database，并且赋予 `developer` 账号对 `app` 的所有管理权限：

{{< copyable "" >}}

```yaml
spec:
...
initSql: |-
    CREATE DATABASE app;
    GRANT ALL PRIVILEGES ON app.* TO 'developer'@'%';
```

> **注意：**
>
> 目前没有对 initSql 做校验，尽管也可以在 initSql 里面创建账户和设置密码，但这种方式会将密码以明文形式存到 initializer Job 对象上，不建议这么做。

## 执行初始化

{{< copyable "shell-regular" >}}

```shell
kubectl apply -f ${cluster_name}/tidb-initializer.yaml --namespace=${namespace}
```

以上命令会自动创建一个初始化的 Job，该 Job 会尝试利用提供的 secret 给 root 账号创建初始密码，并且创建其它账号和密码（如果指定了的话）。初始化完成后 Pod 状态会变成 Completed，之后通过 MySQL 客户端登录时需要指定这里设置的密码。

如果服务器没有外网，需要在有外网的机器上将集群初始化用到的 Docker 镜像下载下来并上传到服务器上，然后使用 `docker load` 将 Docker 镜像安装到服务器上。

初始化一套 TiDB 集群会用到下面这些 Docker 镜像：

```shell
tnir/mysqlclient:latest
```

接下来通过下面的命令将所有这些镜像下载下来：

{{< copyable "shell-regular" >}}

```shell
docker pull tnir/mysqlclient:latest
docker save -o mysqlclient-latest.tar tnir/mysqlclient:latest
```

接下来将这些 Docker 镜像上传到服务器上，并执行 `docker load` 将这些 Docker 镜像安装到服务器上：

{{< copyable "shell-regular" >}}

```shell
docker load -i mysqlclient-latest.tar
```
