---
title: Prerequisites for TiDB in Kubernetes
summary: Learn the prerequisites for TiDB in Kubernetes.
aliases: ['/docs/tidb-in-kubernetes/dev/prerequisites/']
---

# Prerequisites for TiDB in Kubernetes

This document introduces the hardware and software prerequisites for deploying a TiDB cluster in Kubernetes.

## Software version

| Software Name | Version |
| :--- | :--- |
| Docker | Docker CE 18.09.6 |
| Kubernetes | v1.12.5+ |
| CentOS | 7.6 and kernel 3.10.0-957 or later |
| Helm | v3.0.0+ |

## Configure the firewall

It is recommended that you disable the firewall.

{{< copyable "shell-regular" >}}

```shell
systemctl stop firewalld
systemctl disable firewalld
```

If you cannot stop the firewalld service, to ensure the normal operation of Kubernetes, take the following steps:

1. Enable the following ports on the master, and then restart the service:

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

    # Set it when you need to expose NodePort on the master node.
    firewall-cmd --permanent --add-port=30000-32767/tcp
    systemctl restart firewalld
    ```

2. Enable the following ports on the nodes, and then restart the service:

    {{< copyable "shell-regular" >}}

    ```shell
    firewall-cmd --permanent --add-port=10250/tcp
    firewall-cmd --permanent --add-port=10255/tcp
    firewall-cmd --permanent --add-port=8472/udp
    firewall-cmd --permanent --add-port=30000-32767/tcp
    firewall-cmd --add-masquerade --permanent

    systemctl restart firewalld
    ```

## Configure Iptables

The FORWARD chain is configured to `ACCEPT` by default and is set in the startup script:

{{< copyable "shell-regular" >}}

```shell
iptables -P FORWARD ACCEPT
```

## Disable SELinux

{{< copyable "shell-regular" >}}

```shell
setenforce 0
sed -i 's/^SELINUX=enforcing$/SELINUX=permissive/' /etc/selinux/config
```

## Disable swap

To make kubelet work, you need to turn off swap and comment out the swap-related line in the `/etc/fstab` file.

{{< copyable "shell-regular" >}}

```shell
swapoff -a
sed -i 's/^\(.*swap.*\)$/#\1/' /etc/fstab
```

## Configure kernel parameters

Configure the kernel parameters as follows. You can also adjust them according to your environment:

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

## Configure the Irqbalance service

The [Irqbalance](https://access.redhat.com/documentation/en-us/red_hat_enterprise_linux/6/html/performance_tuning_guide/sect-red_hat_enterprise_linux-performance_tuning_guide-tool_reference-irqbalance) service binds the interrupts of each equipment to different CPUs respectively. This avoids the performance bottleneck when all interrupt requests are sent to the same CPU.

{{< copyable "shell-regular" >}}

```shell
systemctl enable irqbalance
systemctl start irqbalance
```

## Configure the CPUfreq governor mode

To make full use of CPU performance, set the CPUfreq governor mode to `performance`. For details, see [Configure the CPUfreq governor mode on the target machine](https://docs.pingcap.com/tidb/v4.0/online-deployment-using-ansible#step-7-configure-the-cpufreq-governor-mode-on-the-target-machine).

{{< copyable "shell-regular" >}}

```shell
cpupower frequency-set --governor performance
```

## Configure `ulimit`

The TiDB cluster uses many file descriptors by default. The `ulimit` of the worker node must be greater than or equal to `1048576`.

{{< copyable "shell-regular" >}}

```shell
cat <<EOF >>  /etc/security/limits.conf
root        soft        nofile        1048576
root        hard        nofile        1048576
root        soft        stack         10240
EOF
sysctl --system
```

## Docker service

It is recommended to install Docker CE 18.09.6 or later versions. See [Install Docker](https://docs.docker.com/engine/install/centos/) for details.

After the installation, take the following steps:

1. Save the Docker data to a separate disk. The data mainly contains images and the container logs. To implement this, set the [`--data-root`](https://docs.docker.com/config/daemon/systemd/#runtime-directory-and-storage-driver) parameter:

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

    The above command sets the data directory of Docker to `/data1/docker`.

2. Set `ulimit` for the Docker daemon:

    1. Create the systemd drop-in directory for the docker service:

        {{< copyable "shell-regular" >}}

        ```shell
        mkdir -p /etc/systemd/system/docker.service.d
        ```

    2. Create a file named as `/etc/systemd/system/docker.service.d/limit-nofile.conf`, and configure the value of the  `LimitNOFILE` parameter. The value must be a number equal to or greater than `1048576`.

        {{< copyable "shell-regular" >}}

        ```shell
        cat > /etc/systemd/system/docker.service.d/limit-nofile.conf <<EOF
        [Service]
        LimitNOFILE=1048576
        EOF
        ```

        > **Note:**
        >
        > DO NOT set the value of `LimitNOFILE` to `infinity`. Due to [a bug of `systemd`](https://github.com/systemd/systemd/commit/6385cb31ef443be3e0d6da5ea62a267a49174688#diff-108b33cf1bd0765d116dd401376ca356L1186), the `infinity` value of `systemd` in some versions is `65536`.

    3. Reload the configuration.

        {{< copyable "shell-regular" >}}

        ```shell
        systemctl daemon-reload && systemctl restart docker
        ```

## Kubernetes service

To deploy a multi-master, highly available cluster, see [Kubernetes documentation](https://kubernetes.io/docs/setup/production-environment/tools/kubeadm/high-availability/).

The configuration of the Kubernetes master depends on the number of nodes. More nodes consumes more resources. You can adjust the number of nodes as needed.

| Nodes in a Kubernetes cluster | Kubernetes master configuration |
| :--- | :--- |
| 1-5 | 1vCPUs 4GB Memory|
| 6-10 | 2vCPUs 8GB Memory|
| 11-100 | 4vCPUs 16GB Memory|
| 101-250 | 8vCPUs 32GB Memory|
| 251-500 | 16vCPUs 64GB Memory|
| 501-5000 | 32vCPUs 128GB Memory|

After Kubelet is installed, take the following steps:

1. Save the Kubelet data to a separate disk (it can share the same disk with Docker). The data mainly contains the data used by [emptyDir](https://kubernetes.io/docs/concepts/storage/volumes/#emptydir). To implement this, set the `--root-dir` parameter:

    {{< copyable "shell-regular" >}}

    ```shell
    echo "KUBELET_EXTRA_ARGS=--root-dir=/data1/kubelet" > /etc/sysconfig/kubelet
    systemctl restart kubelet
    ```

    The above command sets the data directory of Kubelet to `/data1/kubelet`.

2. [Reserve compute resources](https://kubernetes.io/docs/tasks/administer-cluster/reserve-compute-resources/) by using Kubelet, to ensure that the system process of the machine and the kernel process of Kubernetes have enough resources for operation in heavy workloads. This maintains the stability of the entire system.

## TiDB cluster's requirements for resources

To determine the machine configuration, see [Server recommendations](https://docs.pingcap.com/tidb/v4.0/hardware-and-software-requirements#production-environment).

In a production environment, avoid deploying TiDB instances on a kubernetes master, or deploy as few TiDB instances as possible. Due to the NIC bandwidth, if the NIC of the master node works at full capacity, the heartbeat report between the worker node and the master node will be affected and might lead to serious problems.
