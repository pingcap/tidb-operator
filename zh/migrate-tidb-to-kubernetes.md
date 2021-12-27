---
title: 将 TiDB 迁移至 Kubernetes
summary: 介绍如何将部署在物理机或虚拟机中的 TiDB 迁移至 Kubernetes 集群中
---

# 将 TiDB 迁移至 Kubernetes

本文介绍一种不借助备份恢复工具将部署在物理机或虚拟机中的 TiDB 迁移至 Kubernetes 中的方法。

## 先置条件

- Kubernetes 集群外物理机或虚拟机节点必须与集群内 Pod 网络互通
- Kubernetes 集群外物理机或虚拟机节点必须能够解析 Kubernetes 集群内部 Pod 域名（具体配置方法见第一步）
- 待迁移集群没有开启[组件间 TLS 加密通信](enable-tls-between-components.md)

## 第一步：在待迁移集群的所有节点中配置 DNS 服务

1. 获取 Kubernetes 集群 CoreDNS 或 kube-dns 服务的 endpoints 的 Pod ip 地址列表：

    {{< copyable "shell-regular" >}}

    ```bash
    kubectl describe svc/kube-dns -n kube-system
    ```
   
2. 修改待迁移集群节点 `/etc/resolv.conf` 配置，向该配置文件中添加如下内容：

    {{< copyable "shell-regular" >}}

    ```bash
    search default.svc.cluster.local svc.cluster.local cluster.local
    nameserver <CoreDNS Pod_IP_1>
    nameserver <CoreDNS Pod_IP_2>
    nameserver <CoreDNS Pod_IP_n>
    ```

3. 测试解析 Kubernetes 集群内部域名是否成功：

    ```bash
    $ ping basic-pd-2.basic-pd-peer.blade.svc
    PING basic-pd-2.basic-pd-peer.blade.svc (10.24.66.178) 56(84) bytes of data.
    64 bytes from basic-pd-2.basic-pd-peer.blade.svc (10.24.66.178): icmp_seq=1 ttl=61 time=0.213 ms
    64 bytes from basic-pd-2.basic-pd-peer.blade.svc (10.24.66.178): icmp_seq=2 ttl=61 time=0.175 ms
    64 bytes from basic-pd-2.basic-pd-peer.blade.svc (10.24.66.178): icmp_seq=3 ttl=61 time=0.188 ms
    64 bytes from basic-pd-2.basic-pd-peer.blade.svc (10.24.66.178): icmp_seq=4 ttl=61 time=0.157 ms
    ```

## 第二步：在 Kubernetes 中创建 TiDB 集群

1. 通过 [PD Control](https://docs.pingcap.com/zh/tidb/stable/pd-control) 获取待迁移集群 PD 节点地址及端口号：

    {{< copyable "shell-regular" >}}

    ```bash
    pd-ctl -u http://<address>:<port> member | jq '.members | .[] | .client_urls'
    ```

2. 在 Kubernetes 中创建目标 TiDB 集群（TiKV 节点个数不少于 3 个），并在 `spec.pdAddresses` 字段中指定待迁移 TiDB 集群的 PD 节点地址（以 `http://` 开头）：

    ```yaml
    spec
      ...
      pdAddresses:
      - http://pd1_addr:port
      - http://pd2_addr:port
      - http://pd3_addr:port
    ```

3. 确认部署在 Kubernetes 内的 TiDB 集群与待迁移 TiDB 集群组成的新集群正常运行。

    - 获取新集群 store 个数、状态：

        {{< copyable "shell-regular" >}}

        ```bash
        # store 个数
        pd-ctl -u http://<address>:<port> store | jq '.count'
        # store 状态
        pd-ctl -u http://<address>:<port> store | jq '.stores | .[] | .store.state_name'
        ```

    - 通过 MySQL 客户端[访问 Kubernetes 上的 TiDB 集群](access-tidb.md)。

## 第三步：缩容待迁移集群 TiDB 节点

将待迁移集群的 TiDB 节点缩容至 0 个：

- 如果待迁移集群使用 TiUP 部署，参考[缩容 TiDB/PD/TiKV 节点](https://docs.pingcap.com/zh/tidb/stable/scale-tidb-using-tiup#缩容-tidbpdtikv-节点)一节。
- 如果待迁移集群使用 TiDB Ansible 部署，参考[缩容 TiDB 节点](https://docs.pingcap.com/zh/tidb/stable/scale-tidb-using-ansible#缩容-tidb-节点)一节。

> **注意：**
>
> 若通过负载均衡或数据库访问层中间件的方式接入待迁移 TiDB 集群，则先修改配置，将业务流量迁移至目标 TiDB 集群，避免影响业务。

## 第四步：缩容待迁移集群 TiKV 节点

将待迁移集群的 TiKV 节点缩容至 0 个：

- 如果待迁移集群使用 TiUP 部署，参考[缩容 TiDB/PD/TiKV 节点](https://docs.pingcap.com/zh/tidb/stable/scale-tidb-using-tiup#缩容-tidbpdtikv-节点)一节。
- 如果待迁移集群使用 TiDB Ansible 部署，参考[缩容 TiKV 节点](https://docs.pingcap.com/zh/tidb/stable/scale-tidb-using-ansible#缩容-tikv-节点)一节。

> **注意：**
>
> * 依次缩容待迁移集群的 TiKV 节点，等待上一个 TiKV 节点对应的 store 状态变为 "tombstone" 后，再执行下一个 TiKV 节点的缩容操作。
> * 可通过 PD Control 工具查看 store 状态。

## 第五步：缩容待迁移集群 PD 节点

将待迁移集群的 PD 节点缩容至 0 个：

- 如果待迁移集群使用 TiUP 部署，参考[缩容 TiDB/PD/TiKV 节点](https://docs.pingcap.com/zh/tidb/stable/scale-tidb-using-tiup#缩容-tidbpdtikv-节点)一节。
- 如果待迁移集群使用 TiDB Ansible 部署，参考[缩容 PD 节点](https://docs.pingcap.com/zh/tidb/stable/scale-tidb-using-ansible#缩容-pd-节点)一节。

## 第六步：删除 `spec.pdAddresses` 字段

为避免后续对集群进行操作时产生困惑，迁移成功后，建议将新集群的 manifest 中的 `spec.pdAddresses` 字段删除。
