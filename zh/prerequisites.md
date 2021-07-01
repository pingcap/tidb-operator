---
title: Kubernetes 上的 TiDB 集群环境需求
summary: 介绍在 Kubernetes 上部署 TiDB 集群的软硬件环境需求。
aliases: ['/docs-cn/tidb-in-kubernetes/dev/prerequisites/']
---

# Kubernetes 上的 TiDB 集群环境需求

本文介绍在 Kubernetes 上部署 TiDB 集群的软硬件环境需求。

## 软件版本要求

| 软件名称 | 版本 |
| :--- | :--- |
| Docker | Docker CE 18.09.6 |
| Kubernetes | v1.12.5+ |
| CentOS | CentOS 7.6，内核要求为 3.10.0-957 或之后版本 |
| Helm | v3.0.0+ |

## 配置防火墙

建议关闭防火墙：

{{< copyable "shell-regular" >}}

```shell
systemctl stop firewalld
systemctl disable firewalld
```

如果无法关闭 firewalld 服务，为了保证 Kubernetes 正常运行，需要打开以下端口：

1. 在 Master 节点上，打开以下端口，然后重新启动服务：

    {{< copyable "shell-regular" >}}

    ```shell
    firewall-cmd --permanent --add-port=6443/tcp
    firewall-cmd --permanent --add-port=2379-2380/tcp
    firewall-cmd --permanent --add-port=10250/tcp
    firewall-cmd --permanent --add-port=10251/tcp
    firewall-cmd --permanent --add-port=10252/tcp
    firewall-cmd --permanent --add-port=10255/tcp
    firewall-cmd --permanent --add-port=8472/udp
    firewall-cmd --add-masquerade --permanent

    # 当需要在 Master 节点上暴露 NodePort 时候设置
    firewall-cmd --permanent --add-port=30000-32767/tcp

    systemctl restart firewalld
    ```

2. 在 Node 节点上，打开以下端口，然后重新启动服务：

    {{< copyable "shell-regular" >}}

    ```shell
    firewall-cmd --permanent --add-port=10250/tcp
    firewall-cmd --permanent --add-port=10255/tcp
    firewall-cmd --permanent --add-port=8472/udp
    firewall-cmd --permanent --add-port=30000-32767/tcp
    firewall-cmd --add-masquerade --permanent

    systemctl restart firewalld
    ```

## 配置 Iptables

FORWARD 链默认配置成 ACCEPT，并将其设置到开机启动脚本里：

{{< copyable "shell-regular" >}}

```shell
iptables -P FORWARD ACCEPT
```

## 禁用 SELinux

{{< copyable "shell-regular" >}}

```shell
setenforce 0
sed -i 's/^SELINUX=enforcing$/SELINUX=permissive/' /etc/selinux/config
```

## 关闭 Swap

Kubelet 正常工作需要关闭 Swap，并且把 `/etc/fstab` 里面有关 Swap 的那行注释掉：

{{< copyable "shell-regular" >}}

```shell
swapoff -a
sed -i 's/^\(.*swap.*\)$/#\1/' /etc/fstab 
```

## 内核参数设置

按照下面的配置设置内核参数，也可根据自身环境进行微调：

{{< copyable "shell-regular" >}}

```shell
modprobe br_netfilter

cat <<EOF >  /etc/sysctl.d/k8s.conf
net.bridge.bridge-nf-call-ip6tables = 1
net.bridge.bridge-nf-call-iptables = 1
net.bridge.bridge-nf-call-arptables = 1
net.core.somaxconn = 32768
vm.swappiness = 0
net.ipv4.tcp_syncookies = 0
net.ipv4.ip_forward = 1
fs.file-max = 1000000
fs.inotify.max_user_watches = 1048576
fs.inotify.max_user_instances = 1024
net.ipv4.conf.all.rp_filter = 1
net.ipv4.neigh.default.gc_thresh1 = 80000
net.ipv4.neigh.default.gc_thresh2 = 90000
net.ipv4.neigh.default.gc_thresh3 = 100000
EOF

sysctl --system
```

## 配置 Irqbalance 服务

