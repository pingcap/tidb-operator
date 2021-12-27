---
title: 为 DM 开启 TLS
summary: 在 Kubernetes 上如何为 DM 开启 TLS。
---

# 为 DM 开启 TLS

本文主要描述了在 Kubernetes 上如何为 DM 集群组件间开启 TLS，以及如何用 DM 集群同步开启了 MySQL 客户端 TLS 验证的 MySQL/TiDB 数据库。

## 为 DM 组件间开启 TLS

TiDB Operator 从 v1.2 开始已经支持为 Kubernetes 上 DM 集群组件间开启 TLS。开启步骤为：

1. 为即将被创建的 DM 集群的每个组件生成证书：
    - 为 DM-master/DM-worker 组件分别创建一套 Server 端证书，保存为 Kubernetes Secret 对象：`${cluster_name}-${component_name}-cluster-secret`
    - 为它们的各种客户端创建一套共用的 Client 端证书，保存为 Kubernetes Secret 对象：`${cluster_name}-dm-client-secret`

    > **注意：**
    >
    > 创建的 Secret 对象必须符合上述命名规范，否则将导致 DM 集群部署失败。

2. 部署集群，设置 `.spec.tlsCluster.enabled` 属性为 `true`；
3. 配置 `dmctl` 连接集群。

其中，颁发证书的方式有多种，本文档提供两种方式，用户也可以根据需要为 DM 集群颁发证书，这两种方式分别为：