[Irqbalance](https://access.redhat.com/documentation/en-us/red_hat_enterprise_linux/6/html/performance_tuning_guide/sect-red_hat_enterprise_linux-performance_tuning_guide-tool_reference-irqbalance) 服务可以将各个设备对应的中断号分别绑定到不同的 CPU 上，以防止所有中断请求都落在同一个 CPU 上而引发性能瓶颈。

{{< copyable "shell-regular" >}}

```shell
systemctl enable irqbalance
systemctl start irqbalance
```

## CPUfreq 调节器模式设置

为了让 CPU 发挥最大性能，请将 CPUfreq 调节器模式设置为 performance 模式。详细参考[在部署目标机器上配置 CPUfreq 调节器模式](https://pingcap.com/docs-cn/stable/online-deployment-using-ansible/#查看系统支持的调节器模式)。

{{< copyable "shell-regular" >}}

```shell
cpupower frequency-set --governor performance
```

## Ulimit 设置

TiDB 集群默认会使用很多文件描述符，需要将工作节点上面的 `ulimit` 设置为大于等于 `1048576`：

{{< copyable "shell-regular" >}}

```shell
cat <<EOF >>  /etc/security/limits.conf
root        soft        nofile        1048576
root        hard        nofile        1048576
root        soft        stack         10240
EOF

sysctl --system
```

## Docker 服务

安装 Docker 时，建议选择 Docker CE 18.09.6 及以上版本。请参考 [Docker 安装指南](https://docs.docker.com/engine/install/centos/) 进行安装。

安装完 Docker 服务以后，执行以下步骤：

1. 将 Docker 的数据保存到一块单独的盘上，Docker 的数据主要包括镜像和容器日志数据。通过设置 [`--data-root`](https://docs.docker.com/config/daemon/systemd/#runtime-directory-and-storage-driver) 参数来实现：

    {{< copyable "shell-regular" >}}

    ```shell
    cat > /etc/docker/daemon.json <<EOF
    {
      "exec-opts": ["native.cgroupdriver=systemd"],
      "log-driver": "json-file",
      "log-opts": {
        "max-size": "100m"
      },
      "storage-driver": "overlay2",
      "storage-opts": [
        "overlay2.override_kernel_check=true"
      ],
      "data-root": "/data1/docker"
    }
    EOF
    ```

    上面会将 Docker 的数据目录设置为 `/data1/docker`。

2. 设置 Docker daemon 的 ulimit。

    1. 创建 docker service 的 systemd drop-in 目录 `/etc/systemd/system/docker.service.d`：

        {{< copyable "shell-regular" >}}

        ```shell
        mkdir -p /etc/systemd/system/docker.service.d
        ```

    2. 创建 `/etc/systemd/system/docker.service.d/limit-nofile.conf` 文件，并配置 `LimitNOFILE` 参数的值，取值范围为大于等于 `1048576` 的数字即可。

        {{< copyable "shell-regular" >}}

        ```shell
        cat > /etc/systemd/system/docker.service.d/limit-nofile.conf <<EOF
        [Service]
        LimitNOFILE=1048576
        EOF
        ```

        > **注意：**
        >
        > 请勿将 `LimitNOFILE` 的值设置为 `infinity`。由于 [`systemd` 的 bug](https://github.com/systemd/systemd/commit/6385cb31ef443be3e0d6da5ea62a267a49174688#diff-108b33cf1bd0765d116dd401376ca356L1186)，`infinity` 在 `systemd` 某些版本中指的是 `65536`。

    3. 重新加载配置。

        {{< copyable "shell-regular" >}}

        ```shell
        systemctl daemon-reload && systemctl restart docker
        ```

## Kubernetes 服务

参考 [Kubernetes 官方文档](https://kubernetes.io/docs/setup/production-environment/tools/kubeadm/high-availability/)，部署一套多 Master 节点高可用集群。

Kubernetes Master 节点的配置取决于 Kubernetes 集群中 Node 节点个数，节点数越多，需要的资源也就越多。节点数可根据需要做微调。

| Kubernetes 集群 Node 节点个数 | Kubernetes Master 节点配置 |
| :--- | :--- |
| 1-5 | 1vCPUs 4GB Memory|
| 6-10 | 2vCPUs 8GB Memory|
| 11-100 | 4vCPUs 16GB Memory|
| 101-250 | 8vCPUs 32GB Memory|
| 251-500 | 16vCPUs 64GB Memory|
| 501-5000 | 32vCPUs 128GB Memory|

安装完 Kubelet 之后，执行以下步骤：

1. 将 Kubelet 的数据保存到一块单独盘上（可跟 Docker 共用一块盘），Kubelet 主要占盘的数据是 [emptyDir](https://kubernetes.io/docs/concepts/storage/volumes/#emptydir) 所使用的数据。通过设置 `--root-dir` 参数来实现：

    {{< copyable "shell-regular" >}}
    
    ```shell
    echo "KUBELET_EXTRA_ARGS=--root-dir=/data1/kubelet" > /etc/sysconfig/kubelet
    systemctl restart kubelet
    ```
   
    上面会将 Kubelet 数据目录设置为 `/data1/kubelet`。
    
2. 通过 kubelet 设置[预留资源](https://kubernetes.io/docs/tasks/administer-cluster/reserve-compute-resources/)，保证机器上的系统进程以及 Kubernetes 的核心进程在工作负载很高的情况下仍然有足够的资源来运行，从而保证整个系统的稳定。

## TiDB 集群资源需求

请根据[服务器建议配置](https://pingcap.com/docs-cn/stable/hardware-and-software-requirements/#生产环境)来规划机器的配置。

另外，在生产环境中，尽量不要在 Kubernetes Master 节点部署 TiDB 实例，或者尽可能少地部署 TiDB 实例。因为网卡带宽的限制，Master 节点网卡满负荷工作会影响到 Worker 节点和 Master 节点之间的心跳信息汇报，导致比较严重的问题。