- [使用 `cfssl` 系统颁发证书](#使用-cfssl-系统颁发证书)；
- [使用 `cert-manager` 系统颁发证书](#使用-cert-manager-系统颁发证书)；

当需要更新已有 TLS 证书时，可参考[更新和替换 TLS 证书](renew-tls-certificate.md)。

### 第一步：为 DM 集群各个组件生成证书

#### 使用 `cfssl` 系统颁发证书

1. 首先下载 `cfssl` 软件并初始化证书颁发机构：

    {{< copyable "shell-regular" >}}

    ``` shell
    mkdir -p ~/bin
    curl -s -L -o ~/bin/cfssl https://pkg.cfssl.org/R1.2/cfssl_linux-amd64
    curl -s -L -o ~/bin/cfssljson https://pkg.cfssl.org/R1.2/cfssljson_linux-amd64
    chmod +x ~/bin/{cfssl,cfssljson}
    export PATH=$PATH:~/bin

    mkdir -p cfssl
    cd cfssl
    ```

2. 生成 `ca-config.json` 配置文件：

    ```bash
    cat << EOF > ca-config.json
    {
        "signing": {
            "default": {
                "expiry": "8760h"
            },
            "profiles": {
                "internal": {
                    "expiry": "8760h",
                    "usages": [
                        "signing",
                        "key encipherment",
                        "server auth",
                        "client auth"
                    ]
                },
                "client": {
                    "expiry": "8760h",
                    "usages": [
                        "signing",
                        "key encipherment",
                        "client auth"
                    ]
                }
            }
        }
    }
    EOF
    ```

3. 生成 `ca-csr.json` 配置文件：

    ```bash
    cat << EOF > ca-csr.json
    {
        "CN": "TiDB",
        "CA": {
            "expiry": "87600h"
        },
        "key": {
            "algo": "rsa",
            "size": 2048
        },
        "names": [
            {
                "C": "US",
                "L": "CA",
                "O": "PingCAP",
                "ST": "Beijing",
                "OU": "TiDB"
            }
        ]
    }
    EOF
    ```

4. 使用定义的选项生成 CA：

    {{< copyable "shell-regular" >}}

    ``` shell
    cfssl gencert -initca ca-csr.json | cfssljson -bare ca -
    ```

5. 生成 Server 端证书。

    这里需要为每个 DM 集群的组件生成一套 Server 端证书。

    - DM-master Server 端证书

        首先生成默认的 `dm-master-server.json` 文件：

        {{< copyable "shell-regular" >}}

        ``` shell
        cfssl print-defaults csr > dm-master-server.json
        ```

        然后编辑这个文件，修改 `CN`，`hosts` 属性：

        ``` json
        ...
            "CN": "TiDB",
            "hosts": [
              "127.0.0.1",
              "::1",
              "${cluster_name}-dm-master",
              "${cluster_name}-dm-master.${namespace}",
              "${cluster_name}-dm-master.${namespace}.svc",
              "${cluster_name}-dm-master-peer",
              "${cluster_name}-dm-master-peer.${namespace}",
              "${cluster_name}-dm-master-peer.${namespace}.svc",
              "*.${cluster_name}-dm-master-peer",
              "*.${cluster_name}-dm-master-peer.${namespace}",
              "*.${cluster_name}-dm-master-peer.${namespace}.svc"
            ],
        ...
        ```

        其中 `${cluster_name}` 为 DM 集群的名字，`${namespace}` 为 DM 集群部署的命名空间，用户也可以添加自定义 `hosts`。

        最后生成 DM-master Server 端证书：

        {{< copyable "shell-regular" >}}

        ``` shell
        cfssl gencert -ca=ca.pem -ca-key=ca-key.pem -config=ca-config.json -profile=internal dm-master-server.json | cfssljson -bare dm-master-server
        ```

    - DM-worker Server 端证书

        首先生成默认的 `dm-worker-server.json` 文件：

        {{< copyable "shell-regular" >}}

        ``` shell
        cfssl print-defaults csr > dm-worker-server.json
        ```

        然后编辑这个文件，修改 `CN`，`hosts` 属性：

        ``` json
        ...
            "CN": "TiDB",
            "hosts": [
              "127.0.0.1",
              "::1",
              "${cluster_name}-dm-worker",
              "${cluster_name}-dm-worker.${namespace}",
              "${cluster_name}-dm-worker.${namespace}.svc",
              "${cluster_name}-dm-worker-peer",
              "${cluster_name}-dm-worker-peer.${namespace}",
              "${cluster_name}-dm-worker-peer.${namespace}.svc",
              "*.${cluster_name}-dm-worker-peer",
              "*.${cluster_name}-dm-worker-peer.${namespace}",
              "*.${cluster_name}-dm-worker-peer.${namespace}.svc"
            ],
        ...
        ```

        其中 `${cluster_name}` 为集群的名字，`${namespace}` 为 DM 集群部署的命名空间，用户也可以添加自定义 `hosts`。

        最后生成 DM-worker Server 端证书：

        {{< copyable "shell-regular" >}}

        ``` shell
        cfssl gencert -ca=ca.pem -ca-key=ca-key.pem -config=ca-config.json -profile=internal dm-worker-server.json | cfssljson -bare dm-worker-server
        ```

6. 生成 Client 端证书。

    首先生成默认的 `client.json` 文件：

    {{< copyable "shell-regular" >}}

    ``` shell
    cfssl print-defaults csr > client.json
    ```

    然后编辑这个文件，修改 `CN`，`hosts` 属性，`hosts` 可以留空：

    ``` json
    ...
        "CN": "TiDB",
        "hosts": [],
    ...
    ```

    最后生成 Client 端证书：

    ``` shell
    cfssl gencert -ca=ca.pem -ca-key=ca-key.pem -config=ca-config.json -profile=client client.json | cfssljson -bare client
    ```

7. 创建 Kubernetes Secret 对象。

    假设你已经按照上述文档为每个组件创建了一套 Server 端证书，并为各个客户端创建了一套 Client 端证书。通过下面的命令为 DM 集群创建这些 Secret 对象：

    * DM-master 集群证书 Secret：

        {{< copyable "shell-regular" >}}

        ``` shell
        kubectl create secret generic ${cluster_name}-dm-master-cluster-secret --namespace=${namespace} --from-file=tls.crt=dm-master-server.pem --from-file=tls.key=dm-master-server-key.pem --from-file=ca.crt=ca.pem
        ```

    * DM-worker 集群证书 Secret：

        {{< copyable "shell-regular" >}}

        ``` shell
        kubectl create secret generic ${cluster_name}-dm-worker-cluster-secret --namespace=${namespace} --from-file=tls.crt=dm-worker-server.pem --from-file=tls.key=dm-worker-server-key.pem --from-file=ca.crt=ca.pem
        ```

    * Client 证书 Secret：

        {{< copyable "shell-regular" >}}

        ``` shell
        kubectl create secret generic ${cluster_name}-dm-client-secret --namespace=${namespace} --from-file=tls.crt=client.pem --from-file=tls.key=client-key.pem --from-file=ca.crt=ca.pem
        ```

    这里给 DM-master/DM-worker 的 Server 端证书分别创建了一个 Secret 供他们启动时加载使用，另外一套 Client 端证书供他们的客户端连接使用。

#### 使用 `cert-manager` 系统颁发证书

1. 安装 cert-manager。

    请参考官网安装：[cert-manager installation in Kubernetes](https://docs.cert-manager.io/en/release-0.11/getting-started/install/kubernetes.html)。

2. 创建一个 Issuer 用于给 DM 集群颁发证书。

    为了配置 `cert-manager` 颁发证书，必须先创建 Issuer 资源。

    首先创建一个目录保存 `cert-manager` 创建证书所需文件：

    {{< copyable "shell-regular" >}}

    ``` shell
    mkdir -p cert-manager
    cd cert-manager
    ```

    然后创建一个 `dm-cluster-issuer.yaml` 文件，输入以下内容：

    ``` yaml
    apiVersion: cert-manager.io/v1alpha2
    kind: Issuer
    metadata:
      name: ${cluster_name}-selfsigned-ca-issuer
      namespace: ${namespace}
    spec:
      selfSigned: {}
    ---
    apiVersion: cert-manager.io/v1alpha2
    kind: Certificate
    metadata:
      name: ${cluster_name}-ca
      namespace: ${namespace}
    spec:
      secretName: ${cluster_name}-ca-secret
      commonName: "TiDB"
      isCA: true
      duration: 87600h # 10yrs
      renewBefore: 720h # 30d
      issuerRef:
        name: ${cluster_name}-selfsigned-ca-issuer
        kind: Issuer
    ---
    apiVersion: cert-manager.io/v1alpha2
    kind: Issuer
    metadata:
      name: ${cluster_name}-dm-issuer
      namespace: ${namespace}
    spec:
      ca:
        secretName: ${cluster_name}-ca-secret
    ```

    其中 `${cluster_name}` 为集群的名字，上面的文件创建三个对象：

    - 一个 SelfSigned 类型的 Isser 对象（用于生成 CA 类型 Issuer 所需要的 CA 证书）;
    - 一个 Certificate 对象，`isCa` 属性设置为 `true`；
    - 一个可以用于颁发 DM 组件间 TLS 证书的 Issuer。

    最后执行下面的命令进行创建：

    {{< copyable "shell-regular" >}}

    ``` shell
    kubectl apply -f dm-cluster-issuer.yaml
    ```

3. 创建 Server 端证书。

    在 `cert-manager` 中，Certificate 资源表示证书接口，该证书将由上面创建的 Issuer 颁发并保持更新。

    我们需要为每个组件创建一个 Server 端证书，并且为它们的 Client 创建一套公用的 Client 端证书。

    - DM-master 组件的 Server 端证书。

        ``` yaml
        apiVersion: cert-manager.io/v1alpha2
        kind: Certificate
        metadata:
          name: ${cluster_name}-dm-master-cluster-secret
          namespace: ${namespace}
        spec:
          secretName: ${cluster_name}-dm-master-cluster-secret
          duration: 8760h # 365d
          renewBefore: 360h # 15d
          organization:
          - PingCAP
          commonName: "TiDB"
          usages:
            - server auth
            - client auth
          dnsNames:
          - "${cluster_name}-dm-master"
          - "${cluster_name}-dm-master.${namespace}"
          - "${cluster_name}-dm-master.${namespace}.svc"
          - "${cluster_name}-dm-master-peer"
          - "${cluster_name}-dm-master-peer.${namespace}"
          - "${cluster_name}-dm-master-peer.${namespace}.svc"
          - "*.${cluster_name}-dm-master-peer"
          - "*.${cluster_name}-dm-master-peer.${namespace}"
          - "*.${cluster_name}-dm-master-peer.${namespace}.svc"
          ipAddresses:
          - 127.0.0.1
          - ::1
          issuerRef:
            name: ${cluster_name}-dm-issuer
            kind: Issuer
            group: cert-manager.io
        ```

        其中 `${cluster_name}` 为集群的名字：

        - `spec.secretName` 请设置为 `${cluster_name}-dm-master-cluster-secret`；
        - `usages` 请添加上  `server auth` 和 `client auth`；
        - `dnsNames` 需要填写上面 yaml 中的 DNS，根据需要可以填写其他 DNS；
        - `ipAddresses` 需要填写这两个 IP ，根据需要可以填写其他 IP：
            - `127.0.0.1`
            - `::1`
        - `issuerRef` 请填写上面创建的 Issuer；
        - 其他属性请参考 [cert-manager API](https://cert-manager.io/docs/reference/api-docs/#cert-manager.io/v1alpha2.CertificateSpec)。

        创建这个对象以后，`cert-manager` 会生成一个名字为 `${cluster_name}-dm-master-cluster-secret` 的 Secret 对象供 DM 集群的 DM-master 组件使用。

    - DM-worker 组件的 Server 端证书。

        ``` yaml
        apiVersion: cert-manager.io/v1alpha2
        kind: Certificate
        metadata:
          name: ${cluster_name}-dm-worker-cluster-secret
          namespace: ${namespace}
        spec:
          secretName: ${cluster_name}-dm-worker-cluster-secret
          duration: 8760h # 365d
          renewBefore: 360h # 15d
          organization:
          - PingCAP
          commonName: "TiDB"
          usages:
            - server auth
            - client auth
          dnsNames:
          - "${cluster_name}-dm-worker"
          - "${cluster_name}-dm-worker.${namespace}"
          - "${cluster_name}-dm-worker.${namespace}.svc"
          - "${cluster_name}-dm-worker-peer"
          - "${cluster_name}-dm-worker-peer.${namespace}"
          - "${cluster_name}-dm-worker-peer.${namespace}.svc"
          - "*.${cluster_name}-dm-worker-peer"
          - "*.${cluster_name}-dm-worker-peer.${namespace}"
          - "*.${cluster_name}-dm-worker-peer.${namespace}.svc"
          ipAddresses:
          - 127.0.0.1
          - ::1
          issuerRef:
            name: ${cluster_name}-dm-issuer
            kind: Issuer
            group: cert-manager.io
        ```

        其中 `${cluster_name}` 为集群的名字：

        - `spec.secretName` 请设置为 `${cluster_name}-dm-worker-cluster-secret`；
        - `usages` 请添加上  `server auth` 和 `client auth`；
        - `dnsNames` 需要填写上面 yaml 中的 DNS，根据需要可以填写其他 DNS；
        - `ipAddresses` 需要填写这两个 IP ，根据需要可以填写其他 IP：
            - `127.0.0.1`
            - `::1`
        - `issuerRef` 请填写上面创建的 Issuer；
        - 其他属性请参考 [cert-manager API](https://cert-manager.io/docs/reference/api-docs/#cert-manager.io/v1alpha2.CertificateSpec)。

        创建这个对象以后，`cert-manager` 会生成一个名字为 `${cluster_name}-dm-cluster-secret` 的 Secret 对象供 DM 集群的 DM-worker 组件使用。

    - 一套 DM 集群组件的 Client 端证书。

        ``` yaml
        apiVersion: cert-manager.io/v1alpha2
        kind: Certificate
        metadata:
          name: ${cluster_name}-dm-client-secret
          namespace: ${namespace}
        spec:
          secretName: ${cluster_name}-dm-client-secret
          duration: 8760h # 365d
          renewBefore: 360h # 15d
          organization:
          - PingCAP
          commonName: "TiDB"
          usages:
            - client auth
          issuerRef:
            name: ${cluster_name}-dm-issuer
            kind: Issuer
            group: cert-manager.io
        ```

        其中 `${cluster_name}` 为集群的名字：

        - `spec.secretName` 请设置为 `${cluster_name}-dm-client-secret`；
        - `usages` 请添加上  `client auth`；
        - `dnsNames` 和 `ipAddresses` 不需要填写；
        - `issuerRef` 请填写上面创建的 Issuer；
        - 其他属性请参考 [cert-manager API](https://cert-manager.io/docs/reference/api-docs/#cert-manager.io/v1alpha2.CertificateSpec)。

        创建这个对象以后，`cert-manager` 会生成一个名字为 `${cluster_name}-cluster-client-secret` 的 Secret 对象供 DM 组件的 Client 使用。

### 第二步：部署 DM 集群

在部署 DM 集群时，可以开启集群间的 TLS，同时可以设置 `cert-allowed-cn` 配置项，用来验证集群间各组件证书的 CN (Common Name)。

> **注意：**
>
> 目前 DM-master 的 `cert-allowed-cn` 配置项只能设置一个值。因此所有 `Certificate` 对象的 `commonName` 都要设置成同样一个值。

创建 `dm-cluster.yaml` 文件：

```yaml
apiVersion: pingcap.com/v1alpha1
kind: DMCluster
metadata:
  name: ${cluster_name}
  namespace: ${namespace}
spec:
  tlsCluster:
    enabled: true
  version: v2.0.7
  pvReclaimPolicy: Retain
  discovery: {}
  master:
    baseImage: pingcap/dm
    maxFailoverCount: 0
    replicas: 1
    storageSize: "1Gi"
    config:
      cert-allowed-cn:
        - TiDB
  worker:
    baseImage: pingcap/dm
    maxFailoverCount: 0
    replicas: 1
    storageSize: "1Gi"
    config:
      cert-allowed-cn:
        - TiDB
```

然后使用 `kubectl apply -f dm-cluster.yaml` 来创建 DM 集群。

### 第三步：配置 `dmctl` 连接集群

进入 DM-master Pod：

{{< copyable "shell-regular" >}}

``` shell
kubectl exec -it ${cluster_name}-dm-master-0 -n ${namespace} sh
```

使用 `dmctl`：

{{< copyable "shell-regular" >}}

``` shell
cd /var/lib/dm-master-tls
/dmctl --ssl-ca=ca.crt --ssl-cert=tls.crt --ssl-key=tls.key --master-addr 127.0.0.1:8261 list-member
```

## 用 DM 集群同步开启了 MySQL 客户端 TLS 验证的 MySQL/TiDB 数据库

下面部分主要介绍如何配置 DM 同步开启了 MySQL 客户端 TLS 验证的 MySQL/TiDB 数据库。如需了解如何为 TiDB 的 MySQL 客户端开启 TLS，可以参考[为 MySQL 客户端开启 TLS](enable-tls-for-mysql-client.md)

### 第一步：创建各 MySQL 客户端 TLS 的 Kubernetes Secret 对象 

到这里假设你已经部署了开启 MySQL 客户端 TLS 的 MySQL/TiDB 数据库。通过下面的命令为 TiDB 集群创建 Secret 对象：

{{< copyable "shell-regular" >}}

```bash
kubectl create secret generic ${mysql_secret_name1} --namespace=${namespace} --from-file=tls.crt=client.pem --from-file=tls.key=client-key.pem --from-file=ca.crt=ca.pem
kubectl create secret generic ${tidb_secret_name} --namespace=${namespace} --from-file=tls.crt=client.pem --from-file=tls.key=client-key.pem --from-file=ca.crt=ca.pem
```

### 第二步：挂载 Secret 对象到 DM 集群

创建好上下游数据库的 Kubernetes Secret 对象后，我们需要设置 `spec.tlsClientSecretNames` 使得 Secret 对象被挂载到 DM-master/DM-worker 的 Pod。

```yaml
apiVersion: pingcap.com/v1alpha1
kind: DMCluster
metadata:
  name: ${cluster_name}
  namespace: ${namespace}
spec:
  version: v2.0.7
  pvReclaimPolicy: Retain
  discovery: {}
  tlsClientSecretNames:
    - ${mysql_secret_name1}
    - ${tidb_secret_name}
  master:
    ...
```

### 第三步：修改数据源配置与同步任务配置

设置 `spec.tlsClientSecretNames` 选项后，TiDB Operator 会将 Secret 对象 ${secret_name} 挂载到 `/var/lib/source-tls/${secret_name}` 路径。

1. 填写[数据源配置](use-tidb-dm.md#创建数据源) `source1.yaml` 的 `from.security` 选项：

    ``` yaml
    source-id: mysql-replica-01
    relay-dir: /var/lib/dm-worker/relay
    from:
      host: ${mysql_host1}
      user: dm
      password: ""
      port: 3306
      security:
        ssl-ca: /var/lib/source-tls/${mysql_secret_name1}/ca.crt
        ssl-cert: /var/lib/source-tls/${mysql_secret_name1}/tls.crt
        ssl-key: /var/lib/source-tls/${mysql_secret_name1}/tls.key
    ```

2. 填写[同步任务配置](use-tidb-dm.md#配置同步任务) `task.yaml` 的 `target-database.security` 选项：

    ``` yaml
    name: test
    task-mode: all
    is-sharding: false

    target-database:
      host: ${tidb_host}
      port: 4000
      user: "root"
      password: ""
      security:
        ssl-ca: /var/lib/source-tls/${tidb_secret_name}/ca.crt
        ssl-cert: /var/lib/source-tls/${tidb_secret_name}/tls.crt
        ssl-key: /var/lib/source-tls/${tidb_secret_name}/tls.key

    mysql-instances:
    - source-id: "replica-01"
      loader-config-name: "global"

    loaders:
      global:
        dir: "/var/lib/dm-worker/dumped_data"
    ```

### 第四步：启动同步任务

参考[启动同步任务](use-tidb-dm.md#启动查询停止同步任务)。
